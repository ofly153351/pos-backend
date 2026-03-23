package app

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/config"
	"pos-backend/internal/modules/auth"
)

func newAuthDependencies(cfg config.Config, db *gorm.DB) (auth.TokenManager, auth.Handler) {
	tokenManager := auth.NewTokenManager(cfg.TokenKey, time.Duration(cfg.TokenTTL)*time.Hour)
	repo := auth.NewPostgresUserRepository(db)
	service := auth.NewService(repo, tokenManager)
	return tokenManager, auth.NewHandler(service)
}

func registerAuthRoutes(api fiber.Router, deps appDependencies) {
	group := api.Group("/auth")
	group.Post("/register", deps.authHandler.Register)
	group.Post("/login", deps.authHandler.Login)
}
