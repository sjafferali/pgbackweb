package api

import (
	"github.com/eduardolat/pgbackweb/internal/service"
)

type handlers struct {
	servs *service.Services
}

func NewHandlers(servs *service.Services) *handlers {
	return &handlers{
		servs: servs,
	}
} 