package app

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"pos-backend/internal/config"
	"pos-backend/internal/database"
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
