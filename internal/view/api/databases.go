package api

import (
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type createDatabaseRequest struct {
	Name             string `json:"name" validate:"required"`
	Version          string `json:"version" validate:"required,oneof=13 14 15 16"`
	ConnectionString string `json:"connection_string" validate:"required"`
}

type databaseResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"`
}

func (h *handlers) createDatabaseAPI(c echo.Context) error {
	ctx := c.Request().Context()

	var req createDatabaseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if err := validate.Struct(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Test the database connection first
	if err := h.servs.DatabasesService.TestDatabase(ctx, req.Version, req.ConnectionString); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to connect to database: " + err.Error(),
		})
	}

	// Create the database
	db, err := h.servs.DatabasesService.CreateDatabase(
		ctx, dbgen.DatabasesServiceCreateDatabaseParams{
			Name:             req.Name,
			PgVersion:        req.Version,
			ConnectionString: req.ConnectionString,
		},
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create database: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, databaseResponse{
		ID:        db.ID.String(),
		Name:      db.Name,
		Version:   db.PgVersion,
		CreatedAt: db.CreatedAt.String(),
	})
}

func (h *handlers) deleteDatabaseAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get database ID from URL parameter
	dbIDStr := c.Param("id")
	dbID, err := uuid.Parse(dbIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid database ID",
		})
	}

	// Delete the database
	err = h.servs.DatabasesService.DeleteDatabase(ctx, dbID)
	if err != nil {
		// Check if the error is due to database not found
		if err.Error() == "database not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Database not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete database: " + err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *handlers) listDatabasesAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get all databases
	databases, err := h.servs.DatabasesService.GetAllDatabases(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list databases: " + err.Error(),
		})
	}

	// Convert to response format
	response := make([]databaseResponse, len(databases))
	for i, db := range databases {
		response[i] = databaseResponse{
			ID:        db.ID.String(),
			Name:      db.Name,
			Version:   db.PgVersion,
			CreatedAt: db.CreatedAt.String(),
		}
	}

	return c.JSON(http.StatusOK, response)
}

func (h *handlers) getDatabaseAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get database ID from URL parameter
	dbIDStr := c.Param("id")
	dbID, err := uuid.Parse(dbIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid database ID",
		})
	}

	// Get the database
	db, err := h.servs.DatabasesService.GetDatabase(ctx, dbID)
	if err != nil {
		// Check if the error is due to database not found
		if err.Error() == "database not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Database not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get database: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, databaseResponse{
		ID:        db.ID.String(),
		Name:      db.Name,
		Version:   db.PgVersion,
		CreatedAt: db.CreatedAt.String(),
	})
}
