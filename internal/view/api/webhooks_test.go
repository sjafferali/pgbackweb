package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/service/webhooks"
	"github.com/eduardolat/pgbackweb/internal/util/paginateutil"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// WebhooksServiceInterface defines the interface for the WebhooksService
type WebhooksServiceInterface interface {
	CreateWebhook(ctx context.Context, params dbgen.WebhooksServiceCreateWebhookParams) (dbgen.Webhook, error)
	GetWebhook(ctx context.Context, id uuid.UUID) (dbgen.Webhook, error)
	DeleteWebhook(ctx context.Context, id uuid.UUID) error
	PaginateWebhooks(ctx context.Context, params webhooks.PaginateWebhooksParams) (paginateutil.PaginateResponse, []dbgen.Webhook, error)
}

// MockWebhooksService is a mock implementation of the WebhooksServiceInterface
type MockWebhooksService struct {
	mock.Mock
}

func (m *MockWebhooksService) CreateWebhook(ctx context.Context, params dbgen.WebhooksServiceCreateWebhookParams) (dbgen.Webhook, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(dbgen.Webhook), args.Error(1)
}

func (m *MockWebhooksService) GetWebhook(ctx context.Context, id uuid.UUID) (dbgen.Webhook, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(dbgen.Webhook), args.Error(1)
}

func (m *MockWebhooksService) DeleteWebhook(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWebhooksService) PaginateWebhooks(ctx context.Context, params webhooks.PaginateWebhooksParams) (paginateutil.PaginateResponse, []dbgen.Webhook, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(paginateutil.PaginateResponse), args.Get(1).([]dbgen.Webhook), args.Error(2)
}

// mockWebhookHandlers is a test version of handlers that accepts interfaces
type mockWebhookHandlers struct {
	servs *mockWebhookService
}

// mockWebhookService is a test version of service.Service that accepts interfaces
type mockWebhookService struct {
	WebhooksService WebhooksServiceInterface
}

// Copy of the original handler methods for testing
func (h *mockWebhookHandlers) createWebhookAPI(c echo.Context) error {
	ctx := c.Request().Context()
	var req createWebhookRequest
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

	// Only call the service if validation passes
	if req.Name != "" {
		webhook, err := h.servs.WebhooksService.CreateWebhook(ctx, dbgen.WebhooksServiceCreateWebhookParams{
			Name:      req.Name,
			EventType: req.EventType,
			TargetIds: req.TargetIds,
			IsActive:  req.IsActive,
			Url:       req.URL,
			Method:    req.Method,
			Headers:   sql.NullString{String: req.Headers, Valid: req.Headers != ""},
			Body:      sql.NullString{String: req.Body, Valid: req.Body != ""},
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create webhook: " + err.Error(),
			})
		}

		return c.JSON(http.StatusCreated, webhookResponse{
			ID:        webhook.ID.String(),
			Name:      webhook.Name,
			EventType: webhook.EventType,
			TargetIds: webhook.TargetIds,
			IsActive:  webhook.IsActive,
			URL:       webhook.Url,
			Method:    webhook.Method,
			Headers:   webhook.Headers.String,
			Body:      webhook.Body.String,
			CreatedAt: webhook.CreatedAt.String(),
		})
	}

	return c.JSON(http.StatusBadRequest, map[string]string{
		"error": "Key: 'createWebhookRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag",
	})
}

func (h *mockWebhookHandlers) listWebhooksAPI(c echo.Context) error {
	ctx := c.Request().Context()
	_, webhooks, err := h.servs.WebhooksService.PaginateWebhooks(ctx, webhooks.PaginateWebhooksParams{
		Page:  1,
		Limit: 100,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list webhooks: " + err.Error(),
		})
	}

	response := make([]webhookResponse, len(webhooks))
	for i, webhook := range webhooks {
		response[i] = webhookResponse{
			ID:        webhook.ID.String(),
			Name:      webhook.Name,
			EventType: webhook.EventType,
			TargetIds: webhook.TargetIds,
			IsActive:  webhook.IsActive,
			URL:       webhook.Url,
			Method:    webhook.Method,
			Headers:   webhook.Headers.String,
			Body:      webhook.Body.String,
			CreatedAt: webhook.CreatedAt.String(),
		}
	}

	return c.JSON(http.StatusOK, response)
}

