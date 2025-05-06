package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// DatabasesServiceInterface defines the interface for the DatabasesService
type DatabasesServiceInterface interface {
	TestDatabase(ctx context.Context, version, connString string) error
	CreateDatabase(ctx context.Context, params dbgen.DatabasesServiceCreateDatabaseParams) (dbgen.Database, error)
}

// MockDatabasesService is a mock implementation of the DatabasesServiceInterface
type MockDatabasesService struct {
	mock.Mock
}

func (m *MockDatabasesService) TestDatabase(ctx context.Context, version, connString string) error {
	args := m.Called(ctx, version, connString)
	return args.Error(0)
}

func (m *MockDatabasesService) CreateDatabase(ctx context.Context, params dbgen.DatabasesServiceCreateDatabaseParams) (dbgen.Database, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(dbgen.Database), args.Error(1)
}

// mockHandlers is a test version of handlers that accepts interfaces
type mockHandlers struct {
	servs *mockService
}

// mockService is a test version of service.Service that accepts interfaces
type mockService struct {
	DatabasesService DatabasesServiceInterface
}

// createDatabaseAPI is a copy of the original handler but using our mock types
func (h *mockHandlers) createDatabaseAPI(c echo.Context) error {
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

func TestCreateDatabaseAPI(t *testing.T) {
	// Setup
	e := echo.New()
	mockDBService := &MockDatabasesService{}

	// Create a mock service with our mock DatabasesService
	servs := &mockService{
		DatabasesService: mockDBService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Set test API key
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	tests := []struct {
		name           string
		apiKey         string
		requestBody    createDatabaseRequest
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "successful database creation",
			apiKey: "test-api-key",
			requestBody: createDatabaseRequest{
				Name:             "test-db",
				Version:          "15",
				ConnectionString: "postgresql://user:pass@localhost:5432/test",
			},
			mockSetup: func() {
				mockDBService.On("TestDatabase", mock.Anything, "15", "postgresql://user:pass@localhost:5432/test").Return(nil)
				mockDBService.On("CreateDatabase", mock.Anything, mock.MatchedBy(func(params dbgen.DatabasesServiceCreateDatabaseParams) bool {
					return params.Name == "test-db" && params.PgVersion == "15"
				})).Return(dbgen.Database{
					ID:        uuid.New(),
					Name:      "test-db",
					PgVersion: "15",
					CreatedAt: time.Now(),
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"name":    "test-db",
				"version": "15",
			},
		},
		{
			name:   "missing API key",
			apiKey: "",
			requestBody: createDatabaseRequest{
				Name:             "test-db",
				Version:          "15",
				ConnectionString: "postgresql://user:pass@localhost:5432/test",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:   "invalid API key",
			apiKey: "wrong-key",
			requestBody: createDatabaseRequest{
				Name:             "test-db",
				Version:          "15",
				ConnectionString: "postgresql://user:pass@localhost:5432/test",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:   "invalid version",
			apiKey: "test-api-key",
			requestBody: createDatabaseRequest{
				Name:             "test-db",
				Version:          "12", // Invalid version
				ConnectionString: "postgresql://user:pass@localhost:5432/test",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "error in field Version: oneof",
			},
		},
		{
			name:   "connection test failure",
			apiKey: "test-api-key",
			requestBody: createDatabaseRequest{
				Name:             "test-db",
				Version:          "15",
				ConnectionString: "postgresql://user:pass@localhost:5432/test",
			},
			mockSetup: func() {
				mockDBService.On("TestDatabase", mock.Anything, "15", "postgresql://user:pass@localhost:5432/test").Return(assert.AnError)
				// Don't set up CreateDatabase mock since we expect TestDatabase to fail
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Failed to connect to database: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.mockSetup()

			// Create request
			reqBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/databases", bytes.NewReader(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Test with middleware
			handler := APIKeyAuth()(h.createDatabaseAPI)
			err := handler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check expected fields
			for key, expectedValue := range tt.expectedBody {
				assert.Equal(t, expectedValue, response[key])
			}

			// Additional assertions for successful creation
			if tt.expectedStatus == http.StatusCreated {
				assert.NotEmpty(t, response["id"])
				assert.NotEmpty(t, response["created_at"])
			}

			// Reset mock for next test
			mockDBService.ExpectedCalls = nil
		})
	}
}
