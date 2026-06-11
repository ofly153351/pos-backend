package app

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/modules/expense"
)

func newExpenseHandler(db *gorm.DB) expense.Handler {
	repo := expense.NewPostgresRepository(db)
	service := expense.NewService(repo)
	return expense.NewHandler(service)
}

func registerExpenseRoutes(protected fiber.Router, deps appDependencies) {
	handler := deps.expenseHandler.(expense.Handler)
	// NOTE: register /summary before /:expenseID so Fiber matches it first.
	protected.Get("/stores/:storeID/expenses/summary", handler.Summary)
	protected.Get("/stores/:storeID/expenses", handler.List)
	protected.Post("/stores/:storeID/expenses", handler.Create)
	protected.Get("/stores/:storeID/expenses/:expenseID", handler.GetByID)
	protected.Patch("/stores/:storeID/expenses/:expenseID", handler.Update)
	protected.Delete("/stores/:storeID/expenses/:expenseID", handler.Delete)
	protected.Get("/stores/:storeID/expense-categories", handler.ListCategories)
	protected.Post("/stores/:storeID/expense-categories", handler.CreateCategory)
	protected.Patch("/stores/:storeID/expense-categories/:categoryID", handler.UpdateCategory)
	protected.Delete("/stores/:storeID/expense-categories/:categoryID", handler.DeactivateCategory)
}
