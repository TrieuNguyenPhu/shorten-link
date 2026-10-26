// Package lambdaapi handles API Gateway HTTP API payload v2 directly.
package lambdaapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"shortlink/internal/application"
	"shortlink/internal/domain"
)

const (
	scopeRead  = "shortly-api/links.read"
	scopeWrite = "shortly-api/links.write"
)

type Handler struct{ service *application.Service }

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

func New(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return h.handle(ctx, request, "create"), nil
}
func (h *Handler) Manage(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return h.handle(ctx, request, "manage"), nil
}
func (h *Handler) Resolve(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return h.handle(ctx, request, "resolve"), nil
}

var routes = map[string]string{
	"POST /api/links":          "create",
	"GET /api/links":           "manage",
	"HEAD /api/links":          "manage",
	"GET /api/links/{code}":    "manage",
	"HEAD /api/links/{code}":   "manage",
	"PATCH /api/links/{code}":  "manage",
	"DELETE /api/links/{code}": "manage",
	"GET /health":              "manage",
	"HEAD /health":             "manage",
	"GET /r/{code}":            "resolve",
	"HEAD /r/{code}":           "resolve",
}

func authorizeRequest(request events.APIGatewayV2HTTPRequest, requiredScope string) (string, error) {
	if request.RequestContext.Authorizer == nil || request.RequestContext.Authorizer.JWT == nil {
		return "", domain.ErrUnauthorized
	}
	jwt := request.RequestContext.Authorizer.JWT
	sub := strings.TrimSpace(jwt.Claims["sub"])
	if !application.ValidActor(sub) {
		return "", domain.ErrUnauthorized
	}
	if tokenUse := strings.TrimSpace(jwt.Claims["token_use"]); tokenUse != "" && tokenUse != "access" {
		return "", domain.ErrForbidden
	}
	scopes := slices.Clone(jwt.Scopes)
	if rawScope := strings.TrimSpace(jwt.Claims["scope"]); rawScope != "" {
		scopes = append(scopes, strings.Fields(rawScope)...)
	}
	if !slices.Contains(scopes, requiredScope) {
		return "", domain.ErrForbidden
	}
	return sub, nil
}

func (h *Handler) handle(ctx context.Context, request events.APIGatewayV2HTTPRequest, group string) (response events.APIGatewayV2HTTPResponse) {
	started := time.Now()
	defer func() {
		if request.RequestContext.HTTP.Method == "HEAD" {
			response.Body = ""
		}
		lambdaID := ""
		if lc, ok := lambdacontext.FromContext(ctx); ok {
			lambdaID = lc.AwsRequestID
		}
		slog.Info("request completed", "gatewayRequestId", request.RequestContext.RequestID, "lambdaRequestId", lambdaID, "route", request.RouteKey, "status", response.StatusCode, "durationMs", time.Since(started).Milliseconds())
	}()
	if request.Version != "2.0" {
		return failure(400, "invalid_input", "API Gateway payload version 2.0 is required")
	}
	if routes[request.RouteKey] != group || !strings.HasPrefix(request.RouteKey, request.RequestContext.HTTP.Method+" ") {
		return failure(404, "not_found", "route not found")
	}
	code := request.PathParameters["code"]
	switch request.RouteKey {
	case "POST /api/links":
		actor, err := authorizeRequest(request, scopeWrite)
		if err != nil {
			return operationError(err, request)
		}
		var input application.CreateInput
		if invalid := decode(request, &input); invalid != nil {
			return *invalid
		}
		link, err := h.service.Create(ctx, actor, input)
		if err != nil {
			return operationError(err, request)
		}
		response = jsonResponse(201, toLinkResponse(link))
		response.Headers["Location"] = "/api/links/" + link.Code
		return response
	case "GET /r/{code}", "HEAD /r/{code}":
		link, err := h.service.Resolve(ctx, code)
		if err != nil {
			return operationError(err, request)
		}
		response = jsonResponse(302, nil)
		response.Headers["Location"], response.Body = link.URL, ""
		return response
	case "GET /health", "HEAD /health":
		return jsonResponse(200, map[string]string{"status": "ok"})
	case "GET /api/links", "HEAD /api/links":
		actor, err := authorizeRequest(request, scopeRead)
		if err != nil {
			return operationError(err, request)
		}
		query, err := url.ParseQuery(request.RawQueryString)
		if err != nil {
			return failure(400, "invalid_input", "malformed query string")
		}
		page, err := h.service.List(ctx, actor, query.Get("cursor"))
		if err != nil {
			return operationError(err, request)
		}
		return jsonResponse(200, toPageResponse(page))
	case "PATCH /api/links/{code}":
		actor, err := authorizeRequest(request, scopeWrite)
		if err != nil {
			return operationError(err, request)
		}
		var input application.UpdateInput
		if invalid := decode(request, &input); invalid != nil {
			return *invalid
		}
		link, err := h.service.Update(ctx, actor, code, input)
		if err != nil {
			return operationError(err, request)
		}
		return jsonResponse(200, toLinkResponse(link))
	case "DELETE /api/links/{code}":
		actor, err := authorizeRequest(request, scopeWrite)
		if err != nil {
			return operationError(err, request)
		}
		query, err := url.ParseQuery(request.RawQueryString)
		if err != nil {
			return failure(400, "invalid_input", "malformed query string")
		}
		var version int64
		if rawVersion := query.Get("version"); rawVersion != "" {
			parsed, parseErr := strconv.ParseInt(rawVersion, 10, 64)
			if parseErr != nil || parsed < 1 {
				return failure(400, "invalid_input", "version must be a positive integer")
			}
			version = parsed
		}
		if err := h.service.Delete(ctx, actor, code, version); err != nil {
			return operationError(err, request)
		}
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 204,
			Headers:    map[string]string{"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff"},
		}
	default:
		actor, err := authorizeRequest(request, scopeRead)
		if err != nil {
			return operationError(err, request)
		}
		link, err := h.service.Get(ctx, actor, code)
		if err != nil {
			return operationError(err, request)
		}
		return jsonResponse(200, toLinkResponse(link))
	}
}

