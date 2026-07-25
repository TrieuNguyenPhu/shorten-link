package ginhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/adapters/repository/memory"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/application/service"
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
	return &ginTestRouter{handler: http.Handler(NewRouter(handler, nil))}
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
