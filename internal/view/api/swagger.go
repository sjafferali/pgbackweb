package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

//go:embed swagger-ui/*
var swaggerUI embed.FS

//go:embed swagger.json
var swaggerSpec []byte

// RegisterSwaggerUI registers the Swagger UI routes
func RegisterSwaggerUI(parent *echo.Group) {
	// Serve Swagger UI static files
	swaggerUIFS, err := fs.Sub(swaggerUI, "swagger-ui")
	if err != nil {
		panic(err)
	}
	parent.StaticFS("/swagger-ui", echo.MustSubFS(swaggerUIFS, "swagger-ui"))

	// Serve Swagger spec
	parent.GET("/swagger.json", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, swaggerSpec)
	})

	// Serve Swagger UI index
	parent.GET("/docs", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/api/swagger-ui/index.html")
	})
}
