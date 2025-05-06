package api

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

// APIKeyAuth middleware checks for a valid API key in the X-API-Key header
func APIKeyAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-API-Key")
			expectedAPIKey := os.Getenv("API_KEY")

			if apiKey == "" || apiKey != expectedAPIKey {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or missing API key",
				})
			}

			return next(c)
		}
	}
} 