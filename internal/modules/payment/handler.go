// Package payment exposes payment helper endpoints (dynamic PromptPay QR).
package payment

import (
	"context"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"pos-backend/internal/middleware"
	"pos-backend/internal/platform/httpx"
	"pos-backend/internal/platform/receipthtml"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) Handler {
	return Handler{db: db}
}

// GenerateQR → GET /stores/:storeID/promptpay-qr?amount=123.45
// Returns a dynamic PromptPay QR (data URI) for the store's promptpay_id + amount.
func (h Handler) GenerateQR(c *fiber.Ctx) error {
	storeID := c.Params("storeID")
	if strings.TrimSpace(storeID) == "" {
		return httpx.ErrBadRequest(c, "storeID is required")
	}

	// Authorisation: caller must be able to operate this store.
	claims := middleware.ClaimsFromContext(c)
	if !h.userCanOperate(c.UserContext(), storeID, claims.UserID, claims.Role) {
		return httpx.ErrForbidden(c, "user cannot operate this store")
	}

	amount, _ := strconv.ParseFloat(strings.TrimSpace(c.Query("amount")), 64)
	if amount < 0 {
		amount = 0
	}

	var promptPayID string
	_ = h.db.WithContext(c.UserContext()).
		Table("stores").
		Select("COALESCE(promptpay_id, '')").
		Where("id = ?", storeID).
		Scan(&promptPayID).Error
	promptPayID = strings.TrimSpace(promptPayID)

	if promptPayID == "" {
		return httpx.Err422(c, "promptpay_id", "store has no PromptPay number configured")
	}

	qr := receipthtml.PromptPayQRDataURI(promptPayID, amount)
	if qr == "" {
		return httpx.Err422(c, "promptpay_id", "invalid PromptPay number")
	}

	return httpx.Success(c, fiber.StatusOK, "promptpay qr generated", fiber.Map{
		"qr":           qr,
		"amount":       amount,
		"promptpay_id": promptPayID,
	})
}

func (h Handler) userCanOperate(ctx context.Context, storeID, userID, role string) bool {
	if role == "platform_admin" {
		return true
	}
	var count int64
	h.db.WithContext(ctx).
		Table("store_members").
		Where("store_id = ? AND user_id = ? AND role IN ?", storeID, userID, []string{"owner", "manager", "cashier"}).
		Count(&count)
	return count > 0
}
