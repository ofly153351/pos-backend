package expense

import "time"

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// AllowedPaymentMethods — six canonical keys shown in the UI, plus the four
// legacy keys (transfer/qr/credit/card) kept valid so pre-existing expense rows
// stay editable until migration 021 normalizes them.
var AllowedPaymentMethods = map[string]bool{
	"cash":          true,
	"bank_transfer": true,
	"promptpay":     true,
	"credit_card":   true,
	"debit_card":    true,
	"cheque":        true,
	// legacy — accepted for backward compatibility, not offered in the UI
	"transfer": true,
	"qr":       true,
	"credit":   true,
	"card":     true,
}

// DefaultCategoryNames seeds new stores (user-editable afterwards). Kept in sync
// with the SQL seed in init-db/020_expenses.sql so a store gets the SAME nine
// categories whether it was seeded by the migration (existing stores) or lazily
// by the service on first category list (new stores) — no language/granularity
// drift between the two paths.
var DefaultCategoryNames = []string{
	"ค่าเช่า",
	"ค่าน้ำค่าไฟ",
	"ค่าน้ำมัน",
	"ค่าขนส่ง",
	"เงินเดือน",
	"ค่าซ่อมบำรุง",
	"อุปกรณ์สำนักงาน",
	"การตลาด",
	"อื่นๆ",
}

type ExpenseCategory struct {
	ID        string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID   string    `json:"store_id" gorm:"column:store_id"`
	Name      string    `json:"name" gorm:"column:name"`
	IsActive  bool      `json:"is_active" gorm:"column:is_active"`
	SortOrder int       `json:"sort_order" gorm:"column:sort_order"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (ExpenseCategory) TableName() string {
	return "expense_categories"
}

type Expense struct {
	ID            string    `json:"id" gorm:"column:id;primaryKey"`
	StoreID       string    `json:"store_id" gorm:"column:store_id"`
	ExpenseDate   time.Time `json:"expense_date" gorm:"column:expense_date"`
	CategoryID    string    `json:"category_id" gorm:"column:category_id"`
	Description   string    `json:"description" gorm:"column:description"`
	Amount        float64   `json:"amount" gorm:"column:amount"`
	PaymentMethod string    `json:"payment_method" gorm:"column:payment_method"`
	Note          string    `json:"note,omitempty" gorm:"column:note"`
	Status        string    `json:"status" gorm:"column:status"`
	CreatedBy     string    `json:"created_by" gorm:"column:created_by"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"column:updated_at"`

	// Joined display columns — read-only, populated by repository SELECTs only.
	CategoryName  string `json:"category_name" gorm:"->;column:category_name"`
	CreatedByName string `json:"created_by_name" gorm:"->;column:created_by_name"`
}

func (Expense) TableName() string {
	return "expenses"
}

type ExpenseListQuery struct {
	From          string // YYYY-MM-DD (inclusive)
	To            string // YYYY-MM-DD (inclusive)
	CategoryID    string
	PaymentMethod string
	Page          int
	Limit         int
}

type ExpenseListResult struct {
	Items      []Expense `json:"items"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	Total      int64     `json:"total"`
	TotalPages int       `json:"total_pages"`
}

type CategoryTotal struct {
	CategoryID string  `json:"category_id"`
	Name       string  `json:"name"`
	Total      float64 `json:"total"`
}

type MonthTotal struct {
	Month string  `json:"month"` // YYYY-MM
	Total float64 `json:"total"`
}

type ExpenseSummary struct {
	MonthlyTotal     float64         `json:"monthly_total"`
	MonthlyCount     int64           `json:"monthly_count"`
	AverageAmount    float64         `json:"average_amount"`
	TopCategoryName  string          `json:"top_category_name"`
	TopCategoryTotal float64         `json:"top_category_total"`
	ByCategory       []CategoryTotal `json:"by_category"`
	Trend            []MonthTotal    `json:"trend"`
}

type CreateExpenseRequest struct {
	ExpenseDate   string  `json:"expense_date"`
	CategoryID    string  `json:"category_id"`
	Description   string  `json:"description"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	Note          string  `json:"note"`
}

type UpdateExpenseRequest struct {
	ExpenseDate   *string  `json:"expense_date"`
	CategoryID    *string  `json:"category_id"`
	Description   *string  `json:"description"`
	Amount        *float64 `json:"amount"`
	PaymentMethod *string  `json:"payment_method"`
	Note          *string  `json:"note"`
}

type CreateCategoryRequest struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	Name      *string `json:"name"`
	IsActive  *bool   `json:"is_active"`
	SortOrder *int    `json:"sort_order"`
}
