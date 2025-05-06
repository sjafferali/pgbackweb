package executions

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockExecutionsService is a mock implementation of the ExecutionsServiceInterface
type MockExecutionsService struct {
	mock.Mock
}

func (m *MockExecutionsService) ListBackupExecutions(ctx context.Context, backupID uuid.UUID) ([]dbgen.Execution, error) {
	args := m.Called(ctx, backupID)
	return args.Get(0).([]dbgen.Execution), args.Error(1)
}

func (m *MockExecutionsService) GetExecution(ctx context.Context, id uuid.UUID) (dbgen.ExecutionsServiceGetExecutionRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(dbgen.ExecutionsServiceGetExecutionRow), args.Error(1)
}

// mockHandlers is a test version of handlers that accepts interfaces
type mockHandlers struct {
	servs *mockExecutionService
}

// mockExecutionService is a test version of service.Service that accepts interfaces
type mockExecutionService struct {
	ExecutionsService ExecutionsServiceInterface
}

// listExecutionsHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) listExecutionsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup_id from query parameter if provided
	var backupID uuid.UUID
	if backupIDStr := c.QueryParam("backup_id"); backupIDStr != "" {
		id, err := uuid.Parse(backupIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid backup ID",
			})
		}
		backupID = id
	}

	// Get executions from database
	executions, err := h.servs.ExecutionsService.ListBackupExecutions(ctx, backupID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get executions: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, executions)
}

// getExecutionHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) getExecutionHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get execution ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid execution ID",
		})
	}

	// Get execution from database
	execution, err := h.servs.ExecutionsService.GetExecution(ctx, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get execution: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, execution)
}

func TestListExecutionsHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockExecutionsService)

	// Create a mock service with our mock ExecutionsService
	servs := &mockExecutionService{
		ExecutionsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	now := time.Date(2024, 5, 6, 8, 32, 42, 43834000, time.UTC)
	expectedExecutions := []dbgen.Execution{
		{
			ID:        uuid.New(),
			BackupID:  uuid.New(),
			Status:    "success",
			Message:   sql.NullString{String: "Backup completed successfully", Valid: true},
			Path:      sql.NullString{String: "/backups/backup1.dump", Valid: true},
			StartedAt: now.Add(-1 * time.Hour),
			UpdatedAt: sql.NullTime{Time: now, Valid: true},
		},
		{
			ID:        uuid.New(),
			BackupID:  uuid.New(),
			Status:    "running",
			Message:   sql.NullString{String: "Backup in progress", Valid: true},
			Path:      sql.NullString{Valid: false},
			StartedAt: now,
			UpdatedAt: sql.NullTime{Time: now, Valid: true},
		},
	}

	// Setup expectations
	mockService.On("ListBackupExecutions", mock.Anything, uuid.Nil).Return(expectedExecutions, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.listExecutionsHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []dbgen.Execution
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedExecutions, response)

	mockService.AssertExpectations(t)
}

func TestListExecutionsHandler_WithBackupID(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockExecutionsService)

	// Create a mock service with our mock ExecutionsService
	servs := &mockExecutionService{
		ExecutionsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	backupID := uuid.New()
	now := time.Date(2024, 5, 6, 8, 32, 42, 43834000, time.UTC)
	expectedExecutions := []dbgen.Execution{
		{
			ID:        uuid.New(),
			BackupID:  backupID,
			Status:    "success",
			Message:   sql.NullString{String: "Backup completed successfully", Valid: true},
			Path:      sql.NullString{String: "/backups/backup1.dump", Valid: true},
			StartedAt: now.Add(-1 * time.Hour),
			UpdatedAt: sql.NullTime{Time: now, Valid: true},
		},
	}

	// Setup expectations
	mockService.On("ListBackupExecutions", mock.Anything, backupID).Return(expectedExecutions, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/?backup_id="+backupID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.listExecutionsHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []dbgen.Execution
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedExecutions, response)

	mockService.AssertExpectations(t)
}

func TestListExecutionsHandler_InvalidBackupID(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockExecutionsService)

	// Create a mock service with our mock ExecutionsService
	servs := &mockExecutionService{
		ExecutionsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Create request with invalid backup ID
	req := httptest.NewRequest(http.MethodGet, "/?backup_id=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.listExecutionsHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid backup ID", response["error"])

	mockService.AssertExpectations(t)
}

func TestGetExecutionHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockExecutionsService)

	// Create a mock service with our mock ExecutionsService
	servs := &mockExecutionService{
		ExecutionsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	executionID := uuid.New()
	now := time.Date(2024, 5, 6, 8, 32, 42, 43834000, time.UTC)
	expectedExecution := dbgen.ExecutionsServiceGetExecutionRow{
		ID:                executionID,
		BackupID:          uuid.New(),
		Status:            "success",
		Message:           sql.NullString{String: "Backup completed successfully", Valid: true},
		Path:              sql.NullString{String: "/backups/backup1.dump", Valid: true},
		StartedAt:         now.Add(-1 * time.Hour),
		UpdatedAt:         sql.NullTime{Time: now, Valid: true},
		FinishedAt:        sql.NullTime{Time: now, Valid: true},
		DeletedAt:         sql.NullTime{Valid: false},
		FileSize:          sql.NullInt64{Int64: 1024, Valid: true},
		DatabaseID:        uuid.New(),
		DatabasePgVersion: "15",
	}

	// Setup expectations
	mockService.On("GetExecution", mock.Anything, executionID).Return(expectedExecution, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues(executionID.String())

	// Test
	err := h.getExecutionHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response dbgen.ExecutionsServiceGetExecutionRow
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedExecution, response)

	mockService.AssertExpectations(t)
}

func TestGetExecutionHandler_InvalidID(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockExecutionsService)

	// Create a mock service with our mock ExecutionsService
	servs := &mockExecutionService{
		ExecutionsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Create request with invalid execution ID
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Test
	err := h.getExecutionHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid execution ID", response["error"])

	mockService.AssertExpectations(t)
}

func TestGetExecutionHandler_Error(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockExecutionsService)

	// Create a mock service with our mock ExecutionsService
	servs := &mockExecutionService{
		ExecutionsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	executionID := uuid.New()

	// Setup expectations
	mockService.On("GetExecution", mock.Anything, executionID).Return(dbgen.ExecutionsServiceGetExecutionRow{}, assert.AnError)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues(executionID.String())

	// Test
	err := h.getExecutionHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to get execution: "+assert.AnError.Error(), response["error"])

	mockService.AssertExpectations(t)
}
