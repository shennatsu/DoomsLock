package validator

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type EchoValidator struct {
	v *validator.Validate
}

func New() *EchoValidator {
	return &EchoValidator{v: validator.New()}
}

func (ev *EchoValidator) Validate(i interface{}) error {
	return ev.v.Struct(i)
}

func BindAndValidate(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return err
	}
	if err := c.Validate(req); err != nil {
		return err
	}
	return nil
}
