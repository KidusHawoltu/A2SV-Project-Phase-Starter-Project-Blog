package domain

import "errors"

// Custom errors for domain and application logic
var (
	// Domain validation errors
	ErrUsernameEmpty      = errors.New("username cannot be empty")
	ErrUsernameTooLong    = errors.New("username cannot exceed 50 characters")
	ErrPasswordEmpty      = errors.New("password cannot be empty")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrInvalidRole        = errors.New("invalid role provided")
	ErrValidation         = errors.New("validation error")

	// Application-level errors
	ErrEmailExists          = errors.New("a user with this email already exists")
	ErrAuthenticationFailed = errors.New("authentication failed: invalid credentials")
	ErrUserNotFound         = errors.New("user not found")
	ErrPermissionDenied     = errors.New("permission denied")
)
