package app

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"pos-backend/internal/config"
	"pos-backend/internal/database"
	"pos-backend/internal/middleware"
	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/product"
	"pos-backend/internal/modules/producttype"
	"pos-backend/internal/modules/productunit"
	"pos-backend/internal/modules/sale"
	"pos-backend/internal/modules/store"
	"pos-backend/internal/modules/subscription"
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

	authRepo := auth.NewPostgresUserRepository(db)
	tokenManager := auth.NewTokenManager(cfg.TokenKey, time.Duration(cfg.TokenTTL)*time.Hour)
	authService := auth.NewService(authRepo, tokenManager)
	authHandler := auth.NewHandler(authService)
	storeRepo := store.NewPostgresRepository(db)
	storeStorage := store.NewLocalLogoStorage(cfg.UploadDir, "/uploads")
	storeService := store.NewService(storeRepo, storeStorage)
	storeHandler := store.NewHandler(storeService)
	productTypeRepo := producttype.NewPostgresRepository(db)
	productTypeService := producttype.NewService(productTypeRepo)
	productTypeHandler := producttype.NewHandler(productTypeService)
	productUnitRepo := productunit.NewPostgresRepository(db)
	productUnitService := productunit.NewService(productUnitRepo)
	productUnitHandler := productunit.NewHandler(productUnitService)
	productRepo := product.NewPostgresRepository(db)
	productStorage := product.NewLocalImageStorage(cfg.UploadDir, "/uploads")
	productService := product.NewService(productRepo, productStorage)
	productHandler := product.NewHandler(productService)
	saleRepo := sale.NewPostgresRepository(db)
	saleService := sale.NewService(saleRepo)
	saleHandler := sale.NewHandler(saleService)
	subscriptionRepo := subscription.NewPostgresRepository(db)
	subscriptionService := subscription.NewService(subscriptionRepo)
	subscriptionHandler := subscription.NewHandler(subscriptionService)

	fiberApp := fiber.New(fiber.Config{
		AppName: cfg.AppName,
	})

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	fiberApp.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})
	fiberApp.Static("/uploads", cfg.UploadDir)

	api := fiberApp.Group("/api/v1")
	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	protected := api.Group("", middleware.AuthRequired(tokenManager))
	protected.Get("/subscriptions/plans", subscriptionHandler.ListPlans)
	protected.Post("/stores", storeHandler.Create)
	protected.Post("/stores/:storeID/product-types", productTypeHandler.Create)
	protected.Get("/stores/:storeID/product-types", productTypeHandler.ListByStore)
	protected.Patch("/stores/:storeID/product-types/:productTypeID", productTypeHandler.Update)
	protected.Delete("/stores/:storeID/product-types/:productTypeID", productTypeHandler.Delete)
	protected.Post("/stores/:storeID/product-units", productUnitHandler.Create)
	protected.Get("/stores/:storeID/product-units", productUnitHandler.List)
	protected.Patch("/stores/:storeID/product-units/:unitID", productUnitHandler.Update)
	protected.Delete("/stores/:storeID/product-units/:unitID", productUnitHandler.Delete)
	protected.Post("/stores/:storeID/products", productHandler.Create)
	protected.Get("/stores/:storeID/products", productHandler.ListByStore)
	protected.Get("/stores/:storeID/products/:productID", productHandler.GetByID)
	protected.Patch("/stores/:storeID/products/:productID", productHandler.Update)
	protected.Delete("/stores/:storeID/products/:productID", productHandler.Delete)
	protected.Post("/stores/:storeID/sales", saleHandler.Create)
	protected.Get("/stores/:storeID/sales", saleHandler.ListByStore)
	protected.Get("/stores/:storeID/sales/:saleID", saleHandler.GetByID)
	protected.Get("/stores/:storeID/subscription", subscriptionHandler.GetCurrentByStore)
	protected.Put("/stores/:storeID/subscription", subscriptionHandler.ChangePlan)
	admin := api.Group("/admin", middleware.AuthRequired(tokenManager), middleware.RequireRoles(auth.RolePlatformAdmin))
	admin.Get("/subscriptions", subscriptionHandler.AdminListAll)
	admin.Get("/stores/:storeID/subscription", subscriptionHandler.GetCurrentByStore)
	admin.Put("/stores/:storeID/subscription", subscriptionHandler.AdminChangePlan)
	admin.Patch("/stores/:storeID/subscription/status", subscriptionHandler.AdminUpdateStatus)

	return &Server{
		app: fiberApp,
		cfg: cfg,
	}, nil
}

func (s *Server) Start() error {
	return s.app.Listen(s.cfg.HTTPAddress())
}
