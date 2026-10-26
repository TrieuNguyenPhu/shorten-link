package lambdaapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"shortlink/internal/adapters/httpapi"
	"shortlink/internal/application"
	"shortlink/internal/domain"
)

type repository struct {
	link         domain.Link
	err          error
	contextValue any
	cursor       string
}

func (r *repository) Create(_ context.Context, link domain.Link) error { r.link = link; return r.err }
func (r *repository) Get(ctx context.Context, _ string) (domain.Link, error) {
	r.contextValue = ctx.Value(contextKey{})
	return r.link, r.err
}
func (r *repository) List(_ context.Context, _, cursor string) (domain.Page, error) {
	r.cursor = cursor
	return domain.Page{Links: []domain.Link{r.link}, NextCursor: "next"}, r.err
}
func (r *repository) Update(_ context.Context, ownerID, _ string, patch domain.LinkPatch) (domain.Link, error) {
	if r.err != nil {
		return domain.Link{}, r.err
	}
	if r.link.OwnerID != ownerID {
		return domain.Link{}, domain.ErrNotFound
	}
	if r.link.Version != patch.Version {
		return domain.Link{}, domain.ErrVersionConflict
	}
	if patch.URL != nil {
		r.link.URL = *patch.URL
	}
	if patch.Enabled != nil {
		r.link.Enabled = *patch.Enabled
	}
	if patch.ExpiresAtSet {
		r.link.ExpiresAt = patch.ExpiresAt
	}
	r.link.Version++
	r.link.UpdatedAt = r.link.CreatedAt.Add(time.Minute)
	return r.link, nil
}
func (r *repository) Delete(_ context.Context, ownerID, _ string, version int64, _ time.Time) error {
	if r.err != nil {
		return r.err
	}
	if r.link.OwnerID != ownerID {
		return domain.ErrNotFound
	}
	if version > 0 && r.link.Version != version {
		return domain.ErrVersionConflict
	}
	r.link.Deleted = true
	return nil
}

type contextKey struct{}

func event(method, path, route, body string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		Version: "2.0", RouteKey: method + " " + route, RawPath: path, Body: body,
		Headers:        map[string]string{"content-type": "application/json"},
		PathParameters: map[string]string{"code": strings.TrimPrefix(strings.TrimPrefix(path, "/api/links/"), "/r/")},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "gateway-test",
			HTTP:      events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: method},
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
					Claims: map[string]string{"sub": "user-123", "token_use": "access"},
					Scopes: []string{scopeRead, scopeWrite},
				},
			},
		},
	}
}

