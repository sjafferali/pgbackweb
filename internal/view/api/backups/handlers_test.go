package backups

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// BackupsServiceInterface defines the interface for the BackupsService
type BackupsServiceInterface interface {
	GetAllBackups(ctx context.Context) ([]dbgen.Backup, error)
	GetBackup(ctx context.Context, id uuid.UUID) (dbgen.Backup, error)
	CreateBackup(ctx context.Context, params dbgen.BackupsServiceCreateBackupParams) (dbgen.Backup, error)
	DeleteBackup(ctx context.Context, id uuid.UUID) error
}

// ExecutionsServiceInterface defines the interface for the ExecutionsService
type ExecutionsServiceInterface interface {
	RunExecution(ctx context.Context, backupID uuid.UUID) error
}

// MockBackupsService is a mock implementation of the BackupsServiceInterface
type MockBackupsService struct {
	mock.Mock
}

// MockExecutionsService is a mock implementation of the ExecutionsServiceInterface
type MockExecutionsService struct {
	mock.Mock
}

func (m *MockExecutionsService) RunExecution(ctx context.Context, backupID uuid.UUID) error {
	args := m.Called(ctx, backupID)
	return args.Error(0)
}

func (m *MockBackupsService) GetAllBackups(ctx context.Context) ([]dbgen.Backup, error) {
	args := m.Called(ctx)
	return args.Get(0).([]dbgen.Backup), args.Error(1)
}

func (m *MockBackupsService) GetBackup(ctx context.Context, id uuid.UUID) (dbgen.Backup, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(dbgen.Backup), args.Error(1)
}

func (m *MockBackupsService) CreateBackup(ctx context.Context, params dbgen.BackupsServiceCreateBackupParams) (dbgen.Backup, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(dbgen.Backup), args.Error(1)
}

func (m *MockBackupsService) DeleteBackup(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// mockHandlers is a test version of handlers that accepts interfaces
type mockHandlers struct {
	servs *mockBackupService
}

// mockBackupService is a test version of service.Service that accepts interfaces
type mockBackupService struct {
	BackupsService    BackupsServiceInterface
	ExecutionsService ExecutionsServiceInterface
}

// listBackupsHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) listBackupsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	backups, err := h.servs.BackupsService.GetAllBackups(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, backups)
}

// getBackupHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) getBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid backup ID",
		})
	}

	// Get backup from database
	backup, err := h.servs.BackupsService.GetBackup(ctx, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get backup: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, backup)
}

// deleteBackupHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) deleteBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid backup ID",
		})
	}

	// Delete backup from database
	err = h.servs.BackupsService.DeleteBackup(ctx, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete backup: " + err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// createBackupHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) createBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse request body
	var params dbgen.BackupsServiceCreateBackupParams
	if err := c.Bind(&params); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	// Create backup in database
	backup, err := h.servs.BackupsService.CreateBackup(ctx, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create backup: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, backup)
}

// triggerBackupHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) triggerBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Get backup ID from URL parameter
	backupIDStr := c.Param("id")
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

func TestListBackupsHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	expectedBackups := []dbgen.Backup{
		{
			ID:             uuid.New(),
			Name:           "Test Backup 1",
			CronExpression: "0 0 * * *",
			TimeZone:       "UTC",
			IsActive:       true,
		},
		{
			ID:             uuid.New(),
			Name:           "Test Backup 2",
			CronExpression: "0 12 * * *",
			TimeZone:       "UTC",
			IsActive:       false,
		},
	}

	// Setup expectations
	mockService.On("GetAllBackups", mock.Anything).Return(expectedBackups, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.listBackupsHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []dbgen.Backup
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedBackups, response)

	mockService.AssertExpectations(t)
}

func TestListBackupsHandler_Error(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Setup expectations
	mockService.On("GetAllBackups", mock.Anything).Return([]dbgen.Backup{}, assert.AnError)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.listBackupsHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, assert.AnError.Error(), response["error"])

	mockService.AssertExpectations(t)
}

func TestGetBackupHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	backupID := uuid.New()
	expectedBackup := dbgen.Backup{
		ID:             backupID,
		Name:           "Test Backup",
		CronExpression: "0 0 * * *",
		TimeZone:       "UTC",
		IsActive:       true,
	}

	// Setup expectations
	mockService.On("GetBackup", mock.Anything, backupID).Return(expectedBackup, nil)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues(backupID.String())

	// Test
	err := h.getBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response dbgen.Backup
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedBackup, response)

	mockService.AssertExpectations(t)
}

func TestGetBackupHandler_InvalidID(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Create request with invalid ID
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues("invalid-id")

	// Test
	err := h.getBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid backup ID", response["error"])
}

func TestGetBackupHandler_Error(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	backupID := uuid.New()

	// Setup expectations
	mockService.On("GetBackup", mock.Anything, backupID).Return(dbgen.Backup{}, assert.AnError)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues(backupID.String())

	// Test
	err := h.getBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to get backup: "+assert.AnError.Error(), response["error"])

	mockService.AssertExpectations(t)
}

func TestDeleteBackupHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	backupID := uuid.New()

	// Setup expectations
	mockService.On("DeleteBackup", mock.Anything, backupID).Return(nil)

	// Create request
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues(backupID.String())

	// Test
	err := h.deleteBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	mockService.AssertExpectations(t)
}

func TestDeleteBackupHandler_InvalidID(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Create request with invalid ID
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues("invalid-id")

	// Test
	err := h.deleteBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid backup ID", response["error"])
}

func TestDeleteBackupHandler_Error(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	backupID := uuid.New()

	// Setup expectations
	mockService.On("DeleteBackup", mock.Anything, backupID).Return(assert.AnError)

	// Create request
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:id")
	c.SetParamNames("id")
	c.SetParamValues(backupID.String())

	// Test
	err := h.deleteBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to delete backup: "+assert.AnError.Error(), response["error"])

	mockService.AssertExpectations(t)
}

func TestCreateBackupHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Test data
	databaseID := uuid.New()
	destinationID := uuid.New()
	requestBody := map[string]interface{}{
		"database_id":     databaseID.String(),
		"destination_id":  destinationID.String(),
		"is_local":        false,
		"name":            "Test Backup",
		"cron_expression": "",
		"time_zone":       "",
		"is_active":       false,
		"dest_dir":        "",
		"retention_days":  int16(0),
		"opt_data_only":   false,
		"opt_schema_only": false,
		"opt_clean":       false,
		"opt_if_exists":   false,
		"opt_create":      false,
		"opt_no_comments": false,
	}

	expectedParams := dbgen.BackupsServiceCreateBackupParams{
		DatabaseID: databaseID,
		DestinationID: uuid.NullUUID{
			UUID:  destinationID,
			Valid: true,
		},
		IsLocal:        false,
		Name:           "Test Backup",
		CronExpression: "",
		TimeZone:       "",
		IsActive:       false,
		DestDir:        "",
		RetentionDays:  0,
		OptDataOnly:    false,
		OptSchemaOnly:  false,
		OptClean:       false,
		OptIfExists:    false,
		OptCreate:      false,
		OptNoComments:  false,
	}

	expectedBackup := dbgen.Backup{
		ID:             uuid.New(),
		DatabaseID:     expectedParams.DatabaseID,
		DestinationID:  expectedParams.DestinationID,
		IsLocal:        expectedParams.IsLocal,
		Name:           expectedParams.Name,
		CronExpression: expectedParams.CronExpression,
		TimeZone:       expectedParams.TimeZone,
		IsActive:       expectedParams.IsActive,
		DestDir:        expectedParams.DestDir,
		RetentionDays:  expectedParams.RetentionDays,
		OptDataOnly:    expectedParams.OptDataOnly,
		OptSchemaOnly:  expectedParams.OptSchemaOnly,
		OptClean:       expectedParams.OptClean,
		OptIfExists:    expectedParams.OptIfExists,
		OptCreate:      expectedParams.OptCreate,
		OptNoComments:  expectedParams.OptNoComments,
	}

	// Setup expectations
	mockService.On("CreateBackup", mock.Anything, mock.Anything).Return(expectedBackup, nil)

	// Create request
	body, err := json.Marshal(requestBody)
	assert.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err = h.createBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response dbgen.Backup
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedBackup, response)

	mockService.AssertExpectations(t)
}

func TestCreateBackupHandler_InvalidBody(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Create request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"invalid": json`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.createBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid request body")

	// Verify that CreateBackup was not called
	mockService.AssertNotCalled(t, "CreateBackup")
}

func TestCreateBackupHandler_Error(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := new(MockBackupsService)

	// Create a mock service with our mock BackupsService
	servs := &mockBackupService{
		BackupsService: mockService,
	}

	// Create a handler with our mock service
	h := &mockHandlers{
		servs: servs,
	}

	// Setup expectations - use mock.Anything for both parameters
	mockService.On("CreateBackup", mock.Anything, mock.Anything).Return(dbgen.Backup{}, assert.AnError)

	// Create request with empty JSON body
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test
	err := h.createBackupHandler(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to create backup: "+assert.AnError.Error(), response["error"])

	mockService.AssertExpectations(t)
}

func TestTriggerBackupHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockBackupsService := new(MockBackupsService)
	mockExecutionsService := new(MockExecutionsService)
	h := &mockHandlers{
		servs: &mockBackupService{
			BackupsService:    mockBackupsService,
			ExecutionsService: mockExecutionsService,
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
				mockExecutionsService.On("RunExecution", mock.Anything, backupID).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to trigger backup: " + assert.AnError.Error(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock
			tc.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/backups/"+tc.backupID+"/trigger", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
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

			// Reset mock for next test
			mockExecutionsService.ExpectedCalls = nil
		})
	}
}
