package expense

import "errors"

var (
	ErrExpenseStoreIDRequired = errors.New("storeID is required")
	ErrExpenseForbidden       = errors.New("user cannot operate this store")
	ErrExpenseDeleteForbidden = errors.New("only the store owner can delete expenses")
	ErrExpenseEditForbidden   = errors.New("only the store owner or manager can edit expenses")
	ErrExpenseNotFound        = errors.New("expense not found")
	ErrInvalidExpenseDate     = errors.New("expense_date must be a valid date (YYYY-MM-DD)")
	ErrInvalidDescription     = errors.New("description is required")
	ErrInvalidAmount          = errors.New("amount must be greater than zero")
	ErrInvalidPaymentMethod   = errors.New("payment_method must be one of: cash, bank_transfer, promptpay, credit_card, debit_card, cheque")
	ErrInvalidCategory        = errors.New("category_id must reference an active expense category of this store")
	ErrCategoryNotFound       = errors.New("expense category not found")
	ErrCategoryNameRequired   = errors.New("category name is required")
	ErrCategoryDuplicate      = errors.New("an expense category with this name already exists")
)
