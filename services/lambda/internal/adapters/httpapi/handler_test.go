package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"shortlink/internal/application"
	"shortlink/internal/domain"
)

type repository struct {
	link domain.Link
	err  error
}

func (r *repository) Create(_ context.Context, l domain.Link) error        { r.link = l; return r.err }
func (r *repository) Get(_ context.Context, _ string) (domain.Link, error) { return r.link, r.err }
func (r *repository) List(_ context.Context, _, _ string) (domain.Page, error) {
	return domain.Page{Links: []domain.Link{r.link}}, r.err
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
	r.link.UpdatedAt = patch.UpdatedAt
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

func makeJWT(sub, scope, tokenUse string, exp int64) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims := map[string]any{
		"sub":       sub,
		"scope":     scope,
		"token_use": tokenUse,
		"exp":       exp,
	}
	rawClaims, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(rawClaims)
	return header + "." + payload + ".signature"
}

func validBearer(sub string) string {
	return "Bearer " + makeJWT(sub, "openid email shortly-api/links.read shortly-api/links.write", "access", time.Now().Add(time.Hour).Unix())
}

func TestHTTPContract(t *testing.T) {
	now := time.Now().UTC()
	repo := &repository{link: domain.Link{Code: "test1234", OwnerID: "user-123", URL: "https://example.com/path?q=1", Enabled: true, Version: 1, CreatedAt: now, UpdatedAt: now}}
	h := New(application.New(repo), "http://localhost:3000")
	for _, tc := range []struct {
		method, path, body, auth string
		status                   int
	}{
		{"GET", "/health", "", "", 200},
		{"POST", "/api/links", `{"url":"https://example.com","alias":"hello"}`, validBearer("user-123"), 201},
		{"GET", "/api/links/hello", "", validBearer("user-123"), 200},
		{"GET", "/api/links", "", validBearer("user-123"), 200},
		{"PATCH", "/api/links/hello", `{"version":1,"enabled":false}`, validBearer("user-123"), 200},
		{"GET", "/r/hello", "", "", 410},
		{"PATCH", "/api/links/hello", `{"version":2,"enabled":true,"url":"https://example.com/v2"}`, validBearer("user-123"), 200},
		{"GET", "/r/hello", "", "", 302},
		{"PATCH", "/api/links/hello", `{"version":1,"enabled":false}`, validBearer("user-123"), 409},
		{"GET", "/api/links/hello", "", validBearer("other-user"), 404},
		{"PATCH", "/api/links/hello", `{"version":3,"enabled":false}`, validBearer("other-user"), 404},
		{"DELETE", "/api/links/hello?version=3", "", validBearer("other-user"), 404},
		{"DELETE", "/api/links/hello?version=1", "", validBearer("user-123"), 409},
		{"DELETE", "/api/links/hello?version=3", "", validBearer("user-123"), 204},
		{"GET", "/r/hello", "", "", 410},
		{"POST", "/api/links", `{"url":"https://example.com"}`, "", 401},
		{"GET", "/api/links", "", "Bearer " + makeJWT("user-123", "shortly-api/links.read", "access", time.Now().Add(-time.Minute).Unix()), 401},
		{"GET", "/api/links", "", "Bearer " + makeJWT("user-123", "shortly-api/links.read", "id", time.Now().Add(time.Hour).Unix()), 403},
		{"POST", "/api/links", `{"url":"https://example.com"}`, "Bearer " + makeJWT("user-123", "shortly-api/links.read", "access", time.Now().Add(time.Hour).Unix()), 403},
		{"POST", "/api/links", `{"url":"javascript:alert(1)"}`, validBearer("user-123"), 400},
		{"POST", "/api/links", `{"url":"https://example.com","unknown":true}`, validBearer("user-123"), 400},
		{"POST", "/api/links", `{"url":"https://example.com"} {}`, validBearer("user-123"), 400},
		{"POST", "/api/links", `{`, validBearer("user-123"), 400},
		{"PATCH", "/api/links/hello", `{}`, validBearer("user-123"), 400},
		{"PATCH", "/api/links/hello", `{"version":1,"enabled":null}`, validBearer("user-123"), 400},
		{"DELETE", "/api/links/hello?version=0", "", validBearer("user-123"), 400},
		{"POST", "/api/links", `{"url":"https://example.com","expiresAt":"yesterday"}`, validBearer("user-123"), 400},
		{"GET", "/api/links/a!b", "", validBearer("user-123"), 400},
		{"PUT", "/api/links/hello", "", validBearer("user-123"), 405},
		{"POST", "/r/hello", "", "", 405},
		{"GET", "/unknown", "", "", 404},
	} {
		t.Run(tc.method+" "+tc.path+" "+tc.body, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://127.0.0.1:8080"+tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", w.Code, tc.status, w.Body.String())
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Error("missing no-store")
			}
			if tc.status >= 400 {
				var envelope struct {
					Error struct{ Code, Message string }
				}
				if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil || envelope.Error.Code == "" || envelope.Error.Message == "" {
					t.Fatalf("bad error envelope: %s", w.Body.String())
				}
			}
			if tc.status == 302 && w.Header().Get("Location") != repo.link.URL {
				t.Fatal("wrong redirect target")
			}
		})
	}
}

func TestHTTPErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrNotFound, 404, "not_found"},
		{domain.ErrConflict, 409, "conflict"},
		{domain.ErrVersionConflict, 409, "version_conflict"},
		{domain.ErrGone, 410, "gone"},
		{domain.ErrUnavailable, 503, "unavailable"},
		{errors.New("secret backend details"), 500, "internal_error"},
	} {
		h := New(application.New(&repository{err: tc.err}), "http://localhost:3000")
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/links/hello", nil)
		req.Header.Set("Authorization", validBearer("user-123"))
		h.ServeHTTP(w, req)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) || strings.Contains(w.Body.String(), "secret backend details") {
			t.Fatalf("%v: %d %s", tc.err, w.Code, w.Body.String())
		}
	}
}

