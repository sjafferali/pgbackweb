package api

import (
	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/eduardolat/pgbackweb/internal/view/api/restorations"
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
	protected.GET("/databases", h.listDatabasesAPI)
	protected.GET("/databases/:id", h.getDatabaseAPI)
	protected.DELETE("/databases/:id", h.deleteDatabaseAPI)

	// Webhook endpoints
	protected.POST("/webhooks", h.createWebhookAPI)
	protected.GET("/webhooks", h.listWebhooksAPI)
	protected.GET("/webhooks/:id", h.getWebhookAPI)
	protected.DELETE("/webhooks/:id", h.deleteWebhookAPI)

	// Restoration endpoints
	restorationsGroup := protected.Group("/restorations")
	restorations.MountRouter(restorationsGroup, servs)
}
