package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/activity_log"
)

func newActivityLogHandler(db *gorm.DB) activity_log.Handler {
	repo := activity_log.NewRepository(db)
	svc := activity_log.NewService(repo)
	return activity_log.NewHandler(svc)
}

func registerActivityLogRoutes(router fiber.Router, deps appDependencies) {
	g := newStoreGuards(deps)
	router.Get("/stores/:storeID/activity-logs", g.operate, deps.activityLogHandler.List)
}
