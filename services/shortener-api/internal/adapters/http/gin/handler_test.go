package ginhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/adapters/repository/memory"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/application/service"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

type handlerClock struct {
	now time.Time
}

func (c handlerClock) Now() time.Time {
	return c.now
}

type handlerGenerator struct {
	code string
}

type panickingApplication struct{}

func (panickingApplication) Create(context.Context, service.CreateLinkInput) (service.LinkView, error) {
	panic("sensitive internal detail")
}

func (panickingApplication) Resolve(context.Context, string) (domain.Link, error) {
	panic("sensitive internal detail")
}

func (panickingApplication) GetMetadata(context.Context, string) (service.LinkView, error) {
	panic("sensitive internal detail")
}

func (g handlerGenerator) Generate(context.Context) (string, error) {
	return g.code, nil
}

func newTestRouter() *ginTestRouter {
	repository := memory.NewLinkRepository()
	application := service.NewLinkService(
		repository,
		handlerGenerator{code: "abc1234"},
		handlerClock{now: time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)},
	)
	handler := NewHandler(application, "https://npt-shortenlink.dev")
	return &ginTestRouter{handler: http.Handler(NewRouter(handler, []string{"http://localhost:3000"}))}
}

type ginTestRouter struct {
	handler http.Handler
}

func (r *ginTestRouter) request(method, path string, body []byte, contentType string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	r.handler.ServeHTTP(response, request)
	return response
}

func TestHealth(t *testing.T) {
	response := newTestRouter().request(http.MethodGet, "/healthz", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
}

func TestCreateLink(t *testing.T) {
	payload := []byte(`{"url":"https://example.com/docs","custom_alias":"my-docs","expires_in_days":30}`)
	response := newTestRouter().request(http.MethodPost, "/api/v1/links", payload, "application/json")
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", response.Code, response.Body.String())
	}

	var envelope linkEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Data.Code != "my-docs" {
		t.Fatalf("code = %q, want my-docs", envelope.Data.Code)
	}
	if envelope.Data.ShortURL != "https://npt-shortenlink.dev/link/my-docs" {
		t.Fatalf("short URL = %q", envelope.Data.ShortURL)
	}
}

func TestCreateRejectsInvalidPayload(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{name: "invalid URL", body: `{"url":"not-a-url"}`, contentType: "application/json", wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"url":"https://example.com","unknown":true}`, contentType: "application/json", wantStatus: http.StatusBadRequest},
		{name: "wrong media type", body: `{"url":"https://example.com"}`, contentType: "text/plain", wantStatus: http.StatusUnsupportedMediaType},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := newTestRouter().request(
				http.MethodPost,
				"/api/v1/links",
				[]byte(test.body),
				test.contentType,
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

func TestResolveRedirect(t *testing.T) {
	router := newTestRouter()
	payload := []byte(`{"url":"https://example.com/docs","custom_alias":"my-docs"}`)
	created := router.request(http.MethodPost, "/api/v1/links", payload, "application/json")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", created.Code)
	}

	resolved := router.request(http.MethodGet, "/link/my-docs", nil, "")
	if resolved.Code != http.StatusFound {
		t.Fatalf("resolve status = %d, want 302", resolved.Code)
	}
	if location := resolved.Header().Get("Location"); location != "https://example.com/docs" {
		t.Fatalf("Location = %q", location)
	}
}

func TestLinkMetadata(t *testing.T) {
	router := newTestRouter()
	payload := []byte(`{"url":"https://example.com/docs","custom_alias":"my-docs","expires_in_days":30}`)
	created := router.request(http.MethodPost, "/api/v1/links", payload, "application/json")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", created.Code)
	}

	response := router.request(http.MethodGet, "/api/v1/links/my-docs", nil, "")
	if response.Code != http.StatusOK {
		t.Fatalf("metadata status = %d, want 200", response.Code)
	}
	var envelope linkEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Data.Status != "active" || envelope.Data.TargetURL != "https://example.com/docs" {
		t.Fatalf("metadata = %#v", envelope.Data)
	}
}

func TestSecurityBoundaries(t *testing.T) {
	router := newTestRouter()

	deniedRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	deniedRequest.Header.Set("Origin", "https://evil.example")
	denied := httptest.NewRecorder()
	router.handler.ServeHTTP(denied, deniedRequest)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("denied origin status = %d, want 403", denied.Code)
	}

	allowedRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	allowedRequest.Header.Set("Origin", "http://localhost:3000")
	allowedRequest.Header.Set(requestIDHeader, "client-request-123")
	allowed := httptest.NewRecorder()
	router.handler.ServeHTTP(allowed, allowedRequest)
	if allowed.Code != http.StatusOK {
		t.Fatalf("allowed origin status = %d, want 200", allowed.Code)
	}
	if value := allowed.Header().Get("Access-Control-Allow-Origin"); value != "http://localhost:3000" {
		t.Fatalf("allow origin = %q", value)
	}
	if value := allowed.Header().Get(requestIDHeader); value != "client-request-123" {
		t.Fatalf("request ID = %q", value)
	}

	preflightRequest := httptest.NewRequest(http.MethodOptions, "/api/v1/links", nil)
	preflightRequest.Header.Set("Origin", "http://localhost:3000")
	preflight := httptest.NewRecorder()
	router.handler.ServeHTTP(preflight, preflightRequest)
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", preflight.Code)
	}

	oversized := strings.Repeat("x", int(maxCreateLinkRequestBytes)+1)
	response := router.request(http.MethodPost, "/api/v1/links", []byte(oversized), "application/json")
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status = %d, want 413", response.Code)
	}
}

func TestPanicRecoveryReturnsSafeJSON(t *testing.T) {
	handler := NewHandler(panickingApplication{}, "https://npt-shortenlink.dev")
	router := NewRouter(handler, []string{"http://localhost:3000"})
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/links",
		bytes.NewBufferString(`{"url":"https://example.com"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(requestIDHeader, "panic-request-123")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if response.Header().Get(requestIDHeader) != "panic-request-123" {
		t.Fatalf("request ID = %q", response.Header().Get(requestIDHeader))
	}
	if strings.Contains(response.Body.String(), "sensitive internal detail") {
		t.Fatalf("response leaked panic detail: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("response body = %s", response.Body.String())
	}
}
