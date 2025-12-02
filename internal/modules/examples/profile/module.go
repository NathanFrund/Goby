package profile

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/registry"
)

type Dependencies struct {
	FileRepository domain.FileRepository
}

type Module struct {
	module.BaseModule
	fileRepo domain.FileRepository
	handler  *Handler
}

func New(deps Dependencies) *Module {
	return &Module{
		fileRepo: deps.FileRepository,
	}
}

func (m *Module) Name() string {
	return "profile"
}

func (m *Module) Register(reg *registry.Registry) error {
	return nil
}

func (m *Module) Boot(ctx context.Context, group *echo.Group, reg *registry.Registry) error {
	m.handler = NewHandler(m.fileRepo)
	group.GET("", m.handler.Get)
	return nil
}
