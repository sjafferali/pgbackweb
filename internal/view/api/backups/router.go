package backups

import (
	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/labstack/echo/v4"
)

func MountRouter(parent *echo.Group, servs *service.Service) {
	h := newHandlers(servs)

	parent.GET("", h.listBackupsHandler)
	parent.GET("/:id", h.getBackupHandler)
	parent.POST("", h.createBackupHandler)
	parent.DELETE("/:id", h.deleteBackupHandler)
}
