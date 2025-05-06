package executions

import (
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/eduardolat/pgbackweb/internal/service/executions"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	servs *service.Service
}

func newHandlers(servs *service.Service) *handlers {
	return &handlers{servs: servs}
}

// ListExecutions godoc
// @Summary List all executions
// @Description Get a paginated list of all executions with optional filtering by backup ID
// @Tags executions
// @Accept json
// @Produce json
// @Param backup_id query string false "Filter by backup ID (UUID)"
// @Success 200 {object} map[string]interface{} "Returns a paginated list of executions"
// @Failure 400 {object} map[string]string "Invalid backup ID"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/executions [get]
func (h *handlers) listExecutionsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse query parameters
	var backupID uuid.NullUUID
	if backupIDStr := c.QueryParam("backup_id"); backupIDStr != "" {
		id, err := uuid.Parse(backupIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid backup ID",
			})
		}
		backupID = uuid.NullUUID{UUID: id, Valid: true}
	}

	// Get executions from database
	paginateResponse, executions, err := h.servs.ExecutionsService.PaginateExecutions(ctx, executions.PaginateExecutionsParams{
		Page:         1,
		Limit:        100,
		BackupFilter: backupID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get executions: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":       executions,
		"pagination": paginateResponse,
	})
}
