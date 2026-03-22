package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/doomslock/backend/internal/limit/service"
	"github.com/doomslock/backend/pkg/middleware"
	"github.com/doomslock/backend/pkg/response"
	"github.com/doomslock/backend/pkg/validator"
)

type Handler struct{ svc service.Service }

func New(svc service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(api *echo.Group, mw echo.MiddlewareFunc) {
	g := api.Group("/limits", mw)
	g.POST("", h.Create)
	g.GET("", h.List)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

func (h *Handler) Create(c echo.Context) error {
	userID := middleware.MustUserID(c)

	var req service.CreateRequest
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	limit, err := h.svc.Create(c.Request().Context(), userID, req)
	if err != nil {
		return mapLimitErr(c, err)
	}

	return response.Created(c, limit)
}

func (h *Handler) List(c echo.Context) error {
	userID := middleware.MustUserID(c)
	groupID := c.QueryParam("group_id")
	if groupID == "" {
		return response.Error(c, http.StatusBadRequest, "group_id is required")
	}

	limits, err := h.svc.ListByGroup(c.Request().Context(), userID, groupID)
	if err != nil {
		return mapLimitErr(c, err)
	}

	return response.OK(c, limits)
}

func (h *Handler) Update(c echo.Context) error {
	userID := middleware.MustUserID(c)
	limitID := c.Param("id")

	var req service.UpdateRequest
	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	limit, err := h.svc.Update(c.Request().Context(), userID, limitID, req)
	if err != nil {
		return mapLimitErr(c, err)
	}

	return response.OK(c, limit)
}

func (h *Handler) Delete(c echo.Context) error {
	userID := middleware.MustUserID(c)
	limitID := c.Param("id")

	if err := h.svc.Delete(c.Request().Context(), userID, limitID); err != nil {
		return mapLimitErr(c, err)
	}

	return response.NoContent(c)
}

func mapLimitErr(c echo.Context, err error) error {
	switch {
	case errors.Is(err, service.ErrLimitNotFound):
		return response.Error(c, http.StatusNotFound, "limit not found")
	case errors.Is(err, service.ErrNotOwner):
		return response.Error(c, http.StatusForbidden, "not your limit")
	case errors.Is(err, service.ErrNotInGroup):
		return response.Error(c, http.StatusForbidden, "not a member of this group")
	default:
		return response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}
