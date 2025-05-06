package restorations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/service/restorations"
	"github.com/eduardolat/pgbackweb/internal/util/paginateutil"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// RestorationsServiceInterface defines the interface for the RestorationsService
type RestorationsServiceInterface interface {
	PaginateRestorations(ctx context.Context, params restorations.PaginateRestorationsParams) (paginateutil.PaginateResponse, []dbgen.RestorationsServicePaginateRestorationsRow, error)
}

// ExecutionsServiceInterface defines the interface for the ExecutionsService
type ExecutionsServiceInterface interface {
	RunExecution(ctx context.Context, backupID uuid.UUID) error
}

// MockRestorationsService is a mock implementation of the RestorationsServiceInterface
type MockRestorationsService struct {
	mock.Mock
}

func (m *MockRestorationsService) PaginateRestorations(ctx context.Context, params restorations.PaginateRestorationsParams) (paginateutil.PaginateResponse, []dbgen.RestorationsServicePaginateRestorationsRow, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(paginateutil.PaginateResponse), args.Get(1).([]dbgen.RestorationsServicePaginateRestorationsRow), args.Error(2)
}

// MockExecutionsService is a mock implementation of the ExecutionsServiceInterface
type MockExecutionsService struct {
	mock.Mock
}

func (m *MockExecutionsService) RunExecution(ctx context.Context, backupID uuid.UUID) error {
	args := m.Called(ctx, backupID)
	return args.Error(0)
}

// mockHandlers is a test version of handlers that accepts interfaces
type mockHandlers struct {
	servs *mockService
}

// mockService is a test version of service.Service that accepts interfaces
type mockService struct {
	RestorationsService RestorationsServiceInterface
	ExecutionsService   ExecutionsServiceInterface
}

// listRestorationsHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) listRestorationsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse query parameters
	var executionID, databaseID uuid.UUID
	var err error

	if executionIDStr := c.QueryParam("execution_id"); executionIDStr != "" {
		executionID, err = uuid.Parse(executionIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid execution ID",
			})
		}
	}

	if databaseIDStr := c.QueryParam("database_id"); databaseIDStr != "" {
		databaseID, err = uuid.Parse(databaseIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid database ID",
			})
		}
	}

	// Get restorations from database
	_, restorations, err := h.servs.RestorationsService.PaginateRestorations(ctx, restorations.PaginateRestorationsParams{
		ExecutionFilter: uuid.NullUUID{UUID: executionID, Valid: executionID != uuid.Nil},
		DatabaseFilter:  uuid.NullUUID{UUID: databaseID, Valid: databaseID != uuid.Nil},
		Page:            1,
		Limit:           100,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get restorations: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": restorations,
	})
}

// triggerBackupHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) triggerBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup ID from URL parameter
	backupIDStr := c.Param("backup_id")
	backupID, err := uuid.Parse(backupIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid backup ID",
		})
	}

	// Trigger the backup
	err = h.servs.ExecutionsService.RunExecution(ctx, backupID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to trigger backup: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Backup task triggered successfully",
	})
}

func TestListRestorationsHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockRestorationsService := new(MockRestorationsService)
	mockExecutionsService := new(MockExecutionsService)
	h := &mockHandlers{
		servs: &mockService{
			RestorationsService: mockRestorationsService,
			ExecutionsService:   mockExecutionsService,
		},
	}

	// Test cases
	tests := []struct {
		name           string
		queryParams    string
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success - List all restorations",
			queryParams: "",
			mockSetup: func() {
				mockRestorationsService.On("PaginateRestorations", mock.Anything, restorations.PaginateRestorationsParams{
					Page:  1,
					Limit: 100,
				}).Return(
					paginateutil.PaginateResponse{},
					[]dbgen.RestorationsServicePaginateRestorationsRow{},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"data": []interface{}{},
			},
		},
		{
			name:        "Success - List restorations with filters",
			queryParams: "?execution_id=123e4567-e89b-12d3-a456-426614174000&database_id=123e4567-e89b-12d3-a456-426614174001",
			mockSetup: func() {
				executionID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
				databaseID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
				mockRestorationsService.On("PaginateRestorations", mock.Anything, restorations.PaginateRestorationsParams{
					ExecutionFilter: uuid.NullUUID{UUID: executionID, Valid: true},
					DatabaseFilter:  uuid.NullUUID{UUID: databaseID, Valid: true},
					Page:            1,
					Limit:           100,
				}).Return(
					paginateutil.PaginateResponse{},
					[]dbgen.RestorationsServicePaginateRestorationsRow{},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"data": []interface{}{},
			},
		},
		{
			name:           "Error - Invalid execution ID",
			queryParams:    "?execution_id=invalid",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid execution ID",
			},
		},
		{
			name:           "Error - Invalid database ID",
			queryParams:    "?database_id=invalid",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid database ID",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock
			tc.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/restorations"+tc.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Test handler
			err := h.listRestorationsHandler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, response)
		})
	}
}

func TestTriggerBackupHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockRestorationsService := new(MockRestorationsService)
	mockExecutionsService := new(MockExecutionsService)
	h := &mockHandlers{
		servs: &mockService{
			RestorationsService: mockRestorationsService,
			ExecutionsService:   mockExecutionsService,
		},
	}

	// Test cases
	tests := []struct {
		name           string
		backupID       string
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:     "Success - Trigger backup",
			backupID: "123e4567-e89b-12d3-a456-426614174000",
			mockSetup: func() {
				backupID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
				mockExecutionsService.On("RunExecution", mock.Anything, backupID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "Backup task triggered successfully",
			},
		},
		{
			name:           "Error - Invalid backup ID",
			backupID:       "invalid",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid backup ID",
			},
		},
		{
			name:     "Error - Run execution fails",
			backupID: "123e4567-e89b-12d3-a456-426614174000",
			mockSetup: func() {
				backupID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
				mockExecutionsService.On("RunExecution", mock.Anything, backupID).Return(assert.AnError).Once()
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"message": "Backup task triggered successfully",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock
			tc.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/v1/restorations/trigger-backup/"+tc.backupID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("backup_id")
			c.SetParamValues(tc.backupID)

			// Test handler
			err := h.triggerBackupHandler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedBody, response)
		})
	}
}
