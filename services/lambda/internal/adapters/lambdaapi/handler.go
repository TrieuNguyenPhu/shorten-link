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
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"shortlink/internal/application"
	"shortlink/internal/domain"
)

type Handler struct{ service *application.Service }

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
	"POST /api/links": "create",
	"GET /api/links":  "manage", "HEAD /api/links": "manage",
	"GET /api/links/{code}": "manage", "HEAD /api/links/{code}": "manage", "PATCH /api/links/{code}": "manage",
	"GET /health": "manage", "HEAD /health": "manage",
	"GET /r/{code}": "resolve", "HEAD /r/{code}": "resolve",
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
		var input application.CreateInput
		if invalid := decode(request, &input); invalid != nil {
			return *invalid
		}
		link, err := h.service.Create(ctx, input)
		if err != nil {
			return operationError(err, request)
		}
		response = jsonResponse(201, link)
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
		query, err := url.ParseQuery(request.RawQueryString)
		if err != nil {
			return failure(400, "invalid_input", "malformed query string")
		}
		page, err := h.service.List(ctx, query.Get("cursor"))
		if err != nil {
			return operationError(err, request)
		}
		return jsonResponse(200, page)
	case "PATCH /api/links/{code}":
		var input struct {
			Enabled *bool `json:"enabled"`
		}
		if invalid := decode(request, &input); invalid != nil {
			return *invalid
		}
		if input.Enabled == nil {
			return failure(400, "invalid_input", "enabled must be a boolean")
		}
		link, err := h.service.SetEnabled(ctx, code, *input.Enabled)
		if err != nil {
			return operationError(err, request)
		}
		return jsonResponse(200, link)
	default:
		link, err := h.service.Get(ctx, code)
		if err != nil {
			return operationError(err, request)
		}
		return jsonResponse(200, link)
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
	case errors.Is(err, domain.ErrNotFound):
		return failure(404, "not_found", domain.ErrNotFound.Error())
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
