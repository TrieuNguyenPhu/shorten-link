package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
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
func (r *repository) List(_ context.Context, _ string) (domain.Page, error) {
	return domain.Page{Links: []domain.Link{r.link}}, r.err
}
func (r *repository) SetEnabled(_ context.Context, _ string, enabled bool) (domain.Link, error) {
	r.link.Enabled = enabled
	return r.link, r.err
}

func TestHTTPContract(t *testing.T) {
	repo := &repository{link: domain.Link{Code: "test1234", URL: "https://example.com/path?q=1", Enabled: true, CreatedAt: time.Now().UTC()}}
	h := New(application.New(repo), "http://localhost:3000")
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/health", "", 200},
		{"POST", "/api/links", `{"url":"https://example.com","alias":"hello"}`, 201},
		{"GET", "/api/links/hello", "", 200},
		{"GET", "/api/links", "", 200},
		{"PATCH", "/api/links/hello", `{"enabled":false}`, 200},
		{"GET", "/r/hello", "", 410},
		{"PATCH", "/api/links/hello", `{"enabled":true}`, 200},
		{"GET", "/r/hello", "", 302},
		{"POST", "/api/links", `{"url":"javascript:alert(1)"}`, 400},
		{"POST", "/api/links", `{"url":"https://example.com","unknown":true}`, 400},
		{"POST", "/api/links", `{"url":"https://example.com"} {}`, 400},
		{"POST", "/api/links", `{`, 400},
		{"PATCH", "/api/links/hello", `{}`, 400},
		{"PATCH", "/api/links/hello", `{"enabled":null}`, 400},
		{"POST", "/api/links", `{"url":"https://example.com","expiresAt":"yesterday"}`, 400},
		{"GET", "/api/links/a!b", "", 400},
		{"DELETE", "/api/links/hello", "", 405},
		{"POST", "/r/hello", "", 405},
		{"GET", "/unknown", "", 404},
	} {
		t.Run(tc.method+" "+tc.path+" "+tc.body, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://127.0.0.1:8080"+tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
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
		{domain.ErrNotFound, 404, "not_found"}, {domain.ErrConflict, 409, "conflict"}, {domain.ErrGone, 410, "gone"}, {domain.ErrUnavailable, 503, "unavailable"}, {errors.New("secret backend details"), 500, "internal_error"},
	} {
		h := New(application.New(&repository{err: tc.err}), "http://localhost:3000")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:8080/api/links/hello", nil))
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