func makeBearer(sub string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	rawClaims, _ := json.Marshal(map[string]any{
		"sub":       sub,
		"scope":     "openid email shortly-api/links.read shortly-api/links.write",
		"token_use": "access",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	payload := base64.RawURLEncoding.EncodeToString(rawClaims)
	return "Bearer " + header + "." + payload + ".signature"
}

func TestHTTPV2(t *testing.T) {
	past := time.Now().Add(-time.Hour).UTC()
	for _, tc := range []struct {
		name, group, method, path, route, body string
		status                                 int
		err                                    error
		disabled, expired                      bool
	}{
		{name: "create", group: "create", method: "POST", path: "/api/links", route: "/api/links", body: `{"url":"https://example.com","alias":"fixed-link"}`, status: 201},
		{name: "invalid URL", group: "create", method: "POST", path: "/api/links", route: "/api/links", body: `{"url":"https://example.com:99999"}`, status: 400},
		{name: "unknown JSON", group: "create", method: "POST", path: "/api/links", route: "/api/links", body: `{"url":"https://example.com","extra":true}`, status: 400},
		{name: "trailing JSON", group: "create", method: "POST", path: "/api/links", route: "/api/links", body: `{} {}`, status: 400},
		{name: "collision", group: "create", method: "POST", path: "/api/links", route: "/api/links", body: `{"url":"https://example.com","alias":"fixed-link"}`, status: 409, err: domain.ErrConflict},
		{name: "redirect", group: "resolve", method: "GET", path: "/r/test-link", route: "/r/{code}", status: 302},
		{name: "head redirect", group: "resolve", method: "HEAD", path: "/r/test-link", route: "/r/{code}", status: 302},
		{name: "disabled", group: "resolve", method: "GET", path: "/r/test-link", route: "/r/{code}", status: 410, disabled: true},
		{name: "expired", group: "resolve", method: "GET", path: "/r/test-link", route: "/r/{code}", status: 410, expired: true},
		{name: "missing", group: "resolve", method: "GET", path: "/r/missing", route: "/r/{code}", status: 404, err: domain.ErrNotFound},
		{name: "invalid code", group: "resolve", method: "GET", path: "/r/ab", route: "/r/{code}", status: 400},
		{name: "database failure", group: "resolve", method: "GET", path: "/r/test-link", route: "/r/{code}", status: 500, err: errors.New("private database details")},
		{name: "metadata", group: "manage", method: "GET", path: "/api/links/test-link", route: "/api/links/{code}", status: 200},
		{name: "list", group: "manage", method: "GET", path: "/api/links", route: "/api/links", status: 200},
		{name: "disable", group: "manage", method: "PATCH", path: "/api/links/test-link", route: "/api/links/{code}", body: `{"version":1,"enabled":false}`, status: 200},
		{name: "edit url", group: "manage", method: "PATCH", path: "/api/links/test-link", route: "/api/links/{code}", body: `{"version":1,"url":"https://example.com/updated"}`, status: 200},
		{name: "version conflict", group: "manage", method: "PATCH", path: "/api/links/test-link", route: "/api/links/{code}", body: `{"version":2,"enabled":false}`, status: 409},
		{name: "delete", group: "manage", method: "DELETE", path: "/api/links/test-link", route: "/api/links/{code}", status: 204},
		{name: "missing fields", group: "manage", method: "PATCH", path: "/api/links/test-link", route: "/api/links/{code}", body: `{"version":1}`, status: 400},
		{name: "health", group: "manage", method: "GET", path: "/health", route: "/health", status: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			link := domain.Link{Code: "test-link", OwnerID: "user-123", URL: "https://example.com/path?q=1#here", Enabled: !tc.disabled, Version: 1, CreatedAt: past, UpdatedAt: past}
			if tc.expired {
				link.ExpiresAt = &past
			}
			repo := &repository{link: link, err: tc.err}
			svc := application.New(repo)
			h := New(svc)
			request := event(tc.method, tc.path, tc.route, tc.body)
			ctx := context.WithValue(context.Background(), contextKey{}, "passed-through")
			invoke := h.Manage
			if tc.group == "create" {
				invoke = h.Create
			}
			if tc.group == "resolve" {
				invoke = h.Resolve
			}
			response, err := invoke(ctx, request)
			if err != nil || response.StatusCode != tc.status {
				t.Fatalf("status=%d err=%v body=%s", response.StatusCode, err, response.Body)
			}
			if response.Headers["Cache-Control"] != "no-store" {
				t.Fatal("missing no-store")
			}
			if tc.status >= 400 {
				var envelope struct{ Error struct{ Code string } }
				if json.Unmarshal([]byte(response.Body), &envelope) != nil || envelope.Error.Code == "" || strings.Contains(response.Body, "private database") {
					t.Fatalf("invalid error: %s", response.Body)
				}
			}
			if tc.method == "HEAD" && response.Body != "" {
				t.Fatal("HEAD returned a body")
			}
			if tc.status == 302 && response.Headers["Location"] != link.URL {
				t.Fatal("wrong redirect")
			}
			if tc.group == "resolve" && tc.status != 400 && repo.contextValue != "passed-through" {
				t.Fatal("lost request context")
			}
			// Compare the native transport's public contract with the local HTTP adapter.
			local := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "http://127.0.0.1:8080"+tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", makeBearer("user-123"))
			httpapi.New(application.New(&repository{link: link, err: tc.err}), "http://localhost:3000").ServeHTTP(local, req)
			if local.Code != response.StatusCode || local.Header().Get("Location") != response.Headers["Location"] {
				t.Fatalf("local/native contract differs: local=%d native=%d", local.Code, response.StatusCode)
			}
			if tc.path == "/api/links/test-link" && tc.status == 200 {
				var fields map[string]any
				if err := json.Unmarshal([]byte(response.Body), &fields); err != nil || fields["code"] == nil || fields["url"] == nil || fields["createdAt"] == nil || fields["version"] == nil {
					t.Fatalf("unexpected link JSON contract: %s", response.Body)
				}
				if strings.TrimSpace(local.Body.String()) != response.Body {
					t.Fatalf("local/native JSON body differs: local=%s native=%s", local.Body.String(), response.Body)
				}
			}
		})
	}
}

func TestHTTPV2AuthAndOwnershipEnforcement(t *testing.T) {
	past := time.Now().Add(-time.Hour).UTC()
	link := domain.Link{Code: "test-link", OwnerID: "user-123", URL: "https://example.com", Enabled: true, Version: 1, CreatedAt: past, UpdatedAt: past}
	h := New(application.New(&repository{link: link}))

	// Missing authorizer claims -> 401
	unauth := event("GET", "/api/links", "/api/links", "")
	unauth.RequestContext.Authorizer = nil
	if resp, _ := h.Manage(context.Background(), unauth); resp.StatusCode != 401 {
		t.Fatalf("expected 401 for missing authorizer, got %d", resp.StatusCode)
	}

	// ID token instead of access token -> 403
	idTokenReq := event("GET", "/api/links", "/api/links", "")
	idTokenReq.RequestContext.Authorizer.JWT.Claims["token_use"] = "id"
	if resp, _ := h.Manage(context.Background(), idTokenReq); resp.StatusCode != 403 {
		t.Fatalf("expected 403 for ID token, got %d", resp.StatusCode)
	}

	// Missing write scope on POST -> 403
	readOnlyReq := event("POST", "/api/links", "/api/links", `{"url":"https://example.com"}`)
	readOnlyReq.RequestContext.Authorizer.JWT.Scopes = []string{scopeRead}
	if resp, _ := h.Create(context.Background(), readOnlyReq); resp.StatusCode != 403 {
		t.Fatalf("expected 403 for missing write scope, got %d", resp.StatusCode)
	}

	// Empty scopes -> 403
	emptyScopeReq := event("GET", "/api/links", "/api/links", "")
	emptyScopeReq.RequestContext.Authorizer.JWT.Scopes = nil
	if resp, _ := h.Manage(context.Background(), emptyScopeReq); resp.StatusCode != 403 {
		t.Fatalf("expected 403 for empty scopes, got %d", resp.StatusCode)
	}

	// Cross-user access -> 404
	otherUserReq := event("GET", "/api/links/test-link", "/api/links/{code}", "")
	otherUserReq.RequestContext.Authorizer.JWT.Claims["sub"] = "user-999"
	if resp, _ := h.Manage(context.Background(), otherUserReq); resp.StatusCode != 404 {
		t.Fatalf("expected 404 for cross-user get, got %d", resp.StatusCode)
	}
}

func TestHTTPV2BodyAndRouteValidation(t *testing.T) {
	h := New(application.New(&repository{}))
	for _, tc := range []struct {
		name, body, contentType string
		encoded                 bool
		version                 string
		want                    int
	}{
		{"base64", base64.StdEncoding.EncodeToString([]byte(`{"url":"https://example.com"}`)), "application/json", true, "2.0", 201},
		{"bad base64", "!!!", "application/json", true, "2.0", 400},
		{"too large", strings.Repeat("x", 16385), "application/json", false, "2.0", 413},
		{"base64 too large", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 16385))), "application/json", true, "2.0", 413},
		{"wrong type", "{}", "text/plain", false, "2.0", 415},
		{"wrong payload", "{}", "application/json", false, "1.0", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := event("POST", "/api/links", "/api/links", tc.body)
			req.Version, req.IsBase64Encoded = tc.version, tc.encoded
			req.Headers = map[string]string{"Content-Type": tc.contentType}
			resp, err := h.Create(context.Background(), req)
			if err != nil || resp.StatusCode != tc.want {
				t.Fatalf("status=%d err=%v body=%s", resp.StatusCode, err, resp.Body)
			}
		})
	}
	resp, _ := h.Resolve(context.Background(), event("GET", "/api/links", "/api/links", ""))
	if resp.StatusCode != 404 {
		t.Fatal("resolve function exposed management route")
	}
}

