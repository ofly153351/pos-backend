package auth

import "errors"

var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidName        = errors.New("name is required")
	ErrInvalidEmail       = errors.New("valid email is required")
	ErrInvalidPassword    = errors.New("password must be at least 8 characters")
	ErrInvalidRole        = errors.New("valid role is required")
)
