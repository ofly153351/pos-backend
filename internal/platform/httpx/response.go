package httpx

import (
	"github.com/gofiber/fiber/v2"
	"pos-backend/internal/platform/fieldcheck"
)

// ── Envelope ──────────────────────────────────────────────────────────────────

type envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// ── Structured error types ────────────────────────────────────────────────────

// ErrCode is a machine-readable error identifier the frontend can switch on.
type ErrCode string

const (
	CodeValidation   ErrCode = "VALIDATION_ERROR"
	CodeNotFound     ErrCode = "NOT_FOUND"
	CodeForbidden    ErrCode = "FORBIDDEN"
	CodeConflict     ErrCode = "CONFLICT"
	CodeUnauthorized ErrCode = "UNAUTHORIZED"
	CodeInternal     ErrCode = "INTERNAL_ERROR"
	CodeBadRequest   ErrCode = "BAD_REQUEST"
)

// FieldError describes a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	// Code is a stable machine code (see platform/fieldcheck) the frontend
	// translates via th.json/en.json; Message stays as the traceable fallback.
	Code string `json:"code,omitempty"`
}

// APIError is the structured error object placed in envelope.Error.
type APIError struct {
	Code    ErrCode      `json:"code"`
	Message string       `json:"message,omitempty"`
	Fields  []FieldError `json:"fields,omitempty"`
	// Details carries an optional machine-readable payload (e.g. a deletion blocker
	// code + dependency assessment) so the frontend can drive an adaptive remediation
	// modal without parsing the human-readable message. Omitted when nil.
	Details any `json:"details,omitempty"`
}

// ── Constructors ──────────────────────────────────────────────────────────────

func JSON(c *fiber.Ctx, status int, payload any) error {
	return c.Status(status).JSON(payload)
}

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return JSON(c, status, envelope{Success: true, Message: message, Data: data})
}

// Error writes a structured error response.
// detail may be nil, a string, an APIError, or any serialisable value.
func Error(c *fiber.Ctx, status int, message string, detail any) error {
	return JSON(c, status, envelope{Success: false, Message: message, Error: detail})
}

// ValidationError writes a 422 with field-level errors.
func ValidationError(c *fiber.Ctx, fields ...FieldError) error {
	return JSON(c, fiber.StatusUnprocessableEntity, envelope{
		Success: false,
		Message: "validation failed",
		Error: APIError{
			Code:   CodeValidation,
			Fields: fields,
		},
	})
}

// Err422 shorthand: single field validation failure.
func Err422(c *fiber.Ctx, field, message string) error {
	return ValidationError(c, FieldError{Field: field, Message: message})
}

// Err422Code is Err422 with a stable machine code for localized messages (B23).
func Err422Code(c *fiber.Ctx, field, code, message string) error {
	return ValidationError(c, FieldError{Field: field, Message: message, Code: code})
}

// FieldErrorResponse writes err as a coded 422 when it carries a
// fieldcheck.Error and reports whether it did.
func FieldErrorResponse(c *fiber.Ctx, err error) (bool, error) {
	fe, ok := fieldcheck.As(err)
	if !ok {
		return false, nil
	}
	return true, Err422Code(c, fe.Field, fe.Code, fe.Message)
}

// ErrCode responses
func ErrNotFound(c *fiber.Ctx, message string) error {
	return JSON(c, fiber.StatusNotFound, envelope{
		Success: false, Message: message,
		Error: APIError{Code: CodeNotFound, Message: message},
	})
}

func ErrForbidden(c *fiber.Ctx, message string) error {
	return JSON(c, fiber.StatusForbidden, envelope{
		Success: false, Message: message,
		Error: APIError{Code: CodeForbidden, Message: message},
	})
}

func ErrConflict(c *fiber.Ctx, message string) error {
	return JSON(c, fiber.StatusConflict, envelope{
		Success: false, Message: message,
		Error: APIError{Code: CodeConflict, Message: message},
	})
}

// ErrConflictDetails writes a 409 carrying a machine-readable detail payload (e.g. a
// deletion blocker code + dependency assessment) alongside the human message. The
// envelope error code stays CONFLICT so existing frontend code keeps working; richer
// consumers read error.details to render adaptive remediation UI.
func ErrConflictDetails(c *fiber.Ctx, message string, details any) error {
	return JSON(c, fiber.StatusConflict, envelope{
		Success: false, Message: message,
		Error: APIError{Code: CodeConflict, Message: message, Details: details},
	})
}

func ErrBadRequest(c *fiber.Ctx, message string) error {
	return JSON(c, fiber.StatusBadRequest, envelope{
		Success: false, Message: message,
		Error: APIError{Code: CodeBadRequest, Message: message},
	})
}

func ErrInternal(c *fiber.Ctx) error {
	return JSON(c, fiber.StatusInternalServerError, envelope{
		Success: false, Message: "internal server error",
		Error: APIError{Code: CodeInternal, Message: "an unexpected error occurred, please try again"},
	})
}
