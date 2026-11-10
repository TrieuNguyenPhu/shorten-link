package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/events"
)

type Handler struct {
	service *service.LinkService
	baseURL string
	logger  *slog.Logger
	metrics io.Writer
}

func New(service *service.LinkService, baseURL string, logger *slog.Logger, metrics io.Writer) *Handler {
	return &Handler{service: service, baseURL: strings.TrimRight(baseURL, "/"), logger: logger, metrics: metrics}
}

func (h *Handler) shortURL(req events.APIGatewayV2HTTPRequest, code string) string {
	base := h.baseURL
	if base == "" {
		base = "https://" + req.RequestContext.DomainName
	}
	return base + "/" + code
}

func identity(req events.APIGatewayV2HTTPRequest) string {
	auth := req.RequestContext.Authorizer
	if auth == nil || auth.JWT == nil {
		return ""
	}
	return strings.TrimSpace(auth.JWT.Claims["sub"])
}

func jsonResponse(status int, value any) (events.APIGatewayV2HTTPResponse, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: status, Headers: map[string]string{"Content-Type": "application/json"}, Body: string(body)}, nil
}

func failure(status int, message string) (events.APIGatewayV2HTTPResponse, error) {
	return jsonResponse(status, map[string]string{"error": message})
}

func (h *Handler) Create(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	owner := identity(req)
	if owner == "" {
		return failure(401, "thieu danh tinh nguoi dung")
	}
	body := req.Body
	if req.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return failure(400, "body JSON khong hop le")
		}
		body = string(decoded)
	}
	if len(body) > 16384 {
		return failure(400, "body JSON qua lon")
	}
	var input *struct {
		URL string `json:"url"`
	}
	decoder := json.NewDecoder(strings.NewReader(body))
	if err := decoder.Decode(&input); err != nil || input == nil {
		return failure(400, "body JSON khong hop le")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return failure(400, "body JSON khong hop le")
	}
	link, err := h.service.Create(ctx, owner, input.URL)
	switch {
	case errors.Is(err, service.ErrInvalidURL):
		return failure(400, "thieu hoac sai dinh dang 'url' (chi nhan http/https cong khai, an toan)")
	case errors.Is(err, service.ErrCodeExhausted):
		return failure(500, "khong sinh duoc ma duy nhat, thu lai")
	case err != nil:
		h.log(ctx, req).Error("create link failed", "error", err)
		return failure(500, "loi he thong")
	}
	return jsonResponse(201, map[string]string{"code": link.Code, "shortUrl": h.shortURL(req, link.Code), "target": link.Target, "status": link.Status})
}

func (h *Handler) List(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	owner := identity(req)
	if owner == "" {
		return failure(401, "thieu danh tinh nguoi dung")
	}
	limit := 25
	if raw := req.QueryStringParameters["limit"]; raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return failure(400, "tham so phan trang khong hop le")
		}
		limit = value
	}
	page, err := h.service.List(ctx, owner, limit, req.QueryStringParameters["cursor"])
	if errors.Is(err, repository.ErrInvalidPagination) {
		return failure(400, "tham so phan trang khong hop le")
	}
	links := page.Links
	if err != nil {
		h.log(ctx, req).Error("list links failed", "error", err)
		return failure(500, "loi he thong")
	}
	type item struct {
		Code           string `json:"code"`
		ShortURL       string `json:"shortUrl"`
		Target         string `json:"target"`
		Clicks         int64  `json:"clicks"`
		CreatedAt      string `json:"createdAt"`
		RealtimeTicket string `json:"realtimeTicket,omitempty"`
		Status         string `json:"status"`
		StatusReason   string `json:"statusReason,omitempty"`
	}
	items := make([]item, 0, len(links))
	for _, link := range links {
		entry := item{Code: link.Code, ShortURL: h.shortURL(req, link.Code), Target: link.Target, Clicks: link.Clicks, CreatedAt: link.CreatedAt, Status: link.Status, StatusReason: link.StatusReason}
		if ticket, ticketErr := h.service.IssueRealtimeTicketForLink(ctx, owner, link); ticketErr == nil {
			entry.RealtimeTicket = ticket.Ticket
		} else if !errors.Is(ticketErr, service.ErrTicketStoreUnavailable) {
			h.log(ctx, req).Error("issue realtime ticket failed", "code", link.Code, "error", ticketErr)
			return failure(500, "loi he thong")
		}
		items = append(items, entry)
	}
	return jsonResponse(200, struct {
		Links      []item `json:"links"`
		NextCursor string `json:"nextCursor,omitempty"`
	}{items, page.NextCursor})
}

func (h *Handler) Delete(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	owner := identity(req)
	if owner == "" {
		return failure(401, "thieu danh tinh nguoi dung")
	}
	code := req.PathParameters["code"]
	if !domain.IsValidShortCode(code) {
		return failure(404, "link khong ton tai hoac khong thuoc ve ban")
	}
	err := h.service.Delete(ctx, code, owner)
	if errors.Is(err, repository.ErrNotFound) {
		return failure(404, "link khong ton tai hoac khong thuoc ve ban")
	}
	if err != nil {
		h.log(ctx, req).Error("delete link failed", "error", err)
		return failure(500, "loi he thong")
	}
	return jsonResponse(200, map[string]string{"deleted": code})
}

func (h *Handler) Resolve(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	code := req.PathParameters["code"]
	if !domain.IsValidShortCode(code) {
		return events.APIGatewayV2HTTPResponse{StatusCode: 404, Headers: map[string]string{"Content-Type": "text/plain"}, Body: "khong tim thay link"}, nil
	}
	target, err := h.service.Resolve(ctx, code)
	if errors.Is(err, repository.ErrNotFound) {
		h.log(ctx, req).Warn("link not found", "code", code)
		h.metric(ctx, req, "LinkNotFound")
		return events.APIGatewayV2HTTPResponse{StatusCode: 404, Headers: map[string]string{"Content-Type": "text/plain"}, Body: "khong tim thay link"}, nil
	}
	if err != nil {
		h.log(ctx, req).Error("resolve link failed", "code", code, "error", err)
		if errors.Is(err, service.ErrPublish) && target != "" {
			h.metric(ctx, req, "ClickPublishFailed")
			return events.APIGatewayV2HTTPResponse{StatusCode: 302, Headers: map[string]string{"Location": target, "Cache-Control": "private, no-store, no-cache, must-revalidate"}, Body: ""}, nil
		}
		h.metric(ctx, req, "ResolveFailed")
		return events.APIGatewayV2HTTPResponse{StatusCode: 500, Headers: map[string]string{"Content-Type": "text/plain"}, Body: "loi he thong"}, nil
	}
	h.log(ctx, req).Info("link resolved", "code", code)
	h.metric(ctx, req, "LinkResolved")
	return events.APIGatewayV2HTTPResponse{StatusCode: 302, Headers: map[string]string{"Location": target, "Cache-Control": "private, no-store, no-cache, must-revalidate"}, Body: ""}, nil
}

func (h *Handler) log(ctx context.Context, req events.APIGatewayV2HTTPRequest) *slog.Logger {
	logger := middleware.WithRequestID(ctx, h.logger)
	if req.RequestContext.RequestID != "" {
		logger = logger.With("apiRequestId", req.RequestContext.RequestID)
	}
	return logger
}

func (h *Handler) metric(ctx context.Context, req events.APIGatewayV2HTTPRequest, name string) {
	if err := middleware.EmitCountMetric(h.metrics, "UrlShortener", "resolve-link", name, 1); err != nil {
		h.log(ctx, req).Error("metric emission failed", "metric", name, "error", err)
	}
}
