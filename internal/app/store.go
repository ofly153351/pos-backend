package app

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/config"
	"pos-backend/internal/modules/store"
)

func newStoreHandler(cfg config.Config, db *sql.DB) store.Handler {
	repo := store.NewPostgresRepository(db)
	storage := store.NewMinIOLogoStorage(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucketName,
		cfg.MinIOUseSSL,
		cfg.MinIOPublicURL,
	)
	service := store.NewService(repo, storage)
	return store.NewHandler(service)
}

func registerStoreRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.storeHandler.(store.Handler)
	protected.Post("/stores", handler.Create)
}
