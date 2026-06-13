package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/member"
)

func newMemberHandler(db *gorm.DB) member.Handler {
	repo := member.NewPostgresRepository(db)
	service := member.NewService(repo)
	return member.NewHandler(service)
}

func registerMemberRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.memberHandler.(member.Handler)
	protected.Get("/stores/:storeID/members", handler.List)
	protected.Post("/stores/:storeID/members", handler.Add)
	protected.Patch("/stores/:storeID/members/:userID", handler.Update)
	protected.Delete("/stores/:storeID/members/:userID", handler.Remove)
}
