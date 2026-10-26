package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"shortlink/internal/application"
	"shortlink/internal/domain"
)

const (
	scopeRead  = "shortly-api/links.read"
	scopeWrite = "shortly-api/links.write"
)

type authCodeEntry struct {
	Challenge   string
	RedirectURI string
	Sub         string
	Email       string
	Scope       string
	Nonce       string
	ExpiresAt   time.Time
}

type refreshEntry struct {
	Sub   string
	Email string
	Scope string
}

type handler struct {
	service   *application.Service
	origin    string
	mu        sync.Mutex
	authCodes map[string]authCodeEntry
	refreshes map[string]refreshEntry
}

type linkResponse struct {
	Code      string     `json:"code"`
	URL       string     `json:"url"`
	Enabled   bool       `json:"enabled"`
	Version   int64      `json:"version"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type pageResponse struct {
	Links      []linkResponse `json:"links"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

func toLinkResponse(l domain.Link) linkResponse {
	version := l.Version
	if version < 1 {
		version = 1
	}
	updatedAt := l.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = l.CreatedAt
	}
	return linkResponse{
		Code:      l.Code,
		URL:       l.URL,
		Enabled:   l.Enabled,
		Version:   version,
		CreatedAt: l.CreatedAt,
		UpdatedAt: updatedAt,
		ExpiresAt: l.ExpiresAt,
	}
}

func toPageResponse(p domain.Page) pageResponse {
	links := make([]linkResponse, len(p.Links))
	for i, l := range p.Links {
		links[i] = toLinkResponse(l)
	}
	return pageResponse{Links: links, NextCursor: p.NextCursor}
}

func loopbackHost(raw string) bool {
	host := raw
	if h, p, err := net.SplitHostPort(raw); err == nil {
		if p == "" {
			return false
		}
		if _, err := strconv.ParseUint(p, 10, 16); err != nil {
			return false
		}
		host = h
	} else if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		host = raw[1 : len(raw)-1]
	} else if strings.Contains(raw, ":") {
		return false
	}
	return strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
}

