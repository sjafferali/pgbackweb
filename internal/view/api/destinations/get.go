package destinations

import (
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *handlers) GetDestination(c echo.Context) error {
	ctx := c.Request().Context()

	// Get destination ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid destination ID")
	}

	// Get destination from database
	dest, err := h.servs.DestinationsServiceGetDestination(ctx, dbgen.DestinationsServiceGetDestinationParams{
		ID: id,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get destination: "+err.Error())
	}

	// Convert to response format
	response := toDestinationResponse(dest)

	return c.JSON(http.StatusOK, response)
}
