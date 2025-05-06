package destinations

import (
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/labstack/echo/v4"
)

func (h *handlers) CreateDestination(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse request body
	var req createDestinationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request: "+err.Error())
	}

	// Create destination in database
	dest, err := h.servs.DestinationsServiceCreateDestination(ctx, dbgen.DestinationsServiceCreateDestinationParams{
		Name:       req.Name,
		BucketName: req.BucketName,
		AccessKey:  req.AccessKey,
		SecretKey:  req.SecretKey,
		Region:     req.Region,
		Endpoint:   req.Endpoint,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create destination: "+err.Error())
	}

	// Convert to response format
	response := toDestinationResponse(dest)

	return c.JSON(http.StatusCreated, response)
}
