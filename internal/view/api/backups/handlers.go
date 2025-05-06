package backups

import (
	"encoding/json"
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/service"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	servs *service.Service
}

func newHandlers(servs *service.Service) *handlers {
	return &handlers{servs: servs}
}

// ListBackups godoc
// @Summary List all backup configurations
// @Description Get a list of all backup configurations
// @Tags backups
// @Accept json
// @Produce json
// @Success 200 {array} dbgen.Backup
// @Router /api/backups [get]
func (h *handlers) listBackupsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	backups, err := h.servs.BackupsService.GetAllBackups(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, backups)
}

func (h *handlers) getBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid backup ID",
		})
	}

	// Get backup from database
	backup, err := h.servs.BackupsService.GetBackup(ctx, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get backup: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, backup)
}

func (h *handlers) deleteBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid backup ID",
		})
	}

	// Delete backup from database
	err = h.servs.BackupsService.DeleteBackup(ctx, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete backup: " + err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// CreateBackup godoc
// @Summary Create a new backup configuration
// @Description Create a new backup configuration with the provided parameters
// @Tags backups
// @Accept json
// @Produce json
// @Param backup body dbgen.BackupsServiceCreateBackupParams true "Backup configuration"
// @Success 201 {object} dbgen.Backup
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/backups [post]
func (h *handlers) createBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse request body
	var requestBody struct {
		DatabaseID     string `json:"database_id"`
		DestinationID  string `json:"destination_id"`
		IsLocal        bool   `json:"is_local"`
		Name           string `json:"name"`
		CronExpression string `json:"cron_expression"`
		TimeZone       string `json:"time_zone"`
		IsActive       bool   `json:"is_active"`
		DestDir        string `json:"dest_dir"`
		RetentionDays  int16  `json:"retention_days"`
		OptDataOnly    bool   `json:"opt_data_only"`
		OptSchemaOnly  bool   `json:"opt_schema_only"`
		OptClean       bool   `json:"opt_clean"`
		OptIfExists    bool   `json:"opt_if_exists"`
		OptCreate      bool   `json:"opt_create"`
		OptNoComments  bool   `json:"opt_no_comments"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&requestBody); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	// Parse UUIDs
	databaseID, err := uuid.Parse(requestBody.DatabaseID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid database ID",
		})
	}

	var destinationID uuid.NullUUID
	if !requestBody.IsLocal {
		id, err := uuid.Parse(requestBody.DestinationID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid destination ID",
			})
		}
		destinationID = uuid.NullUUID{UUID: id, Valid: true}
	}

	// Create backup in database
	backup, err := h.servs.BackupsService.CreateBackup(ctx, dbgen.BackupsServiceCreateBackupParams{
		DatabaseID:     databaseID,
		DestinationID:  destinationID,
		IsLocal:        requestBody.IsLocal,
		Name:           requestBody.Name,
		CronExpression: requestBody.CronExpression,
		TimeZone:       requestBody.TimeZone,
		IsActive:       requestBody.IsActive,
		DestDir:        requestBody.DestDir,
		RetentionDays:  requestBody.RetentionDays,
		OptDataOnly:    requestBody.OptDataOnly,
		OptSchemaOnly:  requestBody.OptSchemaOnly,
		OptClean:       requestBody.OptClean,
		OptIfExists:    requestBody.OptIfExists,
		OptCreate:      requestBody.OptCreate,
		OptNoComments:  requestBody.OptNoComments,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create backup: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, backup)
}

// TriggerBackup godoc
// @Summary Trigger a backup task
// @Description Trigger a backup task to run immediately
// @Tags backups
// @Accept json
// @Produce json
// @Param id path string true "Backup ID"
// @Success 200 {object} map[string]string
// @Router /api/backups/{id}/trigger [post]
func (h *handlers) triggerBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup ID from URL parameter
	backupIDStr := c.Param("id")
	backupID, err := uuid.Parse(backupIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid backup ID",
		})
	}

	// Trigger the backup
	err = h.servs.ExecutionsService.RunExecution(ctx, backupID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to trigger backup: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Backup task triggered successfully",
	})
}
