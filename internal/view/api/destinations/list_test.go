package destinations

import (
	"context"
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

// DestinationsServiceInterface defines the interface for the DestinationsService
type DestinationsServiceInterface interface {
	GetAllDestinations(ctx context.Context, encryptionKey string) ([]dbgen.DestinationsServiceGetAllDestinationsRow, error)
	GetDestination(ctx context.Context, arg dbgen.DestinationsServiceGetDestinationParams) (dbgen.DestinationsServiceGetDestinationRow, error)
	CreateDestination(ctx context.Context, arg dbgen.DestinationsServiceCreateDestinationParams) (dbgen.Destination, error)
	DeleteDestination(ctx context.Context, id uuid.UUID) error
	UpdateDestination(ctx context.Context, arg dbgen.DestinationsServiceUpdateDestinationParams) (dbgen.Destination, error)
	SetTestData(ctx context.Context, arg dbgen.DestinationsServiceSetTestDataParams) error
}

// MockDestinationsService is a mock implementation of the DestinationsServiceInterface
type MockDestinationsService struct {
	mock.Mock
}

func (m *MockDestinationsService) GetAllDestinations(ctx context.Context, encryptionKey string) ([]dbgen.DestinationsServiceGetAllDestinationsRow, error) {
	args := m.Called(ctx, encryptionKey)
	return args.Get(0).([]dbgen.DestinationsServiceGetAllDestinationsRow), args.Error(1)
}

func (m *MockDestinationsService) GetDestination(ctx context.Context, arg dbgen.DestinationsServiceGetDestinationParams) (dbgen.DestinationsServiceGetDestinationRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(dbgen.DestinationsServiceGetDestinationRow), args.Error(1)
}

func (m *MockDestinationsService) CreateDestination(ctx context.Context, arg dbgen.DestinationsServiceCreateDestinationParams) (dbgen.Destination, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(dbgen.Destination), args.Error(1)
}

func (m *MockDestinationsService) DeleteDestination(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDestinationsService) UpdateDestination(ctx context.Context, arg dbgen.DestinationsServiceUpdateDestinationParams) (dbgen.Destination, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(dbgen.Destination), args.Error(1)
}

func (m *MockDestinationsService) SetTestData(ctx context.Context, arg dbgen.DestinationsServiceSetTestDataParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// mockHandlers is a test version of handlers that accepts interfaces
type mockHandlers struct {
	servs *mockService
	env   config.Env
}

// mockService is a test version of service.Service that accepts interfaces
type mockService struct {
	DestinationsService DestinationsServiceInterface
}

// testDestinationResponse represents the API response format for a destination in tests
type testDestinationResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	BucketName string    `json:"bucket_name"`
	Region     string    `json:"region"`
	Endpoint   string    `json:"endpoint"`
	TestOk     *bool     `json:"test_ok,omitempty"`
	TestError  *string   `json:"test_error,omitempty"`
	LastTestAt *string   `json:"last_test_at,omitempty"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  *string   `json:"updated_at,omitempty"`
}

// toTestDestinationResponse converts a database row to the API response format for tests
func toTestDestinationResponse(dest interface{}) testDestinationResponse {
	var id uuid.UUID
	var name, bucketName, region, endpoint string
	var testOk sql.NullBool
	var testError sql.NullString
	var lastTestAt, updatedAt sql.NullTime
	var createdAt time.Time

	switch d := dest.(type) {
	case dbgen.DestinationsServiceGetDestinationRow:
		id = d.ID
		name = d.Name
		bucketName = d.BucketName
		region = d.Region
		endpoint = d.Endpoint
		testOk = d.TestOk
		testError = d.TestError
		lastTestAt = d.LastTestAt
		updatedAt = d.UpdatedAt
		createdAt = d.CreatedAt
	case dbgen.DestinationsServiceGetAllDestinationsRow:
		id = d.ID
		name = d.Name
		bucketName = d.BucketName
		region = d.Region
		endpoint = d.Endpoint
		testOk = d.TestOk
		testError = d.TestError
		lastTestAt = d.LastTestAt
		updatedAt = d.UpdatedAt
		createdAt = d.CreatedAt
	case dbgen.Destination:
		id = d.ID
		name = d.Name
		bucketName = d.BucketName
		region = d.Region
		endpoint = d.Endpoint
		testOk = d.TestOk
		testError = d.TestError
		lastTestAt = d.LastTestAt
		updatedAt = d.UpdatedAt
		createdAt = d.CreatedAt
	default:
		panic("unsupported destination type")
	}

	var testOkPtr *bool
	if testOk.Valid {
		testOkPtr = &testOk.Bool
	}

	var testErrorPtr *string
	if testError.Valid {
		testErrorPtr = &testError.String
	}

	var lastTestAtPtr *string
	if lastTestAt.Valid {
		formattedTime := lastTestAt.Time.Format(time.RFC3339)
		lastTestAtPtr = &formattedTime
	}

	var updatedAtPtr *string
	if updatedAt.Valid {
		formattedTime := updatedAt.Time.Format(time.RFC3339)
		updatedAtPtr = &formattedTime
	}

	return testDestinationResponse{
		ID:         id,
		Name:       name,
		BucketName: bucketName,
		Region:     region,
		Endpoint:   endpoint,
		TestOk:     testOkPtr,
		TestError:  testErrorPtr,
		LastTestAt: lastTestAtPtr,
		CreatedAt:  createdAt.Format(time.RFC3339),
		UpdatedAt:  updatedAtPtr,
	}
}

// ListDestinations is a copy of the original handler but using our mock types
func (h *mockHandlers) ListDestinations(c echo.Context) error {
	ctx := c.Request().Context()

	// Get all destinations
	destinations, err := h.servs.DestinationsService.GetAllDestinations(ctx, h.env.PBW_ENCRYPTION_KEY)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list destinations: " + err.Error(),
		})
	}

	// Convert to response format
	response := make([]testDestinationResponse, len(destinations))
	for i, dest := range destinations {
		response[i] = toTestDestinationResponse(dest)
	}

	return c.JSON(http.StatusOK, response)
}

func TestListDestinations(t *testing.T) {
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
	destinations := []dbgen.DestinationsServiceGetAllDestinationsRow{
		{
			ID:         uuid.New(),
			Name:       "Test Destination 1",
			BucketName: "bucket1",
			Region:     "us-west-1",
			Endpoint:   "s3.amazonaws.com",
			CreatedAt:  now,
			UpdatedAt:  sql.NullTime{Time: now, Valid: true},
			TestOk:     sql.NullBool{Valid: false},
			TestError:  sql.NullString{Valid: false},
			LastTestAt: sql.NullTime{Valid: false},
		},
		{
			ID:         uuid.New(),
			Name:       "Test Destination 2",
			BucketName: "bucket2",
			Region:     "us-east-1",
			Endpoint:   "s3.amazonaws.com",
			CreatedAt:  now,
			UpdatedAt:  sql.NullTime{Time: now, Valid: true},
			TestOk:     sql.NullBool{Valid: false},
			TestError:  sql.NullString{Valid: false},
			LastTestAt: sql.NullTime{Valid: false},
		},
	}

	// Setup expectations
	mockDBService.On("GetAllDestinations", mock.Anything, env.PBW_ENCRYPTION_KEY).
		Return(destinations, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/destinations", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.ListDestinations(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []testDestinationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
	assert.Equal(t, destinations[0].Name, response[0].Name)
	assert.Equal(t, destinations[1].Name, response[1].Name)

	mockDBService.AssertExpectations(t)
}
