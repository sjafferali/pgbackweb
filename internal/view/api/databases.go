package api

import (
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/validate"
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
