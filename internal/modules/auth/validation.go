package auth

import (
	"net/mail"
	"strings"
)

func validateRegister(input RegisterRequest) error {
	if strings.TrimSpace(input.Name) == "" {
		return ErrInvalidName
	}

	if _, err := mail.ParseAddress(strings.TrimSpace(input.Email)); err != nil {
		return ErrInvalidEmail
	}

	if len(strings.TrimSpace(input.Password)) < 8 {
		return ErrInvalidPassword
	}

	if !isValidRole(strings.TrimSpace(input.Role)) {
		return ErrInvalidRole
	}

	return nil
}

func validateLogin(input LoginRequest) error {
	if _, err := mail.ParseAddress(strings.TrimSpace(input.Email)); err != nil {
		return ErrInvalidEmail
	}

	if strings.TrimSpace(input.Password) == "" {
		return ErrInvalidCredentials
	}

	return nil
}

func isValidRole(role string) bool {
	switch role {
	case RolePlatformAdmin, RoleOwner, RoleManager, RoleCashier:
		return true
	default:
		return false
	}
}
