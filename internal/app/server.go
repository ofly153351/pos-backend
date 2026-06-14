package app

import (
	"context"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"pos-backend/internal/config"
	"pos-backend/internal/database"
	"pos-backend/internal/modules/provisioning"
)

type Server struct {
	app *fiber.App
	cfg config.Config
}

func NewServer(cfg config.Config) (*Server, error) {
	db, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.AutoMigrate {
		if err := database.RunMigrations(db, cfg.MigrationsDir); err != nil {
			return nil, err
		}
	}

	// Phase W1: ensure every existing store has a default warehouse + sale location.
	// Idempotent and best-effort (per-store failures are logged, never fatal), so this
	// is safe to run on every boot and never blocks startup.
	provisioning.BackfillAllStores(context.Background(), db)
	// Phase W2: assign that default sale location to existing products that still have a
	// NULL default_location_id (metadata-only; never moves stock). Runs after the store
	// backfill so the default sale location exists.
	provisioning.BackfillProductDefaultLocations(context.Background(), db)

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		return nil, err
	}
	deps := newDependencies(cfg, db)

	fiberApp := fiber.New(fiber.Config{
		AppName: cfg.AppName,
	})

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	registerBaseRoutes(fiberApp, cfg)
	registerAPIRoutes(fiberApp, deps)

	return &Server{
		app: fiberApp,
		cfg: cfg,
	}, nil
}

func (s *Server) Start() error {
	return s.app.Listen(s.cfg.HTTPAddress())
}
