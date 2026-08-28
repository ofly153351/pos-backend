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
	g := newStoreGuards(deps)
	// NOTE: register /summary before /:expenseID so Fiber matches it first.
	protected.Get("/stores/:storeID/expenses/summary", g.operate, handler.Summary)
	protected.Get("/stores/:storeID/expenses", g.operate, handler.List)
	protected.Post("/stores/:storeID/expenses", g.operate, handler.Create)
	protected.Get("/stores/:storeID/expenses/:expenseID", g.operate, handler.GetByID)
	protected.Patch("/stores/:storeID/expenses/:expenseID", g.manage, handler.Update)
	protected.Delete("/stores/:storeID/expenses/:expenseID", g.owner, handler.Delete)
	protected.Get("/stores/:storeID/expense-categories", g.operate, handler.ListCategories)
	protected.Post("/stores/:storeID/expense-categories", g.manage, handler.CreateCategory)
	protected.Patch("/stores/:storeID/expense-categories/:categoryID", g.manage, handler.UpdateCategory)
	protected.Delete("/stores/:storeID/expense-categories/:categoryID", g.manage, handler.DeactivateCategory)
}
