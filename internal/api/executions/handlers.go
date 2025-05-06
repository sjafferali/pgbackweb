package executions

import (
	"context"
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ExecutionsServiceInterface defines the interface for the ExecutionsService
type ExecutionsServiceInterface interface {
	ListBackupExecutions(ctx context.Context, backupID uuid.UUID) ([]dbgen.Execution, error)
	GetExecution(ctx context.Context, id uuid.UUID) (dbgen.ExecutionsServiceGetExecutionRow, error)
}

// Handlers contains all the handlers for the executions API
type Handlers struct {
	servs *service.Service
}

// NewHandlers creates a new instance of Handlers
func NewHandlers(servs *service.Service) *Handlers {
	return &Handlers{
		servs: servs,
	}
}

// ListExecutionsHandler handles GET /api/executions
func (h *Handlers) ListExecutionsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup_id from query parameter if provided
	var backupID uuid.UUID
	if backupIDStr := c.QueryParam("backup_id"); backupIDStr != "" {
		id, err := uuid.Parse(backupIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid backup ID",
			})
		}
		backupID = id
	}

	// Get executions from database
	executions, err := h.servs.ExecutionsService.ListBackupExecutions(ctx, backupID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get executions: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, executions)
}

// GetExecutionHandler handles GET /api/executions/:id
func (h *Handlers) GetExecutionHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get execution ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid execution ID",
		})
	}

	// Get execution from database
	execution, err := h.servs.ExecutionsService.GetExecution(ctx, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get execution: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, execution)
}
