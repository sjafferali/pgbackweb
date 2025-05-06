package destinations

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h *handlers) ListDestinations(c echo.Context) error {
	ctx := c.Request().Context()

	destinations, err := h.servs.DestinationsServiceGetAllDestinations(ctx, h.env.PBW_ENCRYPTION_KEY)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	response := make([]destinationResponse, len(destinations))
	for i, dest := range destinations {
		response[i] = toDestinationResponse(dest)
	}

	return c.JSON(http.StatusOK, response)
}
