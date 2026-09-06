package main

import (
	"context"
	"log/slog"
	"os"

	gormadapter "github.com/BounkhongDev/bkgo/adapter/gorm"
	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/adapter/redis"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/logger"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"

	"github.com/Got17/personal-finance-api/internal/user"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	log := logger.Development()
	if cfg.App.Env == "production" {
		log = logger.Production()
	}
	slog.SetDefault(log)

	db, err := gormadapter.New(cfg.Postgres)
	if err != nil {
		slog.Error("postgres connect failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Raw().AutoMigrate(&user.User{}); err != nil {
		slog.Error("automigrate failed", "error", err)
		os.Exit(1)
	}

	cache, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		slog.Error("redis connect failed", "error", err)
		os.Exit(1)
	}
	defer cache.Close()

	token := jwt.New(cfg.JWT)

	userHandler := user.NewUserHandler(user.NewUserUsecase(user.NewUserRepository(db), token))
	app := newAPIApp(cfg.App.Name, userHandler)

	// Protected API routes
	api := app.Group("/v1", middleware.JWT(token))
	_ = api

	_ = cache

	slog.Info("server starting", "port", cfg.App.Port, "env", cfg.App.Env)
	if err := app.Listen(":" + cfg.App.Port); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func newApp(appName string) *fiber.App {
	app := fiber.New(fiber.Config{AppName: appName})
	app.Use(middleware.CORS())

	// Health is public so deployment tooling can verify the service before
	// authentication is configured by a later ticket.
	app.Get("/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(response.Success(fiber.Map{"status": "ok", "app": appName}))
	})

	return app
}

func newAPIApp(appName string, userHandler *user.UserHandler) *fiber.App {
	app := newApp(appName)
	userHandler.RegisterAuthRoutes(app.Group("/v1"))
	return app
}
