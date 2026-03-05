package health

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

type statusOk struct {
	Status string `json:"status"`
}

type HealthHandler struct{}

func NewHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Register(e *echo.Group) {
	e.GET("", h.Health)
}

func (h *HealthHandler) Health(c *echo.Context) error {
	return c.JSON(http.StatusOK, statusOk{Status: "OK"})
}
