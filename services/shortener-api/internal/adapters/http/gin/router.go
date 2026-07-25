package ginhttp

import "github.com/gin-gonic/gin"

const errorCodeKey = "error_code"

func NewRouter(handler *Handler, _ []string) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)

	router.GET("/healthz", handler.Health)
	router.GET("/link/:code", handler.Resolve)
	router.POST("/api/v1/links", handler.Create)
	router.GET("/api/v1/links/:code", handler.Metadata)

	return router
}
