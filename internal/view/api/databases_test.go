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
	DeleteDatabase(ctx context.Context, id uuid.UUID) error
	GetAllDatabases(ctx context.Context) ([]dbgen.DatabasesServiceGetAllDatabasesRow, error)
	GetDatabase(ctx context.Context, id uuid.UUID) (dbgen.DatabasesServiceGetDatabaseRow, error)
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

func (m *MockDatabasesService) DeleteDatabase(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabasesService) GetAllDatabases(ctx context.Context) ([]dbgen.DatabasesServiceGetAllDatabasesRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]dbgen.DatabasesServiceGetAllDatabasesRow), args.Error(1)
}

func (m *MockDatabasesService) GetDatabase(ctx context.Context, id uuid.UUID) (dbgen.DatabasesServiceGetDatabaseRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(dbgen.DatabasesServiceGetDatabaseRow), args.Error(1)
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

// deleteDatabaseAPI is a copy of the original handler but using our mock types
func (h *mockHandlers) deleteDatabaseAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get database ID from URL parameter
	dbIDStr := c.Param("id")
	dbID, err := uuid.Parse(dbIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid database ID",
		})
	}

	// Delete the database
	err = h.servs.DatabasesService.DeleteDatabase(ctx, dbID)
	if err != nil {
		// Check if the error is due to database not found
		if err.Error() == "database not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Database not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete database: " + err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// listDatabasesAPI is a copy of the original handler but using our mock types
func (h *mockHandlers) listDatabasesAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get all databases
	databases, err := h.servs.DatabasesService.GetAllDatabases(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list databases: " + err.Error(),
		})
	}

	// Convert to response format
	response := make([]databaseResponse, len(databases))
	for i, db := range databases {
		response[i] = databaseResponse{
			ID:        db.ID.String(),
			Name:      db.Name,
			Version:   db.PgVersion,
			CreatedAt: db.CreatedAt.String(),
		}
	}

	return c.JSON(http.StatusOK, response)
}

// getDatabaseAPI is a copy of the original handler but using our mock types
func (h *mockHandlers) getDatabaseAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get database ID from URL parameter
	dbIDStr := c.Param("id")
	dbID, err := uuid.Parse(dbIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid database ID",
		})
	}

	// Get the database
	db, err := h.servs.DatabasesService.GetDatabase(ctx, dbID)
	if err != nil {
		// Check if the error is due to database not found
		if err.Error() == "database not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Database not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get database: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, databaseResponse{
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

func TestDeleteDatabaseAPI(t *testing.T) {
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
		dbID           string
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "successful database deletion",
			apiKey: "test-api-key",
			dbID:   uuid.New().String(),
			mockSetup: func() {
				mockDBService.On("DeleteDatabase", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   nil,
		},
		{
			name:           "missing API key",
			apiKey:         "",
			dbID:           uuid.New().String(),
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:           "invalid API key",
			apiKey:         "wrong-key",
			dbID:           uuid.New().String(),
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:           "invalid database ID",
			apiKey:         "test-api-key",
			dbID:           "invalid-uuid",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid database ID",
			},
		},
		{
			name:   "database not found",
			apiKey: "test-api-key",
			dbID:   uuid.New().String(),
			mockSetup: func() {
				mockDBService.On("DeleteDatabase", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(
					assert.AnError,
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to delete database: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/databases/"+tt.dbID, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/databases/:id")
			c.SetParamNames("id")
			c.SetParamValues(tt.dbID)

			// Test with middleware
			handler := APIKeyAuth()(h.deleteDatabaseAPI)
			err := handler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)

				// Check expected fields
				for key, expectedValue := range tt.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}
			} else {
				assert.Empty(t, rec.Body.String())
			}

			// Reset mock for next test
			mockDBService.ExpectedCalls = nil
		})
	}
}

