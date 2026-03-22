package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/doomslock/backend/internal/group/service"
	"github.com/doomslock/backend/pkg/middleware"
	"github.com/doomslock/backend/pkg/response"
	"github.com/doomslock/backend/pkg/validator"
)

type Handler struct{ svc service.Service }

func New(svc service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(api *echo.Group, mw echo.MiddlewareFunc) {
	g := api.Group("/groups", mw)
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.GetByID)
	g.POST("/:id/invite", h.CreateInvite)
	g.POST("/join", h.AcceptInvite)
	g.POST("/:id/leave", h.Leave)
	g.DELETE("/:id/members/:user_id", h.RemoveMember)
}

func (h *Handler) Create(c echo.Context) error {
	userID := middleware.MustUserID(c)

	var req service.CreateRequest
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	group, err := h.svc.Create(c.Request().Context(), userID, req)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "failed to create group")
	}

	return response.Created(c, group)
}

func (h *Handler) List(c echo.Context) error {
	userID := middleware.MustUserID(c)

	groups, err := h.svc.ListMyGroups(c.Request().Context(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "failed to list groups")
	}

	return response.OK(c, groups)
}

func (h *Handler) GetByID(c echo.Context) error {
	userID := middleware.MustUserID(c)
	groupID := c.Param("id")

	detail, err := h.svc.GetByID(c.Request().Context(), userID, groupID)
	if err != nil {
		return mapGroupErr(c, err)
	}

	return response.OK(c, detail)
}

func (h *Handler) CreateInvite(c echo.Context) error {
	userID := middleware.MustUserID(c)
	groupID := c.Param("id")

	var req service.InviteRequest
	_ = c.Bind(&req)

	result, err := h.svc.CreateInvite(c.Request().Context(), userID, groupID, req)
	if err != nil {
		return mapGroupErr(c, err)
	}

	return response.Created(c, result)
}

func (h *Handler) AcceptInvite(c echo.Context) error {
	userID := middleware.MustUserID(c)

	var req service.JoinRequest
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	group, err := h.svc.AcceptInvite(c.Request().Context(), userID, req)
	if err != nil {
		return mapGroupErr(c, err)
	}

	return response.OK(c, group)
}

func (h *Handler) Leave(c echo.Context) error {
	userID := middleware.MustUserID(c)
	groupID := c.Param("id")

	if err := h.svc.LeaveGroup(c.Request().Context(), userID, groupID); err != nil {
		return mapGroupErr(c, err)
	}

	return response.OK(c, map[string]string{"message": "left group"})
}

func (h *Handler) RemoveMember(c echo.Context) error {
	adminID := middleware.MustUserID(c)
	groupID := c.Param("id")
	targetID := c.Param("user_id")

	if err := h.svc.RemoveMember(c.Request().Context(), adminID, groupID, targetID); err != nil {
		return mapGroupErr(c, err)
	}

	return response.OK(c, map[string]string{"message": "member removed"})
}

func mapGroupErr(c echo.Context, err error) error {
	switch {
	case errors.Is(err, service.ErrGroupNotFound):
		return response.Error(c, http.StatusNotFound, "group not found")
	case errors.Is(err, service.ErrNotMember):
		return response.Error(c, http.StatusForbidden, "you are not a member")
	case errors.Is(err, service.ErrNotAdmin):
		return response.Error(c, http.StatusForbidden, "admin only")
	case errors.Is(err, service.ErrGroupFull):
		return response.Error(c, http.StatusConflict, "group is full")
	case errors.Is(err, service.ErrAlreadyMember):
		return response.Error(c, http.StatusConflict, "already a member")
	case errors.Is(err, service.ErrInviteInvalid):
		return response.Error(c, http.StatusBadRequest, "invite invalid or expired")
	case errors.Is(err, service.ErrCannotRemoveSelf):
		return response.Error(c, http.StatusBadRequest, "use /leave instead")
	default:
		return response.Error(c, http.StatusInternalServerError, "internal server error")
	}
}
