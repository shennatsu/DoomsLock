package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/doomslock/backend/internal/reward/service"
	"github.com/doomslock/backend/pkg/middleware"
	"github.com/doomslock/backend/pkg/response"
)

type Handler struct{ svc service.Service }

func New(svc service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(api *echo.Group, mw echo.MiddlewareFunc) {
	g := api.Group("/rewards", mw)
	g.GET("/streak", h.GetStreak)
	g.POST("/streak/update", h.UpdateStreak)
	g.GET("/badges", h.GetBadges)
}

func (h *Handler) GetStreak(c echo.Context) error {
	userID := middleware.MustUserID(c)

	streak, err := h.svc.GetStreak(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "failed to get streak")
	}

	return response.OK(c, streak)
}

func (h *Handler) UpdateStreak(c echo.Context) error {
	userID := middleware.MustUserID(c)

	streak, err := h.svc.UpdateStreak(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "failed to update streak")
	}

	return response.OK(c, streak)
}

func (h *Handler) GetBadges(c echo.Context) error {
	userID := middleware.MustUserID(c)

	badges, err := h.svc.GetBadges(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "failed to get badges")
	}

	return response.OK(c, badges)
}
