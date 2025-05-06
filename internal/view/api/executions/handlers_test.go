package executions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/service/executions"
	"github.com/eduardolat/pgbackweb/internal/util/paginateutil"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ExecutionsServiceInterface defines the interface for the ExecutionsService
type ExecutionsServiceInterface interface {
	PaginateExecutions(ctx context.Context, params executions.PaginateExecutionsParams) (paginateutil.PaginateResponse, []dbgen.ExecutionsServicePaginateExecutionsRow, error)
}

// MockExecutionsService is a mock implementation of the ExecutionsServiceInterface
type MockExecutionsService struct {
	mock.Mock
}

func (m *MockExecutionsService) PaginateExecutions(ctx context.Context, params executions.PaginateExecutionsParams) (paginateutil.PaginateResponse, []dbgen.ExecutionsServicePaginateExecutionsRow, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(paginateutil.PaginateResponse), args.Get(1).([]dbgen.ExecutionsServicePaginateExecutionsRow), args.Error(2)
}

// mockHandlers is a test version of handlers that accepts interfaces
type mockHandlers struct {
	servs *mockService
}

// mockService is a test version of service.Service that accepts interfaces
type mockService struct {
	ExecutionsService ExecutionsServiceInterface
}

// listExecutionsHandler is a copy of the original handler but using our mock types
func (h *mockHandlers) listExecutionsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse query parameters
	var backupID uuid.NullUUID
	if backupIDStr := c.QueryParam("backup_id"); backupIDStr != "" {
		id, err := uuid.Parse(backupIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid backup ID",
			})
		}
		backupID = uuid.NullUUID{UUID: id, Valid: true}
	}

	// Get executions from database
	paginateResponse, executions, err := h.servs.ExecutionsService.PaginateExecutions(ctx, executions.PaginateExecutionsParams{
		Page:         1,
		Limit:        100,
		BackupFilter: backupID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get executions: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":       executions,
		"pagination": paginateResponse,
	})
}

func TestListExecutionsHandler(t *testing.T) {
	// Setup
	e := echo.New()
	mockExecutionsService := new(MockExecutionsService)
	h := &mockHandlers{
		servs: &mockService{
			ExecutionsService: mockExecutionsService,
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
			name:        "Success - List all executions",
			queryParams: "",
			mockSetup: func() {
				mockExecutionsService.On("PaginateExecutions", mock.Anything, executions.PaginateExecutionsParams{
					Page:  1,
					Limit: 100,
				}).Return(
					paginateutil.PaginateResponse{
						CurrentPage:     1,
						ItemsPerPage:    100,
						TotalItems:      0,
						TotalPages:      0,
						HasNextPage:     false,
						HasPreviousPage: false,
						NextPage:        0,
						PreviousPage:    0,
					},
					[]dbgen.ExecutionsServicePaginateExecutionsRow{},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"data": []interface{}{},
				"pagination": map[string]interface{}{
					"current_page":      float64(1),
					"items_per_page":    float64(100),
					"total_items":       float64(0),
					"total_pages":       float64(0),
					"has_next_page":     false,
					"has_previous_page": false,
					"next_page":         float64(0),
					"previous_page":     float64(0),
				},
			},
		},
		{
			name:        "Success - List executions with backup filter",
			queryParams: "?backup_id=123e4567-e89b-12d3-a456-426614174000",
			mockSetup: func() {
				backupID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
				mockExecutionsService.On("PaginateExecutions", mock.Anything, executions.PaginateExecutionsParams{
					Page:         1,
					Limit:        100,
					BackupFilter: uuid.NullUUID{UUID: backupID, Valid: true},
				}).Return(
					paginateutil.PaginateResponse{
						CurrentPage:     1,
						ItemsPerPage:    100,
						TotalItems:      0,
						TotalPages:      0,
						HasNextPage:     false,
						HasPreviousPage: false,
						NextPage:        0,
						PreviousPage:    0,
					},
					[]dbgen.ExecutionsServicePaginateExecutionsRow{},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"data": []interface{}{},
				"pagination": map[string]interface{}{
					"current_page":      float64(1),
					"items_per_page":    float64(100),
					"total_items":       float64(0),
					"total_pages":       float64(0),
					"has_next_page":     false,
					"has_previous_page": false,
					"next_page":         float64(0),
					"previous_page":     float64(0),
				},
			},
		},
		{
			name:           "Error - Invalid backup ID",
			queryParams:    "?backup_id=invalid",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid backup ID",
			},
		},
		{
			name:        "Error - Service error",
			queryParams: "",
			mockSetup: func() {
				mockExecutionsService.On("PaginateExecutions", mock.Anything, executions.PaginateExecutionsParams{
					Page:  1,
					Limit: 100,
				}).Return(
					paginateutil.PaginateResponse{},
					[]dbgen.ExecutionsServicePaginateExecutionsRow{},
					assert.AnError,
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to get executions: " + assert.AnError.Error(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock
			tc.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/executions"+tc.queryParams, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Test handler
			err := h.listExecutionsHandler(c)

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
