package api

import (
	"database/sql"
	"net/http"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/service/webhooks"
	"github.com/eduardolat/pgbackweb/internal/validate"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type createWebhookRequest struct {
	Name      string      `json:"name" validate:"required"`
	EventType string      `json:"event_type" validate:"required"`
	TargetIds []uuid.UUID `json:"target_ids" validate:"required,gt=0"`
	IsActive  bool        `json:"is_active"`
	URL       string      `json:"url" validate:"required,url"`
	Method    string      `json:"method" validate:"required,oneof=GET POST"`
	Headers   string      `json:"headers,omitempty" validate:"omitempty,json"`
	Body      string      `json:"body,omitempty" validate:"omitempty,json"`
}

type webhookResponse struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	EventType string      `json:"event_type"`
	TargetIds []uuid.UUID `json:"target_ids"`
	IsActive  bool        `json:"is_active"`
	URL       string      `json:"url"`
	Method    string      `json:"method"`
	Headers   string      `json:"headers,omitempty"`
	Body      string      `json:"body,omitempty"`
	CreatedAt string      `json:"created_at"`
}

func (h *handlers) createWebhookAPI(c echo.Context) error {
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

	webhook, err := h.servs.WebhooksService.CreateWebhook(
		ctx, dbgen.WebhooksServiceCreateWebhookParams{
			Name:      req.Name,
			EventType: req.EventType,
			TargetIds: req.TargetIds,
			IsActive:  req.IsActive,
			Url:       req.URL,
			Method:    req.Method,
			Headers:   sql.NullString{String: req.Headers, Valid: req.Headers != ""},
			Body:      sql.NullString{String: req.Body, Valid: req.Body != ""},
		},
	)
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

func (h *handlers) listWebhooksAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get all webhooks using pagination
	_, webhookList, err := h.servs.WebhooksService.PaginateWebhooks(
		ctx, webhooks.PaginateWebhooksParams{
			Page:  1,
			Limit: 100, // Maximum limit
		},
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list webhooks: " + err.Error(),
		})
	}

	// Convert to response format
	response := make([]webhookResponse, len(webhookList))
	for i, webhook := range webhookList {
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

func (h *handlers) getWebhookAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get webhook ID from URL parameter
	webhookIDStr := c.Param("id")
	webhookID, err := uuid.Parse(webhookIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid webhook ID",
		})
	}

	// Get the webhook
	webhook, err := h.servs.WebhooksService.GetWebhook(ctx, webhookID)
	if err != nil {
		// Check if the error is due to webhook not found
		if err.Error() == "webhook not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Webhook not found",
			})
		}
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

func (h *handlers) deleteWebhookAPI(c echo.Context) error {
	ctx := c.Request().Context()

	// Get webhook ID from URL parameter
	webhookIDStr := c.Param("id")
	webhookID, err := uuid.Parse(webhookIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid webhook ID",
		})
	}

	// Delete the webhook
	err = h.servs.WebhooksService.DeleteWebhook(ctx, webhookID)
	if err != nil {
		// Check if the error is due to webhook not found
		if err.Error() == "webhook not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Webhook not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete webhook: " + err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}
