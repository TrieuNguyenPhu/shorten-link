package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"shortlink/internal/application"
	"shortlink/internal/domain"
)

type handler struct{ service *application.Service }

func New(service *application.Service, origin string) http.Handler {
	h := &handler{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !method(w, r, "GET", "HEAD") {
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/links", h.collection)
	mux.HandleFunc("/api/links/{code}", h.link)
	mux.HandleFunc("/r/{code}", h.redirect)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { writeError(w, 404, "not_found", "route not found") })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Add("Vary", "Origin")
		if requestOrigin := r.Header.Get("Origin"); requestOrigin != "" {
			if requestOrigin != origin {
				writeError(w, 403, "forbidden_origin", "origin is not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (h *handler) collection(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "HEAD", "POST") {
		return
	}
	if r.Method == http.MethodPost {
		var input application.CreateInput
		if !decode(w, r, &input) {
			return
		}
		link, err := h.service.Create(r.Context(), input)
		if err != nil {
			respondError(w, err)
			return
		}
		w.Header().Set("Location", "/api/links/"+link.Code)
		writeJSON(w, 201, link)
		return
	}
	page, err := h.service.List(r.Context(), r.URL.Query().Get("cursor"))
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, 200, page)
}

func (h *handler) link(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "HEAD", "PATCH") {
		return
	}
	code := r.PathValue("code")
	if r.Method == http.MethodPatch {
		var input struct {
			Enabled *bool `json:"enabled"`
		}
		if !decode(w, r, &input) {
			return
		}
		if input.Enabled == nil {
			writeError(w, 400, "invalid_input", "enabled must be a boolean")
			return
		}
		link, err := h.service.SetEnabled(r.Context(), code, *input.Enabled)
		if err != nil {
			respondError(w, err)
			return
		}
		writeJSON(w, 200, link)
		return
	}
	link, err := h.service.Get(r.Context(), code)
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, 200, link)
}

func (h *handler) redirect(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "HEAD") {
		return
	}
	link, err := h.service.Resolve(r.Context(), r.PathValue("code"))
	if err != nil {
		respondError(w, err)
		return
	}
	w.Header().Set("Location", link.URL)
	w.WriteHeader(http.StatusFound)
}

func method(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	for _, m := range allowed {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeError(w, 405, "method_not_allowed", "method is not allowed for this route")
	return false
}

func decode(w http.ResponseWriter, r *http.Request, input any) bool {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeError(w, 415, "unsupported_media_type", "use Content-Type: application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(input)
	if err == nil {
		var extra any
		if trailing := decoder.Decode(&extra); trailing != io.EOF {
			err = trailing
			if err == nil {
				err = errors.New("multiple JSON values")
			}
		}
	}
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			writeError(w, 413, "body_too_large", "JSON body must not exceed 16 KiB")
		} else {
			writeError(w, 400, "invalid_json", "body must be one valid JSON object with supported fields and types")
		}
		return false
	}
	return true
}

func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		writeError(w, 400, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, 404, "not_found", domain.ErrNotFound.Error())
	case errors.Is(err, domain.ErrConflict):
		writeError(w, 409, "conflict", domain.ErrConflict.Error())
	case errors.Is(err, domain.ErrGone):
		writeError(w, 410, "gone", domain.ErrGone.Error())
	case errors.Is(err, domain.ErrUnavailable):
		writeError(w, 503, "unavailable", domain.ErrUnavailable.Error())
	default:
		slog.Error("API operation failed", "error", err)
		writeError(w, 500, "internal_error", "unable to complete request; check the local API logs")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write response", "error", err)
	}
}
