package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/usersettings"
)

func newUserSettingsHandler(db *gorm.DB) usersettings.Handler {
	repo := usersettings.NewPostgresRepository(db)
	service := usersettings.NewService(repo)
	return usersettings.NewHandler(service)
}

func registerUserSettingsRoutes(protected fiber.Router, deps appDependencies) {
	group := protected.Group("/me/card-settings")
	group.Get("", deps.userSettingsHandler.GetCardSettings)
	group.Put("", deps.userSettingsHandler.UpdateCardSettings)
}