func TestHTTPV2QueryAndMissingPathParameter(t *testing.T) {
	repo := &repository{}
	h := New(application.New(repo))
	req := event("GET", "/api/links", "/api/links", "")
	req.RawQueryString = "cursor=opaque%2B%2F%3D"
	resp, err := h.Manage(context.Background(), req)
	if err != nil || resp.StatusCode != 200 || repo.cursor != "opaque+/=" {
		t.Fatalf("cursor was not decoded correctly: %q", repo.cursor)
	}
	req.RawQueryString = "cursor=%ZZ"
	resp, _ = h.Manage(context.Background(), req)
	if resp.StatusCode != 400 {
		t.Fatal("malformed query accepted")
	}
	req = event("DELETE", "/api/links/test-link", "/api/links/{code}", "")
	req.RawQueryString = "version=bad"
	resp, _ = h.Manage(context.Background(), req)
	if resp.StatusCode != 400 {
		t.Fatal("malformed version query accepted")
	}
	req = event("HEAD", "/health", "/health", "")
	resp, _ = h.Manage(context.Background(), req)
	if resp.StatusCode != 200 || resp.Body != "" {
		t.Fatal("health HEAD contract changed")
	}
	req = event("GET", "/r/abc", "/r/{code}", "")
	req.PathParameters = nil
	resp, _ = h.Resolve(context.Background(), req)
	if resp.StatusCode != 400 {
		t.Fatal("missing code accepted")
	}
}
