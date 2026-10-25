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
func (r *repository) List(_ context.Context, cursor string) (domain.Page, error) {
	r.cursor = cursor
	return domain.Page{Links: []domain.Link{r.link}, NextCursor: "next"}, r.err
}
func (r *repository) SetEnabled(_ context.Context, _ string, enabled bool) (domain.Link, error) {
	r.link.Enabled = enabled
	return r.link, r.err
}

type contextKey struct{}

func event(method, path, route, body string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		Version: "2.0", RouteKey: method + " " + route, RawPath: path, Body: body,
		Headers:        map[string]string{"content-type": "application/json"},
		PathParameters: map[string]string{"code": strings.TrimPrefix(strings.TrimPrefix(path, "/api/links/"), "/r/")},
		RequestContext: events.APIGatewayV2HTTPRequestContext{RequestID: "gateway-test", HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: method}},
	}
}

func TestHTTPV2(t *testing.T) {
	past := time.Now().Add(-time.Hour)
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
		{name: "disable", group: "manage", method: "PATCH", path: "/api/links/test-link", route: "/api/links/{code}", body: `{"enabled":false}`, status: 200},
		{name: "missing enabled", group: "manage", method: "PATCH", path: "/api/links/test-link", route: "/api/links/{code}", body: `{}`, status: 400},
		{name: "health", group: "manage", method: "GET", path: "/health", route: "/health", status: 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			link := domain.Link{Code: "test-link", URL: "https://example.com/path?q=1#here", Enabled: !tc.disabled, CreatedAt: past}
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
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			httpapi.New(application.New(&repository{link: link, err: tc.err}), "http://localhost:3000").ServeHTTP(local, req)
			if local.Code != response.StatusCode || local.Header().Get("Location") != response.Headers["Location"] {
				t.Fatalf("local/native contract differs: local=%d native=%d", local.Code, response.StatusCode)
			}
		})
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