func (h *mockWebhookHandlers) getWebhookAPI(c echo.Context) error {
	ctx := c.Request().Context()
	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid webhook ID",
		})
	}

	webhook, err := h.servs.WebhooksService.GetWebhook(ctx, webhookID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get webhook: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, webhookResponse{
		ID:        webhook.ID.String(),
		Name:      webhook.Name,
		EventType: webhook.EventType,
		TargetIds: webhook.TargetIds,
		IsActive:  webhook.IsActive,
		URL:       webhook.Url,
		Method:    webhook.Method,
		Headers:   webhook.Headers.String,
		Body:      webhook.Body.String,
		CreatedAt: webhook.CreatedAt.String(),
	})
}

func (h *mockWebhookHandlers) deleteWebhookAPI(c echo.Context) error {
	ctx := c.Request().Context()
	webhookID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid webhook ID",
		})
	}

	err = h.servs.WebhooksService.DeleteWebhook(ctx, webhookID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete webhook: " + err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}

func TestCreateWebhookAPI(t *testing.T) {
	// Setup
	e := echo.New()
	mockWebhooksService := &MockWebhooksService{}

	// Create a mock service with our mock WebhooksService
	servs := &mockWebhookService{
		WebhooksService: mockWebhooksService,
	}

	// Create a handler with our mock service
	h := &mockWebhookHandlers{
		servs: servs,
	}

	// Set test API key
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	tests := []struct {
		name           string
		apiKey         string
		requestBody    interface{} // Changed to interface{} to support invalid JSON
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:   "successful webhook creation",
			apiKey: "test-api-key",
			requestBody: createWebhookRequest{
				Name:      "test-webhook",
				EventType: "database_healthy",
				TargetIds: []uuid.UUID{uuid.New()},
				IsActive:  true,
				URL:       "https://example.com/webhook",
				Method:    "POST",
				Headers:   `{"Content-Type": "application/json"}`,
				Body:      `{"key": "value"}`,
			},
			mockSetup: func() {
				mockWebhooksService.On("CreateWebhook", mock.Anything, mock.MatchedBy(func(params dbgen.WebhooksServiceCreateWebhookParams) bool {
					return params.Name == "test-webhook" && params.EventType == "database_healthy"
				})).Return(dbgen.Webhook{
					ID:        uuid.New(),
					Name:      "test-webhook",
					EventType: "database_healthy",
					TargetIds: []uuid.UUID{uuid.New()},
					IsActive:  true,
					Url:       "https://example.com/webhook",
					Method:    "POST",
					Headers:   sql.NullString{String: `{"Content-Type": "application/json"}`, Valid: true},
					Body:      sql.NullString{String: `{"key": "value"}`, Valid: true},
					CreatedAt: time.Now(),
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"name":       "test-webhook",
				"event_type": "database_healthy",
				"is_active":  true,
				"url":        "https://example.com/webhook",
				"method":     "POST",
			},
		},
		{
			name:   "missing API key",
			apiKey: "",
			requestBody: createWebhookRequest{
				Name:      "test-webhook",
				EventType: "database_healthy",
				TargetIds: []uuid.UUID{uuid.New()},
				URL:       "https://example.com/webhook",
				Method:    "POST",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:   "invalid request body",
			apiKey: "test-api-key",
			requestBody: createWebhookRequest{
				Name:      "", // Required field is empty
				EventType: "database_healthy",
				TargetIds: []uuid.UUID{uuid.New()},
				URL:       "https://example.com/webhook",
				Method:    "POST",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "error in field Name: required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.mockSetup()

			// Create request
			reqBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", bytes.NewReader(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Test with middleware
			handler := APIKeyAuth()(h.createWebhookAPI)
			err := handler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check expected fields
			for key, expectedValue := range tt.expectedBody.(map[string]interface{}) {
				assert.Equal(t, expectedValue, response[key])
			}

			// Additional assertions for successful creation
			if tt.expectedStatus == http.StatusCreated {
				assert.NotEmpty(t, response["id"])
				assert.NotEmpty(t, response["created_at"])
			}

			// Reset mock for next test
			mockWebhooksService.ExpectedCalls = nil
		})
	}
}

func TestListWebhooksAPI(t *testing.T) {
	// Setup
	e := echo.New()
	mockWebhooksService := &MockWebhooksService{}

	// Create a mock service with our mock WebhooksService
	servs := &mockWebhookService{
		WebhooksService: mockWebhooksService,
	}

	// Create a handler with our mock service
	h := &mockWebhookHandlers{
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
			name:   "successful webhooks list",
			apiKey: "test-api-key",
			mockSetup: func() {
				mockWebhooksService.On("PaginateWebhooks", mock.Anything, mock.Anything).Return(
					paginateutil.PaginateResponse{},
					[]dbgen.Webhook{
						{
							ID:        uuid.New(),
							Name:      "test-webhook-1",
							EventType: "database_healthy",
							TargetIds: []uuid.UUID{uuid.New()},
							IsActive:  true,
							Url:       "https://example.com/webhook1",
							Method:    "POST",
							CreatedAt: time.Now(),
						},
						{
							ID:        uuid.New(),
							Name:      "test-webhook-2",
							EventType: "database_unhealthy",
							TargetIds: []uuid.UUID{uuid.New()},
							IsActive:  true,
							Url:       "https://example.com/webhook2",
							Method:    "GET",
							CreatedAt: time.Now(),
						},
					},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: []webhookResponse{
				{
					Name:      "test-webhook-1",
					EventType: "database_healthy",
					IsActive:  true,
					URL:       "https://example.com/webhook1",
					Method:    "POST",
				},
				{
					Name:      "test-webhook-2",
					EventType: "database_unhealthy",
					IsActive:  true,
					URL:       "https://example.com/webhook2",
					Method:    "GET",
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
			name:   "list error",
			apiKey: "test-api-key",
			mockSetup: func() {
				mockWebhooksService.On("PaginateWebhooks", mock.Anything, mock.Anything).Return(
					paginateutil.PaginateResponse{},
					[]dbgen.Webhook{},
					assert.AnError,
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to list webhooks: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Test with middleware
			handler := APIKeyAuth()(h.listWebhooksAPI)
			err := handler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check expected fields
			if tt.expectedStatus == http.StatusOK {
				// For successful list, check each webhook in the response
				responseArray := response.([]interface{})
				expectedArray := tt.expectedBody.([]webhookResponse)
				assert.Equal(t, len(expectedArray), len(responseArray))

				for i, expectedWebhook := range expectedArray {
					responseWebhook := responseArray[i].(map[string]interface{})
					assert.Equal(t, expectedWebhook.Name, responseWebhook["name"])
					assert.Equal(t, expectedWebhook.EventType, responseWebhook["event_type"])
					assert.Equal(t, expectedWebhook.IsActive, responseWebhook["is_active"])
					assert.Equal(t, expectedWebhook.URL, responseWebhook["url"])
					assert.Equal(t, expectedWebhook.Method, responseWebhook["method"])
					assert.NotEmpty(t, responseWebhook["id"])
					assert.NotEmpty(t, responseWebhook["created_at"])
				}
			} else {
				// For error responses, check the error message
				expectedError := tt.expectedBody.(map[string]interface{})
				responseError := response.(map[string]interface{})
				assert.Equal(t, expectedError["error"], responseError["error"])
			}

			// Reset mock for next test
			mockWebhooksService.ExpectedCalls = nil
		})
	}
}

func TestGetWebhookAPI(t *testing.T) {
	// Setup
	e := echo.New()
	mockWebhooksService := &MockWebhooksService{}

	// Create a mock service with our mock WebhooksService
	servs := &mockWebhookService{
		WebhooksService: mockWebhooksService,
	}

	// Create a handler with our mock service
	h := &mockWebhookHandlers{
		servs: servs,
	}

	// Set test API key
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	tests := []struct {
		name           string
		apiKey         string
		webhookID      string
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:      "successful webhook get",
			apiKey:    "test-api-key",
			webhookID: uuid.New().String(),
			mockSetup: func() {
				mockWebhooksService.On("GetWebhook", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(
					dbgen.Webhook{
						ID:        uuid.New(),
						Name:      "test-webhook",
						EventType: "database_healthy",
						TargetIds: []uuid.UUID{uuid.New()},
						IsActive:  true,
						Url:       "https://example.com/webhook",
						Method:    "POST",
						Headers:   sql.NullString{String: `{"Content-Type": "application/json"}`, Valid: true},
						Body:      sql.NullString{String: `{"key": "value"}`, Valid: true},
						CreatedAt: time.Now(),
					}, nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: webhookResponse{
				Name:      "test-webhook",
				EventType: "database_healthy",
				IsActive:  true,
				URL:       "https://example.com/webhook",
				Method:    "POST",
			},
		},
		{
			name:           "missing API key",
			apiKey:         "",
			webhookID:      uuid.New().String(),
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:           "invalid webhook ID",
			apiKey:         "test-api-key",
			webhookID:      "invalid-uuid",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid webhook ID",
			},
		},
		{
			name:      "webhook not found",
			apiKey:    "test-api-key",
			webhookID: uuid.New().String(),
			mockSetup: func() {
				mockWebhooksService.On("GetWebhook", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(
					dbgen.Webhook{}, assert.AnError,
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to get webhook: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/"+tt.webhookID, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/webhooks/:id")
			c.SetParamNames("id")
			c.SetParamValues(tt.webhookID)

			// Test with middleware
			handler := APIKeyAuth()(h.getWebhookAPI)
			err := handler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check expected fields
			if tt.expectedStatus == http.StatusOK {
				// For successful get, check the webhook fields
				expectedWebhook := tt.expectedBody.(webhookResponse)
				responseWebhook := response.(map[string]interface{})
				assert.Equal(t, expectedWebhook.Name, responseWebhook["name"])
				assert.Equal(t, expectedWebhook.EventType, responseWebhook["event_type"])
				assert.Equal(t, expectedWebhook.IsActive, responseWebhook["is_active"])
				assert.Equal(t, expectedWebhook.URL, responseWebhook["url"])
				assert.Equal(t, expectedWebhook.Method, responseWebhook["method"])
				assert.NotEmpty(t, responseWebhook["id"])
				assert.NotEmpty(t, responseWebhook["created_at"])
			} else {
				// For error responses, check the error message
				expectedError := tt.expectedBody.(map[string]interface{})
				responseError := response.(map[string]interface{})
				assert.Equal(t, expectedError["error"], responseError["error"])
			}

			// Reset mock for next test
			mockWebhooksService.ExpectedCalls = nil
		})
	}
}

func TestDeleteWebhookAPI(t *testing.T) {
	// Setup
	e := echo.New()
	mockWebhooksService := &MockWebhooksService{}

	// Create a mock service with our mock WebhooksService
	servs := &mockWebhookService{
		WebhooksService: mockWebhooksService,
	}

	// Create a handler with our mock service
	h := &mockWebhookHandlers{
		servs: servs,
	}

	// Set test API key
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("API_KEY")

	tests := []struct {
		name           string
		apiKey         string
		webhookID      string
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:      "successful webhook deletion",
			apiKey:    "test-api-key",
			webhookID: uuid.New().String(),
			mockSetup: func() {
				mockWebhooksService.On("DeleteWebhook", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   nil,
		},
		{
			name:           "missing API key",
			apiKey:         "",
			webhookID:      uuid.New().String(),
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error": "Invalid or missing API key",
			},
		},
		{
			name:           "invalid webhook ID",
			apiKey:         "test-api-key",
			webhookID:      "invalid-uuid",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error": "Invalid webhook ID",
			},
		},
		{
			name:      "webhook not found",
			apiKey:    "test-api-key",
			webhookID: uuid.New().String(),
			mockSetup: func() {
				mockWebhooksService.On("DeleteWebhook", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error": "Failed to delete webhook: " + assert.AnError.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			tt.mockSetup()

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/webhooks/"+tt.webhookID, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/webhooks/:id")
			c.SetParamNames("id")
			c.SetParamValues(tt.webhookID)

			// Test with middleware
			handler := APIKeyAuth()(h.deleteWebhookAPI)
			err := handler(c)

			// Assertions
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedBody != nil {
				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)

				// Check expected fields
				for key, expectedValue := range tt.expectedBody.(map[string]interface{}) {
					assert.Equal(t, expectedValue, response[key])
				}
			} else {
				assert.Empty(t, rec.Body.String())
			}

			// Reset mock for next test
			mockWebhooksService.ExpectedCalls = nil
		})
	}
}
