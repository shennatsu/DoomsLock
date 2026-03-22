package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/doomslock/backend/internal/usage/service"
	"github.com/doomslock/backend/pkg/middleware"
	"github.com/doomslock/backend/pkg/response"
	"github.com/doomslock/backend/pkg/validator"
)

type Handler struct{ svc service.Service }

func New(svc service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(api *echo.Group, mw echo.MiddlewareFunc) {
	g := api.Group("/usage", mw)
	g.POST("/sync", h.Sync)
	g.GET("/summary", h.Summary)
}

func (h *Handler) Sync(c echo.Context) error {
	userID := middleware.MustUserID(c)

	var req service.SyncRequest
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	count, err := h.svc.Sync(c.Request().Context(), userID, req)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "sync failed")
	}

	return response.OK(c, map[string]int{"synced": count})
}

func (h *Handler) Summary(c echo.Context) error {
	userID := middleware.MustUserID(c)
	date := c.QueryParam("date")

	summary, err := h.svc.GetDailySummary(c.Request().Context(), userID, date)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "failed to get summary")
	}

	return response.OK(c, summary)
}
