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
	ErrNotFound             = errors.New("resource not found")
	ErrEmailExists          = errors.New("a user with this email already exists")
	ErrAuthenticationFailed = errors.New("authentication failed: invalid credentials")
	ErrUserNotFound         = errors.New("user not found")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrUsernameExists       = errors.New("username already exists")
	ErrOAuthUser            = errors.New("this action is not applicable to an account created with an external provider")
	ErrCannotChangeOwnRole  = errors.New("admins cannot change their own role")

	// Token errors
	ErrInvalidID              = errors.New("invalid ID was used")
	ErrInvalidResetToken      = errors.New("invalid or expired password reset token")
	ErrCannotDemoteSelf       = errors.New("admin cannot demote themselves")
	ErrAccountNotActive       = errors.New("this account has not been activated")
	ErrInvalidActivationToken = errors.New("invalid or expired activation token")
)
