package destinations

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardolat/pgbackweb/internal/config"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// DeleteDestination is a copy of the original handler but using our mock types
func (h *mockHandlers) DeleteDestination(c echo.Context) error {
	ctx := c.Request().Context()

	// Get destination ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid destination ID")
	}

	// Delete destination from database
	err = h.servs.DestinationsService.DeleteDestination(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete destination: "+err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

func TestDeleteDestination(t *testing.T) {
	// Setup
	e := echo.New()
	mockDBService := &MockDestinationsService{}

	// Create a mock service with our mock DestinationsService
	servs := &mockService{
		DestinationsService: mockDBService,
	}

	// Create a handler with our mock service
	env := config.Env{PBW_ENCRYPTION_KEY: "test-key"}
	h := &mockHandlers{
		servs: servs,
		env:   env,
	}

	// Test data
	destID := uuid.New()

	// Setup expectations
	mockDBService.On("DeleteDestination", mock.Anything, destID).Return(nil)

	// Create request
	req := httptest.NewRequest(http.MethodDelete, "/api/destinations/"+destID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/destinations/:id")
	c.SetParamNames("id")
	c.SetParamValues(destID.String())

	// Test
	err := h.DeleteDestination(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	mockDBService.AssertExpectations(t)
}

func TestDeleteDestination_InvalidID(t *testing.T) {
	// Setup
	e := echo.New()
	mockDBService := &MockDestinationsService{}

	// Create a mock service with our mock DestinationsService
	servs := &mockService{
		DestinationsService: mockDBService,
	}

	// Create a handler with our mock service
	env := config.Env{PBW_ENCRYPTION_KEY: "test-key"}
	h := &mockHandlers{
		servs: servs,
		env:   env,
	}

	// Create request with invalid ID
	req := httptest.NewRequest(http.MethodDelete, "/api/destinations/invalid-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/destinations/:id")
	c.SetParamNames("id")
	c.SetParamValues("invalid-id")

	// Test
	err := h.DeleteDestination(c)

	// Assertions
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
	assert.Equal(t, "Invalid destination ID", he.Message)
}

func TestDeleteDestination_NotFound(t *testing.T) {
	// Setup
	e := echo.New()
	mockDBService := &MockDestinationsService{}

	// Create a mock service with our mock DestinationsService
	servs := &mockService{
		DestinationsService: mockDBService,
	}

	// Create a handler with our mock service
	env := config.Env{PBW_ENCRYPTION_KEY: "test-key"}
	h := &mockHandlers{
		servs: servs,
		env:   env,
	}

	// Test data
	destID := uuid.New()

	// Setup expectations
	mockDBService.On("DeleteDestination", mock.Anything, destID).Return(sql.ErrNoRows)

	// Create request
	req := httptest.NewRequest(http.MethodDelete, "/api/destinations/"+destID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/destinations/:id")
	c.SetParamNames("id")
	c.SetParamValues(destID.String())

	// Test
	err := h.DeleteDestination(c)

	// Assertions
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, he.Code)
	assert.Contains(t, he.Message, "Failed to delete destination")

	mockDBService.AssertExpectations(t)
}
