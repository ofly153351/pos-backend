package customer_test

import (
	"testing"

	"pos-backend/internal/modules/customer"
)

func TestValidate_NameRequired(t *testing.T) {
	svc := customer.Service{}
	err := svc.ValidateCustomerInput("", "user@email.com", 1)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if err.Error() != customer.ErrInvalidCustomerName.Error() {
		t.Errorf("expected ErrInvalidCustomerName, got %v", err)
	}
}

func TestValidate_InvalidEmail(t *testing.T) {
	svc := customer.Service{}
	err := svc.ValidateCustomerInput("Alice", "not-an-email", 1)
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	if err.Error() != customer.ErrInvalidCustomerEmail.Error() {
		t.Errorf("expected ErrInvalidCustomerEmail, got %v", err)
	}
}

func TestValidate_EmptyEmailAllowed(t *testing.T) {
	svc := customer.Service{}
	err := svc.ValidateCustomerInput("Alice", "", 1)
	if err != nil {
		t.Errorf("expected no error for empty email, got %v", err)
	}
}

func TestValidate_LevelZero(t *testing.T) {
	svc := customer.Service{}
	err := svc.ValidateCustomerInput("Alice", "", 0)
	if err == nil {
		t.Fatal("expected error for level=0")
	}
	if err.Error() != customer.ErrInvalidLevel.Error() {
		t.Errorf("expected ErrInvalidLevel, got %v", err)
	}
}

func TestValidate_NegativeLevel(t *testing.T) {
	svc := customer.Service{}
	err := svc.ValidateCustomerInput("Alice", "", -1)
	if err == nil {
		t.Fatal("expected error for negative level")
	}
}

func TestValidate_ValidInput(t *testing.T) {
	svc := customer.Service{}
	err := svc.ValidateCustomerInput("Alice", "alice@example.com", 2)
	if err != nil {
		t.Errorf("expected no error for valid input, got %v", err)
	}
}
