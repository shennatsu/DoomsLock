package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/doomslock/backend/config"
	"github.com/doomslock/backend/pkg/database"
	"github.com/doomslock/backend/pkg/logger"
	"github.com/doomslock/backend/pkg/redis"

	authHandler      "github.com/doomslock/backend/internal/auth/handler"
	authRepo         "github.com/doomslock/backend/internal/auth/repository"
	authService      "github.com/doomslock/backend/internal/auth/service"

	groupHandler     "github.com/doomslock/backend/internal/group/handler"
	groupRepo        "github.com/doomslock/backend/internal/group/repository"
	groupService     "github.com/doomslock/backend/internal/group/service"

	limitHandler     "github.com/doomslock/backend/internal/limit/handler"
	limitRepo        "github.com/doomslock/backend/internal/limit/repository"
	limitService     "github.com/doomslock/backend/internal/limit/service"

	extensionHandler "github.com/doomslock/backend/internal/extension/handler"
	extensionRepo    "github.com/doomslock/backend/internal/extension/repository"
	extensionService "github.com/doomslock/backend/internal/extension/service"

	usageHandler     "github.com/doomslock/backend/internal/usage/handler"
	usageRepo        "github.com/doomslock/backend/internal/usage/repository"
	usageService     "github.com/doomslock/backend/internal/usage/service"

	rewardHandler    "github.com/doomslock/backend/internal/reward/handler"
	rewardRepo       "github.com/doomslock/backend/internal/reward/repository"
	rewardService    "github.com/doomslock/backend/internal/reward/service"

	"github.com/doomslock/backend/pkg/middleware"
	"github.com/doomslock/backend/pkg/validator"
)

func main() {
	cfg := config.Load()

	log := logger.New(cfg.App.Env)
	defer log.Sync()

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatal(fmt.Sprintf("postgres: %v", err))
	}
	defer db.Close()

	rdb := redis.New(cfg.Redis)
	defer rdb.Close()

	// --- Wire dependencies ---

	// Auth
	aRepo := authRepo.New(db)
	aSvc := authService.New(aRepo, rdb, cfg.JWT)
	aH := authHandler.New(aSvc)

	// Group
	gRepo := groupRepo.New(db)
	gSvc := groupService.New(gRepo)
	gH := groupHandler.New(gSvc)

	// Limit
	lRepo := limitRepo.New(db)
	lSvc := limitService.New(lRepo, gRepo)
	lH := limitHandler.New(lSvc)

	// Extension (vote)
	exRepo := extensionRepo.New(db)
	exSvc := extensionService.New(exRepo, lRepo, gRepo)
	exH := extensionHandler.New(exSvc)

	// Usage
	uRepo := usageRepo.New(db)
	uSvc := usageService.New(uRepo)
	uH := usageHandler.New(uSvc)

	// Reward
	rRepo := rewardRepo.New(db)
	rSvc := rewardService.New(rRepo, uRepo)
	rH := rewardHandler.New(rSvc)

	// --- Echo setup ---
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = validator.New()

	middleware.Register(e, log, cfg)

	jwtMw := middleware.JWT(cfg.JWT)
	api := e.Group("/api/v1")

	aH.RegisterRoutes(api, jwtMw)
	gH.RegisterRoutes(api, jwtMw)
	lH.RegisterRoutes(api, jwtMw)
	exH.RegisterRoutes(api, jwtMw)
	uH.RegisterRoutes(api, jwtMw)
	rH.RegisterRoutes(api, jwtMw)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// --- Graceful shutdown ---
	addr := fmt.Sprintf(":%s", cfg.App.Port)
	go func() {
		log.Info(fmt.Sprintf("server starting on %s", addr))
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatal(fmt.Sprintf("server error: %v", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(fmt.Sprintf("shutdown error: %v", err))
	}
	log.Info("server stopped gracefully")
}
