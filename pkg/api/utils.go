package api

import (
	"github.com/labstack/echo/v4"
)

func errorResonse(c echo.Context, err error, code int) error {
	e := map[string]string{"error": err.Error()}
	return c.JSON(code, e)
}
