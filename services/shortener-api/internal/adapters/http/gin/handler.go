package ginhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/application/service"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

const maxCreateLinkRequestBytes int64 = 16 << 10

type linkApplication interface {
	Create(ctx context.Context, input service.CreateLinkInput) (service.LinkView, error)
	Resolve(ctx context.Context, code string) (domain.Link, error)
	GetMetadata(ctx context.Context, code string) (service.LinkView, error)
}

type Handler struct {
	links         linkApplication
	publicBaseURL string
}

type createLinkRequest struct {
	URL           jsonField[string] `json:"url"`
	CustomAlias   jsonField[string] `json:"custom_alias"`
	ExpiresInDays jsonField[int]    `json:"expires_in_days"`
}

type jsonField[T any] struct {
	Value   T
	Present bool
	Null    bool
}

func (f *jsonField[T]) UnmarshalJSON(data []byte) error {
	f.Present = true
	if string(data) == "null" {
		f.Null = true
		return nil
	}
	return json.Unmarshal(data, &f.Value)
}

type linkEnvelope struct {
	Data linkResponse `json:"data"`
}

type linkResponse struct {
	Code      string  `json:"code"`
	ShortURL  string  `json:"short_url"`
	TargetURL string  `json:"target_url"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt *string `json:"expires_at"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(links linkApplication, publicBaseURL string) *Handler {
	return &Handler{
		links:         links,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Create(c *gin.Context) {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		writeAPIError(c, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return
	}
	if c.Request.ContentLength > maxCreateLinkRequestBytes {
		writePayloadTooLarge(c)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxCreateLinkRequestBytes)

	var request createLinkRequest
	if err := decodeCreateLinkRequest(c, &request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writePayloadTooLarge(c)
			return
		}
		writeAPIError(c, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return
	}
	if request.CustomAlias.Present && request.CustomAlias.Value == "" {
		writeDomainError(c, domain.ErrInvalidAlias)
		return
	}

	var expiresInDays *int
	if request.ExpiresInDays.Present {
		expiresInDays = &request.ExpiresInDays.Value
	}

	view, err := h.links.Create(c.Request.Context(), service.CreateLinkInput{
		URL:           request.URL.Value,
		CustomAlias:   request.CustomAlias.Value,
		ExpiresInDays: expiresInDays,
	})
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, h.linkEnvelope(view))
}

func writePayloadTooLarge(c *gin.Context) {
	writeAPIError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "request body must not exceed 16 KiB")
}

func decodeCreateLinkRequest(c *gin.Context, request *createLinkRequest) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		return err
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain exactly one JSON value")
		}
		return err
	}

	if !request.URL.Present || request.URL.Null || request.CustomAlias.Null || request.ExpiresInDays.Null {
		return errors.New("request fields must be present and non-null according to the API contract")
	}
	return nil
}

func (h *Handler) Resolve(c *gin.Context) {
	link, err := h.links.Resolve(c.Request.Context(), c.Param("code"))
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.Redirect(http.StatusFound, link.TargetURL)
}

func (h *Handler) Metadata(c *gin.Context) {
	view, err := h.links.GetMetadata(c.Request.Context(), c.Param("code"))
	if err != nil {
		writeDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.linkEnvelope(view))
}

func (h *Handler) linkEnvelope(view service.LinkView) linkEnvelope {
	link := view.Link
	return linkEnvelope{Data: linkResponse{
		Code:      link.Code,
		ShortURL:  h.publicBaseURL + "/link/" + url.PathEscape(link.Code),
		TargetURL: link.TargetURL,
		Status:    string(view.Status),
		CreatedAt: link.CreatedAt.UTC().Format(time.RFC3339Nano),
		ExpiresAt: formatOptionalTime(link.ExpiresAt),
	}}
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func writeDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidURL):
		writeAPIError(c, http.StatusBadRequest, "invalid_url", "url must be an absolute http or https URL")
	case errors.Is(err, domain.ErrInvalidAlias):
		writeAPIError(c, http.StatusBadRequest, "invalid_custom_alias", "custom_alias must contain 4-32 lowercase letters, numbers, or hyphens")
	case errors.Is(err, domain.ErrReservedAlias):
		writeAPIError(c, http.StatusBadRequest, "reserved_custom_alias", "custom_alias is reserved")
	case errors.Is(err, domain.ErrInvalidExpiration):
		writeAPIError(c, http.StatusBadRequest, "invalid_expiration", "expires_in_days must be between 1 and 365")
	case errors.Is(err, domain.ErrCodeAlreadyExists):
		writeAPIError(c, http.StatusConflict, "custom_alias_conflict", "custom_alias is already in use")
	case errors.Is(err, domain.ErrLinkNotFound):
		writeAPIError(c, http.StatusNotFound, "link_not_found", "link was not found")
	case errors.Is(err, domain.ErrLinkExpired):
		writeAPIError(c, http.StatusGone, "link_expired", "link has expired")
	case errors.Is(err, domain.ErrLinkDisabled):
		writeAPIError(c, http.StatusGone, "link_disabled", "link is disabled")
	case errors.Is(err, domain.ErrCodeGenerationExhausted):
		writeAPIError(c, http.StatusServiceUnavailable, "code_generation_exhausted", "could not allocate a short code; retry later")
	default:
		writeAPIError(c, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

func writeAPIError(c *gin.Context, status int, code, message string) {
	c.Set(errorCodeKey, code)
	c.AbortWithStatusJSON(status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}