func decode(request events.APIGatewayV2HTTPRequest, target any) *events.APIGatewayV2HTTPResponse {
	reject := func(status int, code, message string) *events.APIGatewayV2HTTPResponse {
		response := failure(status, code, message)
		return &response
	}
	contentType := ""
	for name, value := range request.Headers {
		if strings.EqualFold(name, "Content-Type") {
			contentType = value
		}
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return reject(415, "unsupported_media_type", "use Content-Type: application/json")
	}
	body := request.Body
	const limit = 16 * 1024
	if request.IsBase64Encoded {
		if len(body) > base64.StdEncoding.EncodedLen(limit) {
			return reject(413, "body_too_large", "JSON body must not exceed 16 KiB")
		}
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return reject(400, "invalid_json", "body is not valid base64")
		}
		body = string(decoded)
	}
	if len(body) > limit {
		return reject(413, "body_too_large", "JSON body must not exceed 16 KiB")
	}
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err == nil {
		var extra any
		if decoder.Decode(&extra) == io.EOF {
			return nil
		}
	}
	return reject(400, "invalid_json", "body must be one valid JSON object with supported fields and types")
}

func operationError(err error, request events.APIGatewayV2HTTPRequest) events.APIGatewayV2HTTPResponse {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		return failure(400, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		return failure(401, "unauthorized", domain.ErrUnauthorized.Error())
	case errors.Is(err, domain.ErrForbidden):
		return failure(403, "forbidden", domain.ErrForbidden.Error())
	case errors.Is(err, domain.ErrNotFound):
		return failure(404, "not_found", domain.ErrNotFound.Error())
	case errors.Is(err, domain.ErrVersionConflict):
		return failure(409, "version_conflict", "link was modified; refresh and try again")
	case errors.Is(err, domain.ErrConflict):
		return failure(409, "conflict", domain.ErrConflict.Error())
	case errors.Is(err, domain.ErrGone):
		return failure(410, "gone", domain.ErrGone.Error())
	case errors.Is(err, domain.ErrUnavailable):
		return failure(503, "unavailable", domain.ErrUnavailable.Error())
	default:
		slog.Error("API operation failed", "gatewayRequestId", request.RequestContext.RequestID, "error", err)
		return failure(500, "internal_error", "unable to complete request; check the API logs")
	}
}

func failure(status int, code, message string) events.APIGatewayV2HTTPResponse {
	return jsonResponse(status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func jsonResponse(status int, value any) events.APIGatewayV2HTTPResponse {
	body, err := json.Marshal(value)
	if err != nil {
		status, body = 500, []byte(`{"error":{"code":"internal_error","message":"unable to encode response"}}`)
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: status, Headers: map[string]string{"Content-Type": "application/json; charset=utf-8", "Cache-Control": "no-store", "X-Content-Type-Options": "nosniff"}, Body: string(body)}
}
