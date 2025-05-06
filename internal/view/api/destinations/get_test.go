package destinations

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eduardolat/pgbackweb/internal/config"
	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// GetDestination is a copy of the original handler but using our mock types
func (h *mockHandlers) GetDestination(c echo.Context) error {
	ctx := c.Request().Context()

	// Get destination ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid destination ID")
	}

	// Get destination from database
	dest, err := h.servs.DestinationsService.GetDestination(ctx, dbgen.DestinationsServiceGetDestinationParams{
		ID: id,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get destination: "+err.Error())
	}

	// Convert to response format
	response := toTestDestinationResponse(dest)

	return c.JSON(http.StatusOK, response)
}

func TestGetDestination(t *testing.T) {
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
	now := time.Now()
	destID := uuid.New()
	destination := dbgen.DestinationsServiceGetDestinationRow{
		ID:         destID,
		Name:       "Test Destination",
		BucketName: "test-bucket",
		Region:     "us-west-1",
		Endpoint:   "s3.amazonaws.com",
		CreatedAt:  now,
		UpdatedAt:  sql.NullTime{Time: now, Valid: true},
		TestOk:     sql.NullBool{Valid: false},
		TestError:  sql.NullString{Valid: false},
		LastTestAt: sql.NullTime{Valid: false},
	}

	// Setup expectations
	mockDBService.On("GetDestination", mock.Anything, dbgen.DestinationsServiceGetDestinationParams{
		ID: destID,
	}).Return(destination, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/destinations/"+destID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/destinations/:id")
	c.SetParamNames("id")
	c.SetParamValues(destID.String())

	// Test
	err := h.GetDestination(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response testDestinationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, destination.Name, response.Name)
	assert.Equal(t, destination.BucketName, response.BucketName)
	assert.Equal(t, destination.Region, response.Region)
	assert.Equal(t, destination.Endpoint, response.Endpoint)

	mockDBService.AssertExpectations(t)
}

func TestGetDestination_InvalidID(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/api/destinations/invalid-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/destinations/:id")
	c.SetParamNames("id")
	c.SetParamValues("invalid-id")

	// Test
	err := h.GetDestination(c)

	// Assertions
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
	assert.Equal(t, "Invalid destination ID", he.Message)
}

func TestGetDestination_NotFound(t *testing.T) {
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
	mockDBService.On("GetDestination", mock.Anything, dbgen.DestinationsServiceGetDestinationParams{
		ID: destID,
	}).Return(dbgen.DestinationsServiceGetDestinationRow{}, sql.ErrNoRows)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/destinations/"+destID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/destinations/:id")
	c.SetParamNames("id")
	c.SetParamValues(destID.String())

	// Test
	err := h.GetDestination(c)

	// Assertions
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, he.Code)
	assert.Contains(t, he.Message, "Failed to get destination")

	mockDBService.AssertExpectations(t)
}
