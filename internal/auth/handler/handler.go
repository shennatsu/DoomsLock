package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/doomslock/backend/internal/auth/service"
	"github.com/doomslock/backend/pkg/middleware"
	"github.com/doomslock/backend/pkg/response"
	"github.com/doomslock/backend/pkg/validator"
)

type Handler struct {
	svc service.Service
}

func New(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(api *echo.Group, jwtMw echo.MiddlewareFunc) {
	pub := api.Group("/auth")
	pub.POST("/register", h.Register)
	pub.POST("/login", h.Login)
	pub.POST("/refresh", h.Refresh)

	priv := api.Group("/auth", jwtMw)
	priv.POST("/logout", h.Logout)
}

func (h *Handler) Register(c echo.Context) error {
	var req service.RegisterRequest
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	res, err := h.svc.Register(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			return response.Error(c, http.StatusConflict, "email already registered")
		}
		return response.Error(c, http.StatusInternalServerError, "registration failed")
	}

	return response.Created(c, res)
}

func (h *Handler) Login(c echo.Context) error {
	var req service.LoginRequest
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	res, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) {
			return response.Error(c, http.StatusUnauthorized, "invalid email or password")
		}
		return response.Error(c, http.StatusInternalServerError, "login failed")
	}

	return response.OK(c, res)
}

func (h *Handler) Refresh(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	res, err := h.svc.RefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return response.Error(c, http.StatusUnauthorized, "invalid or expired refresh token")
	}

	return response.OK(c, res)
}

func (h *Handler) Logout(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := validator.BindAndValidate(c, &req); err != nil {
		return response.Error(c, http.StatusBadRequest, err.Error())
	}

	userID := middleware.MustUserID(c)
	if err := h.svc.Logout(c.Request().Context(), userID, req.RefreshToken); err != nil {
		return response.Error(c, http.StatusInternalServerError, "logout failed")
	}

	return response.OK(c, map[string]string{"message": "logged out"})
}
