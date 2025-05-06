package destinations

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *handlers) DeleteDestination(c echo.Context) error {
	ctx := c.Request().Context()

	// Get destination ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid destination ID")
	}

	// Delete destination from database
	err = h.servs.DestinationsServiceDeleteDestination(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete destination: "+err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}
