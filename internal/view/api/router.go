package api

import (
	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/eduardolat/pgbackweb/internal/view/middleware"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	servs *service.Service
}

func MountRouter(
	parent *echo.Group, mids *middleware.Middleware, servs *service.Service,
) {
	v1 := parent.Group("/v1")

	h := &handlers{
		servs: servs,
	}

	// Public endpoints
	v1.GET("/health", h.healthHandler)

	// Protected endpoints
	protected := v1.Group("", APIKeyAuth())
	protected.POST("/databases", h.createDatabaseAPI)
	protected.DELETE("/databases/:id", h.deleteDatabaseAPI)
}
