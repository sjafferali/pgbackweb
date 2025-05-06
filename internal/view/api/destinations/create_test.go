package destinations

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eduardolat/pgbackweb/internal/config"
	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// CustomValidator is a custom validator for Echo
type CustomValidator struct {
	validator *validator.Validate
}

// Validate implements the echo.Validator interface
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// CreateDestination is a copy of the original handler but using our mock types
func (h *mockHandlers) CreateDestination(c echo.Context) error {
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
	dest, err := h.servs.DestinationsService.CreateDestination(ctx, dbgen.DestinationsServiceCreateDestinationParams{
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
	response := toTestDestinationResponse(dest)

	return c.JSON(http.StatusCreated, response)
}

func TestCreateDestination(t *testing.T) {
	// Setup
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
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
	now := time.Now()
	reqBody := createDestinationRequest{
		Name:       "Test Destination",
		BucketName: "test-bucket",
		AccessKey:  "test-access-key",
		SecretKey:  "test-secret-key",
		Region:     "us-east-1",
		Endpoint:   "https://s3.amazonaws.com",
	}

	expectedDest := dbgen.Destination{
		ID:         destID,
		Name:       reqBody.Name,
		BucketName: reqBody.BucketName,
		AccessKey:  []byte("encrypted-access-key"),
		SecretKey:  []byte("encrypted-secret-key"),
		Region:     reqBody.Region,
		Endpoint:   reqBody.Endpoint,
		CreatedAt:  now,
		UpdatedAt:  sql.NullTime{Time: now, Valid: true},
	}

	// Setup expectations
	mockDBService.On("CreateDestination", mock.Anything, dbgen.DestinationsServiceCreateDestinationParams{
		Name:       reqBody.Name,
		BucketName: reqBody.BucketName,
		AccessKey:  reqBody.AccessKey,
		SecretKey:  reqBody.SecretKey,
		Region:     reqBody.Region,
		Endpoint:   reqBody.Endpoint,
	}).Return(expectedDest, nil)

	// Create request
	reqBodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/destinations", bytes.NewReader(reqBodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.CreateDestination(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response testDestinationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, destID, response.ID)
	assert.Equal(t, reqBody.Name, response.Name)
	assert.Equal(t, reqBody.BucketName, response.BucketName)
	assert.Equal(t, reqBody.Region, response.Region)
	assert.Equal(t, reqBody.Endpoint, response.Endpoint)

	mockDBService.AssertExpectations(t)
}

func TestCreateDestination_InvalidRequest(t *testing.T) {
	// Setup
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
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

	// Test data - missing required fields
	reqBody := createDestinationRequest{
		Name: "Test Destination",
		// Missing other required fields
	}

	// Create request
	reqBodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/destinations", bytes.NewReader(reqBodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.CreateDestination(c)

	// Assertions
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
	assert.Contains(t, he.Message, "Invalid request")
}

func TestCreateDestination_DatabaseError(t *testing.T) {
	// Setup
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
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
	reqBody := createDestinationRequest{
		Name:       "Test Destination",
		BucketName: "test-bucket",
		AccessKey:  "test-access-key",
		SecretKey:  "test-secret-key",
		Region:     "us-east-1",
		Endpoint:   "https://s3.amazonaws.com",
	}

	// Setup expectations
	mockDBService.On("CreateDestination", mock.Anything, dbgen.DestinationsServiceCreateDestinationParams{
		Name:       reqBody.Name,
		BucketName: reqBody.BucketName,
		AccessKey:  reqBody.AccessKey,
		SecretKey:  reqBody.SecretKey,
		Region:     reqBody.Region,
		Endpoint:   reqBody.Endpoint,
	}).Return(dbgen.Destination{}, assert.AnError)

	// Create request
	reqBodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/destinations", bytes.NewReader(reqBodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.CreateDestination(c)

	// Assertions
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, he.Code)
	assert.Contains(t, he.Message, "Failed to create destination")

	mockDBService.AssertExpectations(t)
}
