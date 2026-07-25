package ginhttp

import (
	"crypto/rand"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	requestIDHeader = "X-Request-Id"
	requestIDKey    = "request_id"
	errorCodeKey    = "error_code"
)

func NewRouter(handler *Handler, allowedOrigins []string) *gin.Engine {
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.Use(corsMiddleware(allowedOrigins))
	_ = router.SetTrustedProxies(nil)

	router.GET("/healthz", handler.Health)
	router.GET("/link/:code", handler.Resolve)
	router.POST("/api/v1/links", handler.Create)
	router.GET("/api/v1/links/:code", handler.Metadata)

	return router
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if !isValidRequestID(requestID) {
			requestID = newRequestID()
		}

		c.Set(requestIDKey, requestID)
		c.Header(requestIDHeader, requestID)
		c.Next()
	}
}

func newRequestID() string {
	return rand.Text()
}

func isValidRequestID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' || character == '.' || character == ':' {
			continue
		}
		return false
	}
	return true
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		c.Header("Vary", "Origin")
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if _, ok := allowed[origin]; !ok {
			writeAPIError(c, http.StatusForbidden, "origin_not_allowed", "request origin is not allowed")
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id")
		c.Header("Access-Control-Expose-Headers", "Location, X-Request-Id")
		c.Header("Access-Control-Max-Age", "600")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
