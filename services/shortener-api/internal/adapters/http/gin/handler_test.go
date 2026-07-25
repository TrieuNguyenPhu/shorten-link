package ginhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

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

func (g handlerGenerator) Generate(context.Context) (string, error) {
	return g.code, nil
}

func (panickingApplication) Create(context.Context, service.CreateLinkInput) (service.LinkView, error) {
	panic("sensitive panic detail")
}

func (panickingApplication) Resolve(context.Context, string) (domain.Link, error) {
	return domain.Link{}, domain.ErrLinkNotFound
}

func (panickingApplication) GetMetadata(context.Context, string) (service.LinkView, error) {
	return service.LinkView{}, domain.ErrLinkNotFound
}

func newTestRouter(now time.Time) (*gin.Engine, *memory.LinkRepository) {
	gin.SetMode(gin.TestMode)
	repository := memory.NewLinkRepository()
	application := service.NewLinkService(repository, handlerGenerator{code: "abc1234"}, handlerClock{now: now})
	handler := NewHandler(application, "https://npt-shortenlink.dev/")
	return NewRouter(handler, []string{"http://localhost:3000", "https://npt-shortenlink.dev"}), repository
}

func TestHealth(t *testing.T) {
	router, _ := newTestRouter(time.Now())
	response := performRequest(router, http.MethodGet, "/healthz", nil, "")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if response.Body.String() != `{"status":"ok"}` {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestCreateMetadataAndRedirect(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	router, _ := newTestRouter(now)
	body := []byte(`{"url":"https://example.com/guide","expires_in_days":7}`)

	created := performRequest(router, http.MethodPost, "/api/v1/links", body, "http://localhost:3000")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	if created.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("create CORS origin = %q", created.Header().Get("Access-Control-Allow-Origin"))
	}
	if created.Header().Get("Access-Control-Expose-Headers") != "Location, X-Request-Id" {
		t.Fatalf("create CORS exposed headers = %q", created.Header().Get("Access-Control-Expose-Headers"))
	}
	if created.Header().Get(requestIDHeader) == "" {
		t.Fatal("create response is missing a request ID")
	}

	var createPayload linkEnvelope
	if err := json.Unmarshal(created.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createPayload.Data.Code != "abc1234" {
		t.Fatalf("code = %q, want abc1234", createPayload.Data.Code)
	}
	if createPayload.Data.ShortURL != "https://npt-shortenlink.dev/link/abc1234" {
		t.Fatalf("short_url = %q", createPayload.Data.ShortURL)
	}
	if createPayload.Data.Status != "active" {
		t.Fatalf("status = %q, want active", createPayload.Data.Status)
	}
	wantExpiration := now.Add(7 * 24 * time.Hour).Format(time.RFC3339Nano)
	if createPayload.Data.ExpiresAt == nil || *createPayload.Data.ExpiresAt != wantExpiration {
		t.Fatalf("expires_at = %v, want %s", createPayload.Data.ExpiresAt, wantExpiration)
	}

	metadata := performRequest(router, http.MethodGet, "/api/v1/links/abc1234", nil, "")
	if metadata.Code != http.StatusOK {
		t.Fatalf("metadata status = %d, body = %s", metadata.Code, metadata.Body.String())
	}
	if metadata.Body.String() != created.Body.String() {
		t.Fatalf("metadata body = %s, want %s", metadata.Body.String(), created.Body.String())
	}

	redirect := performRequest(router, http.MethodGet, "/link/abc1234", nil, "")
	if redirect.Code != http.StatusFound {
		t.Fatalf("redirect status = %d, want 302", redirect.Code)
	}
	if redirect.Header().Get("Location") != "https://example.com/guide" {
		t.Fatalf("Location = %q", redirect.Header().Get("Location"))
	}
}

func TestCreateValidationAndConflict(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "malformed JSON", body: `{"url":`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "unknown field", body: `{"url":"https://example.com","title":"Example"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "null URL", body: `{"url":null}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "null custom alias", body: `{"url":"https://example.com","custom_alias":null}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "null expiration", body: `{"url":"https://example.com","expires_in_days":null}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "multiple JSON values", body: `{"url":"https://example.com"} {}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "invalid URL", body: `{"url":"javascript:alert(1)"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_url"},
		{name: "empty alias", body: `{"url":"https://example.com","custom_alias":""}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_custom_alias"},
		{name: "invalid alias", body: `{"url":"https://example.com","custom_alias":"Bad Alias"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_custom_alias"},
		{name: "reserved alias", body: `{"url":"https://example.com","custom_alias":"admin"}`, wantStatus: http.StatusBadRequest, wantCode: "reserved_custom_alias"},
		{name: "invalid expiration", body: `{"url":"https://example.com","expires_in_days":366}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_expiration"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, _ := newTestRouter(now)
			response := performRequest(router, http.MethodPost, "/api/v1/links", []byte(test.body), "")
			assertAPIError(t, response, test.wantStatus, test.wantCode)
		})
	}

	router, _ := newTestRouter(now)
	requestBody := []byte(`{"url":"https://example.com","custom_alias":"my-link"}`)
	first := performRequest(router, http.MethodPost, "/api/v1/links", requestBody, "")
	if first.Code != http.StatusCreated {
		t.Fatalf("first custom alias status = %d, body = %s", first.Code, first.Body.String())
	}
	second := performRequest(router, http.MethodPost, "/api/v1/links", requestBody, "")
	assertAPIError(t, second, http.StatusConflict, "custom_alias_conflict")

	omittedRouter, _ := newTestRouter(now)
	omittedOptionalFields := performRequest(omittedRouter, http.MethodPost, "/api/v1/links", []byte(`{"url":"https://example.com/without-optionals"}`), "")
	if omittedOptionalFields.Code != http.StatusCreated {
		t.Fatalf("omitted optional fields status = %d, body = %s", omittedOptionalFields.Code, omittedOptionalFields.Body.String())
	}
}

func TestCreateRequiresJSONContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "missing content type"},
		{name: "plain text", contentType: "text/plain"},
		{name: "JSON suffix", contentType: "application/problem+json"},
		{name: "malformed content type", contentType: "application/json; charset"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, _ := newTestRouter(time.Now())
			request := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(`{"url":"https://example.com"}`))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assertAPIError(t, response, http.StatusUnsupportedMediaType, "unsupported_media_type")
		})
	}

	router, _ := newTestRouter(time.Now())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(`{"url":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("JSON with charset status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestCreateRejectsOversizedBody(t *testing.T) {
	body := `{"url":"https://example.com"}` + strings.Repeat(" ", int(maxCreateLinkRequestBytes))
	tests := []struct {
		name          string
		contentLength int64
	}{
		{name: "declared content length", contentLength: int64(len(body))},
		{name: "streamed body", contentLength: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, _ := newTestRouter(time.Now())
			request := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.ContentLength = test.contentLength
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			assertAPIError(t, response, http.StatusRequestEntityTooLarge, "payload_too_large")
		})
	}
}

func TestPanicRecoveryReturnsSafeJSONAndPreservesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(panickingApplication{}, "https://npt-shortenlink.dev")
	router := NewRouter(handler, []string{"http://localhost:3000"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(`{"url":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(requestIDHeader, "panic-test-123")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertAPIError(t, response, http.StatusInternalServerError, "internal_error")
	if response.Header().Get(requestIDHeader) != "panic-test-123" {
		t.Fatalf("panic request ID = %q", response.Header().Get(requestIDHeader))
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("panic Content-Type = %q", response.Header().Get("Content-Type"))
	}
	if strings.Contains(response.Body.String(), "sensitive panic detail") || strings.Contains(response.Body.String(), "goroutine") {
		t.Fatalf("panic details leaked in response: %s", response.Body.String())
	}
}

func TestResolveStateErrors(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	router, repository := newTestRouter(now)
	seedLinks := []domain.Link{
		{Code: "expired", TargetURL: "https://example.com/expired", Enabled: true, CreatedAt: now.Add(-time.Hour), ExpiresAt: &past},
		{Code: "disabled", TargetURL: "https://example.com/disabled", Enabled: false, CreatedAt: now},
	}
	for _, link := range seedLinks {
		if err := repository.Create(context.Background(), link); err != nil {
			t.Fatalf("seed %s: %v", link.Code, err)
		}
	}

	assertAPIError(t, performRequest(router, http.MethodGet, "/link/missing", nil, ""), http.StatusNotFound, "link_not_found")
	assertAPIError(t, performRequest(router, http.MethodGet, "/link/expired", nil, ""), http.StatusGone, "link_expired")
	assertAPIError(t, performRequest(router, http.MethodGet, "/link/disabled", nil, ""), http.StatusGone, "link_disabled")

	metadata := performRequest(router, http.MethodGet, "/api/v1/links/expired", nil, "")
	if metadata.Code != http.StatusOK {
		t.Fatalf("expired metadata status = %d, body = %s", metadata.Code, metadata.Body.String())
	}
	var payload linkEnvelope
	if err := json.Unmarshal(metadata.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if payload.Data.Status != "expired" {
		t.Fatalf("expired metadata status field = %q", payload.Data.Status)
	}
}

func TestCORSAllowListAndPreflight(t *testing.T) {
	router, _ := newTestRouter(time.Now())

	preflight := performRequest(router, http.MethodOptions, "/api/v1/links", nil, "http://localhost:3000")
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", preflight.Code)
	}
	if preflight.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("preflight allow origin = %q", preflight.Header().Get("Access-Control-Allow-Origin"))
	}
	if preflight.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Fatalf("preflight allow methods = %q", preflight.Header().Get("Access-Control-Allow-Methods"))
	}
	if preflight.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization, X-Request-Id" {
		t.Fatalf("preflight allow headers = %q", preflight.Header().Get("Access-Control-Allow-Headers"))
	}
	if preflight.Header().Get("Access-Control-Expose-Headers") != "Location, X-Request-Id" {
		t.Fatalf("preflight expose headers = %q", preflight.Header().Get("Access-Control-Expose-Headers"))
	}
	if preflight.Header().Get("Access-Control-Max-Age") != "600" {
		t.Fatalf("preflight max age = %q", preflight.Header().Get("Access-Control-Max-Age"))
	}
	if preflight.Header().Get("Vary") != "Origin" {
		t.Fatalf("preflight Vary = %q", preflight.Header().Get("Vary"))
	}

	denied := performRequest(router, http.MethodGet, "/healthz", nil, "https://evil.example")
	assertAPIError(t, denied, http.StatusForbidden, "origin_not_allowed")
	if denied.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("denied allow origin = %q, want empty", denied.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRequestID(t *testing.T) {
	router, _ := newTestRouter(time.Now())

	generated := performRequest(router, http.MethodGet, "/healthz", nil, "")
	generatedID := generated.Header().Get(requestIDHeader)
	if !regexp.MustCompile(`^[A-Z2-7]{26}$`).MatchString(generatedID) {
		t.Fatalf("generated request ID = %q, want a 128-bit base32 token", generatedID)
	}

	const suppliedID = "frontend:request-123"
	preserved := performRequestWithRequestID(router, http.MethodGet, "/healthz", suppliedID)
	if preserved.Header().Get(requestIDHeader) != suppliedID {
		t.Fatalf("preserved request ID = %q, want %q", preserved.Header().Get(requestIDHeader), suppliedID)
	}

	invalid := performRequestWithRequestID(router, http.MethodGet, "/healthz", strings.Repeat("x", 129))
	if invalid.Header().Get(requestIDHeader) == strings.Repeat("x", 129) {
		t.Fatal("invalid request ID was not replaced")
	}
}

func performRequestWithRequestID(router http.Handler, method, path, requestID string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set(requestIDHeader, requestID)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func performRequest(router http.Handler, method, path string, body []byte, origin string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func assertAPIError(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, wantStatus, response.Body.String())
	}
	var payload errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q", payload.Error.Code, wantCode)
	}
}
