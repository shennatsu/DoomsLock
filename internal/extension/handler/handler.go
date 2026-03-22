package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/doomslock/backend/internal/extension/service"
	"github.com/doomslock/backend/pkg/middleware"
	"github.com/doomslock/backend/pkg/response"
	"github.com/doomslock/backend/pkg/validator"
)

type Handler struct{ svc service.Service }

func New(svc service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(api *echo.Group, mw echo.MiddlewareFunc) {
	g := api.Group("/extensions", mw)
	g.POST("", h.Request)
	g.GET("/:id", h.GetByID)
	g.POST("/:id/vote", h.Vote)

	// List extensions by limit
	api.GET("/limits/:limit_id/extensions", h.ListByLimit, mw)
}

func (h *Handler) Request(c echo.Context) error {
	userID := middleware.MustUserID(c)

	var req service.RequestInput
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	ext, err := h.svc.Request(c.Request().Context(), userID, req)
	if err != nil {
		return mapExtErr(c, err)
	}

	return response.Created(c, ext)
}

func (h *Handler) GetByID(c echo.Context) error {
	extID := c.Param("id")

	detail, err := h.svc.GetByID(c.Request().Context(), extID)
	if err != nil {
		return mapExtErr(c, err)
	}

	return response.OK(c, detail)
}

func (h *Handler) Vote(c echo.Context) error {
	voterID := middleware.MustUserID(c)
	extID := c.Param("id")

	var req service.VoteInput
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	ext, err := h.svc.CastVote(c.Request().Context(), voterID, extID, req)
	if err != nil {
		return mapExtErr(c, err)
	}

	return response.OK(c, ext)
}

func (h *Handler) ListByLimit(c echo.Context) error {
	limitID := c.Param("limit_id")

	exts, err := h.svc.ListByLimit(c.Request().Context(), limitID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "failed to list extensions")
	}

	return response.OK(c, exts)
}

func mapExtErr(c echo.Context, err error) error {
	switch {
	case errors.Is(err, service.ErrExtNotFound):
		return response.Error(c, http.StatusNotFound, "extension not found")
	case errors.Is(err, service.ErrAlreadyVoted):
		return response.Error(c, http.StatusConflict, "already voted")
	case errors.Is(err, service.ErrCannotVoteOwn):
		return response.Error(c, http.StatusBadRequest, "cannot vote on your own request")
	case errors.Is(err, service.ErrExtExpired):
		return response.Error(c, http.StatusGone, "extension request expired")
	case errors.Is(err, service.ErrExtResolved):
		return response.Error(c, http.StatusConflict, "extension already resolved")
	case errors.Is(err, service.ErrNotInGroup):
		return response.Error(c, http.StatusForbidden, "not a member of this group")
	default:
		return response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}
