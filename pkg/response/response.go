package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type envelope struct {
	Data  interface{} `json:"data,omitempty"`
	Error interface{} `json:"error,omitempty"`
}

func OK(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, envelope{Data: data})
}

func Created(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, envelope{Data: data})
}

func Error(c echo.Context, status int, msg string) error {
	return c.JSON(status, envelope{Error: msg})
}

func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