func TestListDatabasesAPI(t *testing.T) {
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
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:   "successful database list",
			apiKey: "test-api-key",
			mockSetup: func() {
				mockDBService.On("GetAllDatabases", mock.Anything).Return([]dbgen.DatabasesServiceGetAllDatabasesRow{
					{
						ID:        uuid.New(),
						Name:      "test-db-1",
						PgVersion: "15",
						CreatedAt: time.Now(),
					},
					{
						ID:        uuid.New(),
						Name:      "test-db-2",
						PgVersion: "16",
						CreatedAt: time.Now(),
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: []databaseResponse{
				{
					Name:    "test-db-1",
					Version: "15",
				},
				{
					Name:    "test-db-2",
					Version: "16",
				},
			},
		},
		{
			name:           "missing API key",
			apiKey:         "",
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:           "invalid API key",
			apiKey:         "wrong-key",
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:   "list error",
			apiKey: "test-api-key",
			mockSetup: func() {
				mockDBService.On("GetAllDatabases", mock.Anything).Return([]dbgen.DatabasesServiceGetAllDatabasesRow{}, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to list databases: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/databases", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Test with middleware
			handler := APIKeyAuth()(h.listDatabasesAPI)
			err := handler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check expected fields
			if tt.expectedStatus == http.StatusOK {
				// For successful list, check each database in the response
				responseArray := response.([]interface{})
				expectedArray := tt.expectedBody.([]databaseResponse)
				assert.Equal(t, len(expectedArray), len(responseArray))

				for i, expectedDB := range expectedArray {
					responseDB := responseArray[i].(map[string]interface{})
					assert.Equal(t, expectedDB.Name, responseDB["name"])
					assert.Equal(t, expectedDB.Version, responseDB["version"])
					assert.NotEmpty(t, responseDB["id"])
					assert.NotEmpty(t, responseDB["created_at"])
				}
			} else {
				// For error responses, check the error message
				expectedError := tt.expectedBody.(map[string]interface{})
				responseError := response.(map[string]interface{})
				assert.Equal(t, expectedError["error"], responseError["error"])
			}

			// Reset mock for next test
			mockDBService.ExpectedCalls = nil
		})
	}
}

func TestGetDatabaseAPI(t *testing.T) {
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
		dbID           string
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:   "successful database get",
			apiKey: "test-api-key",
			dbID:   uuid.New().String(),
			mockSetup: func() {
				mockDBService.On("GetDatabase", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(
					dbgen.DatabasesServiceGetDatabaseRow{
						ID:        uuid.New(),
						Name:      "test-db",
						PgVersion: "15",
						CreatedAt: time.Now(),
					}, nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: databaseResponse{
				Name:    "test-db",
				Version: "15",
			},
		},
		{
			name:           "missing API key",
			apiKey:         "",
			dbID:           uuid.New().String(),
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:           "invalid API key",
			apiKey:         "wrong-key",
			dbID:           uuid.New().String(),
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:           "invalid database ID",
			apiKey:         "test-api-key",
			dbID:           "invalid-uuid",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid database ID",
			},
		},
		{
			name:   "database not found",
			apiKey: "test-api-key",
			dbID:   uuid.New().String(),
			mockSetup: func() {
				mockDBService.On("GetDatabase", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(
					dbgen.DatabasesServiceGetDatabaseRow{}, assert.AnError,
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to get database: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/"+tt.dbID, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/databases/:id")
			c.SetParamNames("id")
			c.SetParamValues(tt.dbID)

			// Test with middleware
			handler := APIKeyAuth()(h.getDatabaseAPI)
			err := handler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check expected fields
			if tt.expectedStatus == http.StatusOK {
				// For successful get, check the database fields
				expectedDB := tt.expectedBody.(databaseResponse)
				responseDB := response.(map[string]interface{})
				assert.Equal(t, expectedDB.Name, responseDB["name"])
				assert.Equal(t, expectedDB.Version, responseDB["version"])
				assert.NotEmpty(t, responseDB["id"])
				assert.NotEmpty(t, responseDB["created_at"])
			} else {
				// For error responses, check the error message
				expectedError := tt.expectedBody.(map[string]interface{})
				responseError := response.(map[string]interface{})
				assert.Equal(t, expectedError["error"], responseError["error"])
			}

			// Reset mock for next test
			mockDBService.ExpectedCalls = nil
		})
	}
}
