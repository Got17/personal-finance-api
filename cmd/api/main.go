package main

import (
	"context"
	"log/slog"
	"os"

	gormadapter "github.com/BounkhongDev/bkgo/adapter/gorm"
	"github.com/BounkhongDev/bkgo/adapter/jwt"
	"github.com/BounkhongDev/bkgo/adapter/redis"
	"github.com/BounkhongDev/bkgo/config"
	"github.com/BounkhongDev/bkgo/contract"
	"github.com/BounkhongDev/bkgo/logger"
	"github.com/BounkhongDev/bkgo/middleware"
	"github.com/BounkhongDev/bkgo/response"
	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/Got17/personal-finance-api/internal/account"
	"github.com/Got17/personal-finance-api/internal/category"
	"github.com/Got17/personal-finance-api/internal/user"
	"github.com/Got17/personal-finance-api/internal/workspace"
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

	if err := db.Raw().Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		slog.Error("enable pgcrypto extension failed", "error", err)
		os.Exit(1)
	}

	if err := db.Raw().AutoMigrate(&user.User{}, &workspace.Workspace{}, &account.Account{}, &category.Category{}); err != nil {
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

	userRepo := user.NewUserRepository(db)
	userUsecase := user.NewUserUsecase(userRepo, token)
	userHandler := user.NewUserHandler(userUsecase)

	workspaceRepo := workspace.NewWorkspaceRepository(db)
	workspaceUsecase := workspace.NewWorkspaceUsecase(workspaceRepo)
	workspaceHandler := workspace.NewWorkspaceHandler(workspaceUsecase)

	accountRepo := account.NewAccountRepository(db)
	accountUsecase := account.NewAccountUsecase(accountRepo)
	accountHandler := account.NewAccountHandler(accountUsecase)

	categoryRepo := category.NewCategoryRepository(db)
	categoryUsecase := category.NewCategoryUsecase(categoryRepo)
	categoryHandler := category.NewCategoryHandler(categoryUsecase)

	app := newAPIApp(cfg.App.Name, userHandler, workspaceHandler, accountHandler, categoryHandler, token)

	_ = cache

	slog.Info("server starting", "port", cfg.App.Port, "env", cfg.App.Env)
	if err := app.Listen(":" + cfg.App.Port); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func newApp(appName string) *fiber.App {
	app := fiber.New(fiber.Config{AppName: appName})
	app.Use(fiberlogger.New())
	app.Use(middleware.CORS())

	// Health is public so deployment tooling can verify the service before
	// authentication is configured by a later ticket.
	app.Get("/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(response.Success(fiber.Map{"status": "ok", "app": appName}))
	})

	return app
}

func newAPIApp(appName string, userHandler *user.UserHandler, workspaceHandler *workspace.WorkspaceHandler, accountHandler *account.AccountHandler, categoryHandler *category.CategoryHandler, token contract.Token) *fiber.App {
	app := newApp(appName)

	// Public routes
	userHandler.RegisterAuthRoutes(app.Group("/v1"))

	// Protected routes
	api := app.Group("/v1", middleware.JWT(token))
	userHandler.RegisterProtectedRoutes(api)
	workspaceHandler.RegisterRoutes(api)
	accountHandler.RegisterRoutes(api)
	categoryHandler.RegisterRoutes(api)

	return app
}
