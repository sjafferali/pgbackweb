package restorations

import (
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/eduardolat/pgbackweb/internal/service/restorations"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	servs *service.Service
}

func newHandlers(servs *service.Service) *handlers {
	return &handlers{servs: servs}
}

// ListRestorations godoc
// @Summary List all restorations
// @Description Get a list of all restorations with optional filtering
// @Tags restorations
// @Accept json
// @Produce json
// @Param execution_id query string false "Filter by execution ID"
// @Param database_id query string false "Filter by database ID"
// @Success 200 {array} dbgen.RestorationsServicePaginateRestorationsRow
// @Router /api/restorations [get]
func (h *handlers) listRestorationsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse query parameters
	var executionID, databaseID uuid.UUID
	var err error

	if executionIDStr := c.QueryParam("execution_id"); executionIDStr != "" {
		executionID, err = uuid.Parse(executionIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid execution ID",
			})
		}
	}

	if databaseIDStr := c.QueryParam("database_id"); databaseIDStr != "" {
		databaseID, err = uuid.Parse(databaseIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid database ID",
			})
		}
	}

	// Get restorations from database
	_, restorations, err := h.servs.RestorationsService.PaginateRestorations(ctx, restorations.PaginateRestorationsParams{
		ExecutionFilter: uuid.NullUUID{UUID: executionID, Valid: executionID != uuid.Nil},
		DatabaseFilter:  uuid.NullUUID{UUID: databaseID, Valid: databaseID != uuid.Nil},
		Page:            1,
		Limit:           100,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get restorations: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": restorations,
	})
}