func TestCORSAndBodyLimits(t *testing.T) {
	h := New(application.New(&repository{}), "http://localhost:3000")
	for _, tc := range []struct {
		method, host, origin, contentType, body string
		status                                  int
	}{
		{"OPTIONS", "127.0.0.1:8080", "http://localhost:3000", "", "", 204},
		{"GET", "localhost:8080", "", "", "", 200},
		{"GET", "[::1]:8080", "", "", "", 200},
		{"GET", "[::1]", "", "", "", 200},
		{"POST", "127.0.0.1:8080", "https://untrusted.example", "application/json", `{}`, 403},
		{"GET", "127.0.0.1:8080", "null", "", "", 403},
		{"GET", "attacker.example:8080", "", "", "", 403},
		{"GET", "attacker.example", "", "", "", 403},
		{"GET", "localhost:99999", "", "", "", 403},
		{"GET", "::1", "", "", "", 403},
		{"POST", "127.0.0.1:8080", "", "text/plain", `{}`, 415},
		{"POST", "127.0.0.1:8080", "", "application/json", `{"url":"` + strings.Repeat("a", 17000) + `"}`, 413},
	} {
		req := httptest.NewRequest(tc.method, "http://127.0.0.1:8080/api/links", strings.NewReader(tc.body))
		req.Host = tc.host
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Content-Type", tc.contentType)
		req.Header.Set("Authorization", validBearer("user-123"))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("%+v: status=%d body=%s", tc, w.Code, w.Body.String())
		}
		if tc.origin == "http://localhost:3000" && w.Header().Get("Access-Control-Allow-Origin") != tc.origin {
			t.Fatal("missing CORS header")
		}
		if tc.status == 403 && w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("untrusted CORS allowed")
		}
	}
}

func TestLocalOAuth2PKCEFlow(t *testing.T) {
	h := New(application.New(&repository{}), "http://localhost:3000")
	verifier := "pkce-verifier-string-1234567890-abcdef"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	authReq := httptest.NewRequest("GET", "http://127.0.0.1:8080/oauth2-local/authorize?response_type=code&client_id=local-web-client&redirect_uri="+url.QueryEscape("http://localhost:3000/auth/callback/")+"&state=state-123&code_challenge="+challenge+"&code_challenge_method=S256&login_hint=user-a", nil)
	authRes := httptest.NewRecorder()
	h.ServeHTTP(authRes, authReq)
	if authRes.Code != 302 {
		t.Fatalf("authorize status=%d", authRes.Code)
	}
	loc, err := url.Parse(authRes.Header().Get("Location"))
	if err != nil || loc.Query().Get("state") != "state-123" || loc.Query().Get("code") == "" {
		t.Fatalf("invalid redirect location %q err=%v", authRes.Header().Get("Location"), err)
	}
	code := loc.Query().Get("code")

	// Wrong verifier is rejected.
	badTokenReq := httptest.NewRequest("POST", "http://127.0.0.1:8080/oauth2-local/token", strings.NewReader("grant_type=authorization_code&code="+url.QueryEscape(code)+"&code_verifier=wrong"))
	badTokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badTokenRes := httptest.NewRecorder()
	h.ServeHTTP(badTokenRes, badTokenReq)
	if badTokenRes.Code != 400 {
		t.Fatalf("expected 400 for bad verifier, got %d", badTokenRes.Code)
	}

	// Re-authorize and exchange with matching PKCE verifier.
	authRes2 := httptest.NewRecorder()
	h.ServeHTTP(authRes2, authReq)
	loc2, _ := url.Parse(authRes2.Header().Get("Location"))
	code2 := loc2.Query().Get("code")

	tokenReq := httptest.NewRequest("POST", "http://127.0.0.1:8080/oauth2-local/token", strings.NewReader("grant_type=authorization_code&code="+url.QueryEscape(code2)+"&code_verifier="+url.QueryEscape(verifier)))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenRes := httptest.NewRecorder()
	h.ServeHTTP(tokenRes, tokenReq)
	if tokenRes.Code != 200 {
		t.Fatalf("token status=%d body=%s", tokenRes.Code, tokenRes.Body.String())
	}
	var tokens map[string]any
	if err := json.Unmarshal(tokenRes.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	accessToken, _ := tokens["access_token"].(string)
	refreshToken, _ := tokens["refresh_token"].(string)
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("missing tokens: %+v", tokens)
	}

	// Refresh token grant.
	refreshReq := httptest.NewRequest("POST", "http://127.0.0.1:8080/oauth2-local/token", strings.NewReader("grant_type=refresh_token&refresh_token="+url.QueryEscape(refreshToken)))
	refreshReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	refreshRes := httptest.NewRecorder()
	h.ServeHTTP(refreshRes, refreshReq)
	if refreshRes.Code != 200 {
		t.Fatalf("refresh status=%d body=%s", refreshRes.Code, refreshRes.Body.String())
	}

	// Revoke endpoint.
	revokeReq := httptest.NewRequest("POST", "http://127.0.0.1:8080/oauth2-local/revoke", strings.NewReader("token="+url.QueryEscape(refreshToken)))
	revokeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	revokeRes := httptest.NewRecorder()
	h.ServeHTTP(revokeRes, revokeReq)
	if revokeRes.Code != 200 {
		t.Fatalf("revoke status=%d", revokeRes.Code)
	}
}