func authorizeRequest(r *http.Request, requiredScope string) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(header) < 8 || !strings.EqualFold(header[:7], "Bearer ") {
		return "", domain.ErrUnauthorized
	}
	token := strings.TrimSpace(header[7:])
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", domain.ErrUnauthorized
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", domain.ErrUnauthorized
	}
	var claims struct {
		Sub      string   `json:"sub"`
		Scope    string   `json:"scope"`
		Scopes   []string `json:"scopes"`
		TokenUse string   `json:"token_use"`
		Exp      *int64   `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", domain.ErrUnauthorized
	}
	sub := strings.TrimSpace(claims.Sub)
	if !application.ValidActor(sub) {
		return "", domain.ErrUnauthorized
	}
	if claims.Exp != nil && time.Now().Unix() >= *claims.Exp {
		return "", domain.ErrUnauthorized
	}
	if claims.TokenUse != "" && claims.TokenUse != "access" {
		return "", domain.ErrForbidden
	}
	scopes := append(slices.Clone(claims.Scopes), strings.Fields(claims.Scope)...)
	if !slices.Contains(scopes, requiredScope) {
		return "", domain.ErrForbidden
	}
	return sub, nil
}

func New(service *application.Service, origin string) http.Handler {
	h := &handler{
		service:   service,
		origin:    origin,
		authCodes: make(map[string]authCodeEntry),
		refreshes: make(map[string]refreshEntry),
	}
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
	mux.HandleFunc("/oauth2-local/authorize", h.oauthAuthorize)
	mux.HandleFunc("/oauth2-local/oauth2/authorize", h.oauthAuthorize)
	mux.HandleFunc("/oauth2-local/signup", h.oauthAuthorize)
	mux.HandleFunc("/oauth2-local/token", h.oauthToken)
	mux.HandleFunc("/oauth2-local/oauth2/token", h.oauthToken)
	mux.HandleFunc("/oauth2-local/revoke", h.oauthRevoke)
	mux.HandleFunc("/oauth2-local/oauth2/revoke", h.oauthRevoke)
	mux.HandleFunc("/oauth2-local/logout", h.oauthLogout)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { writeError(w, 404, "not_found", "route not found") })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Add("Vary", "Origin")
		if !loopbackHost(r.Host) {
			writeError(w, 403, "forbidden_host", "host is not allowed")
			return
		}
		if requestOrigin := r.Header.Get("Origin"); requestOrigin != "" {
			if requestOrigin != origin {
				writeError(w, 403, "forbidden_origin", "origin is not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "DELETE, GET, HEAD, OPTIONS, PATCH, POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "Location, WWW-Authenticate")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func encodeUnsignedJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	raw, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return header + "." + payload + ".local-sig"
}

func (h *handler) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET") {
		return
	}
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	challenge := q.Get("code_challenge")
	challengeMethod := q.Get("code_challenge_method")
	if redirectURI == "" || q.Get("response_type") != "code" || challenge == "" || challengeMethod != "S256" {
		writeError(w, 400, "invalid_input", "PKCE S256 authorization code parameters are required")
		return
	}
	if !strings.HasPrefix(redirectURI, h.origin+"/") {
		writeError(w, 400, "invalid_input", "redirect_uri does not match configured web origin")
		return
	}
	sub := "local-user-a"
	email := "alice@example.com"
	if hint := strings.TrimSpace(q.Get("login_hint")); hint != "" {
		email = hint
		sum := sha256.Sum256([]byte(hint))
		sub = "user-" + hex.EncodeToString(sum[:6])
	}
	scope := q.Get("scope")
	if scope == "" {
		scope = "openid email " + scopeRead + " " + scopeWrite
	}
	code := randomHex(16)
	h.mu.Lock()
	h.authCodes[code] = authCodeEntry{
		Challenge:   challenge,
		RedirectURI: redirectURI,
		Sub:         sub,
		Email:       email,
		Scope:       scope,
		Nonce:       q.Get("nonce"),
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	h.mu.Unlock()

	target, err := url.Parse(redirectURI)
	if err != nil {
		writeError(w, 400, "invalid_input", "invalid redirect_uri")
		return
	}
	params := target.Query()
	params.Set("code", code)
	if state != "" {
		params.Set("state", state)
	}
	target.RawQuery = params.Encode()
	w.Header().Set("Location", target.String())
	w.WriteHeader(http.StatusFound)
}

func (h *handler) oauthToken(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "POST") {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := r.ParseForm(); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid_request"})
		return
	}
	grantType := r.FormValue("grant_type")
	now := time.Now().UTC()
	exp := now.Add(time.Hour).Unix()
	issuer := "http://" + r.Host + "/oauth2-local"

	switch grantType {
	case "authorization_code":
		code := r.FormValue("code")
		verifier := r.FormValue("code_verifier")
		redirectURI := r.FormValue("redirect_uri")
		h.mu.Lock()
		entry, ok := h.authCodes[code]
		delete(h.authCodes, code)
		h.mu.Unlock()
		if !ok || now.After(entry.ExpiresAt) || verifier == "" || (redirectURI != "" && redirectURI != entry.RedirectURI) {
			writeJSON(w, 400, map[string]string{"error": "invalid_grant"})
			return
		}
		sum := sha256.Sum256([]byte(verifier))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != entry.Challenge {
			writeJSON(w, 400, map[string]string{"error": "invalid_grant", "error_description": "PKCE verification failed"})
			return
		}
		refreshToken := randomHex(24)
		h.mu.Lock()
		h.refreshes[refreshToken] = refreshEntry{Sub: entry.Sub, Email: entry.Email, Scope: entry.Scope}
		h.mu.Unlock()

		accessToken := encodeUnsignedJWT(map[string]any{
			"iss":       issuer,
			"sub":       entry.Sub,
			"scope":     entry.Scope,
			"token_use": "access",
			"iat":       now.Unix(),
			"exp":       exp,
		})
		idClaims := map[string]any{
			"iss":       issuer,
			"sub":       entry.Sub,
			"aud":       r.FormValue("client_id"),
			"email":     entry.Email,
			"token_use": "id",
			"iat":       now.Unix(),
			"exp":       exp,
		}
		if entry.Nonce != "" {
			idClaims["nonce"] = entry.Nonce
		}
		idToken := encodeUnsignedJWT(idClaims)
		writeJSON(w, 200, map[string]any{
			"access_token":  accessToken,
			"id_token":      idToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         entry.Scope,
		})
	case "refresh_token":
		rt := r.FormValue("refresh_token")
		h.mu.Lock()
		entry, ok := h.refreshes[rt]
		h.mu.Unlock()
		if !ok {
			writeJSON(w, 400, map[string]string{"error": "invalid_grant"})
			return
		}
		accessToken := encodeUnsignedJWT(map[string]any{
			"iss":       issuer,
			"sub":       entry.Sub,
			"scope":     entry.Scope,
			"token_use": "access",
			"iat":       now.Unix(),
			"exp":       exp,
		})
		idToken := encodeUnsignedJWT(map[string]any{
			"iss":       issuer,
			"sub":       entry.Sub,
			"aud":       r.FormValue("client_id"),
			"email":     entry.Email,
			"token_use": "id",
			"iat":       now.Unix(),
			"exp":       exp,
		})
		writeJSON(w, 200, map[string]any{
			"access_token": accessToken,
			"id_token":     idToken,
			"token_type":   "Bearer",
			"expires_in":   3600,
			"scope":        entry.Scope,
		})
	default:
		writeJSON(w, 400, map[string]string{"error": "unsupported_grant_type"})
	}
}

func (h *handler) oauthRevoke(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "POST") {
		return
	}
	_ = r.ParseForm()
	if token := r.FormValue("token"); token != "" {
		h.mu.Lock()
		delete(h.refreshes, token)
		h.mu.Unlock()
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler) oauthLogout(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET") {
		return
	}
	logoutURI := r.URL.Query().Get("logout_uri")
	if logoutURI == "" {
		logoutURI = r.URL.Query().Get("post_logout_redirect_uri")
	}
	if logoutURI == "" || !strings.HasPrefix(logoutURI, h.origin) {
		logoutURI = h.origin + "/"
	}
	w.Header().Set("Location", logoutURI)
	w.WriteHeader(http.StatusFound)
}

func (h *handler) collection(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "HEAD", "POST") {
		return
	}
	if r.Method == http.MethodPost {
		actor, err := authorizeRequest(r, scopeWrite)
		if err != nil {
			respondError(w, err)
			return
		}
		var input application.CreateInput
		if !decode(w, r, &input) {
			return
		}
		link, err := h.service.Create(r.Context(), actor, input)
		if err != nil {
			respondError(w, err)
			return
		}
		w.Header().Set("Location", "/api/links/"+link.Code)
		writeJSON(w, 201, toLinkResponse(link))
		return
	}
	actor, err := authorizeRequest(r, scopeRead)
	if err != nil {
		respondError(w, err)
		return
	}
	page, err := h.service.List(r.Context(), actor, r.URL.Query().Get("cursor"))
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, 200, toPageResponse(page))
}

func (h *handler) link(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "HEAD", "PATCH", "DELETE") {
		return
	}
	code := r.PathValue("code")
	switch r.Method {
	case http.MethodPatch:
		actor, err := authorizeRequest(r, scopeWrite)
		if err != nil {
			respondError(w, err)
			return
		}
		var input application.UpdateInput
		if !decode(w, r, &input) {
			return
		}
		link, err := h.service.Update(r.Context(), actor, code, input)
		if err != nil {
			respondError(w, err)
			return
		}
		writeJSON(w, 200, toLinkResponse(link))
		return
	case http.MethodDelete:
		actor, err := authorizeRequest(r, scopeWrite)
		if err != nil {
			respondError(w, err)
			return
		}
		var version int64
		if rawVersion := r.URL.Query().Get("version"); rawVersion != "" {
			parsed, parseErr := strconv.ParseInt(rawVersion, 10, 64)
			if parseErr != nil || parsed < 1 {
				writeError(w, 400, "invalid_input", "version must be a positive integer")
				return
			}
			version = parsed
		}
		if err := h.service.Delete(r.Context(), actor, code, version); err != nil {
			respondError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		actor, err := authorizeRequest(r, scopeRead)
		if err != nil {
			respondError(w, err)
			return
		}
		link, err := h.service.Get(r.Context(), actor, code)
		if err != nil {
			respondError(w, err)
			return
		}
		writeJSON(w, 200, toLinkResponse(link))
	}
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
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, 401, "unauthorized", domain.ErrUnauthorized.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, 403, "forbidden", domain.ErrForbidden.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, 404, "not_found", domain.ErrNotFound.Error())
	case errors.Is(err, domain.ErrVersionConflict):
		writeError(w, 409, "version_conflict", "link was modified; refresh and try again")
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
