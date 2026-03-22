package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/doomslock/backend/config"
	"github.com/doomslock/backend/pkg/logger"
	"github.com/doomslock/backend/pkg/response"
)

type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func Register(e *echo.Echo, log *logger.Logger, cfg *config.Config) {
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())

	e.Use(echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, v echomw.RequestLoggerValues) error {
			if v.Error != nil {
				log.Errorw("request",
					"method", v.Method,
					"uri", v.URI,
					"status", v.Status,
					"latency", v.Latency,
					"request_id", v.RequestID,
					"error", v.Error,
				)
			} else {
				log.Infow("request",
					"method", v.Method,
					"uri", v.URI,
					"status", v.Status,
					"latency", v.Latency,
					"request_id", v.RequestID,
				)
			}
			return nil
		},
	}))

	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
	}))

	e.Use(echomw.RateLimiter(echomw.NewRateLimiterMemoryStore(100)))
	e.Use(echomw.BodyLimit("10M"))
}

func JWT(cfg config.JWTConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get(echo.HeaderAuthorization)
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				return response.Error(c, http.StatusUnauthorized, "missing or invalid authorization header")
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims := &JWTClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.ErrUnauthorized
				}
				return []byte(cfg.Secret), nil
			})

			if err != nil || !token.Valid {
				return response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			}

			if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
				return response.Error(c, http.StatusUnauthorized, "token expired")
			}

			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)

			return next(c)
		}
	}
}

func MustUserID(c echo.Context) string {
	return c.Get("user_id").(string)
}
