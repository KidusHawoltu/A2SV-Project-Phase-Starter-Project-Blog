package domain

import (
	"net/mail" // The standard library's mail package is good for format validation.
	"regexp"
	"time"
)

type Role string
type AuthProvider string
type AuthProvider string

const (
	RoleUser       Role         = "user"
	RoleAdmin      Role         = "admin"
	ProviderLocal  AuthProvider = "local"
	ProviderGoogle AuthProvider = "google"
	RoleUser       Role         = "user"
	RoleAdmin      Role         = "admin"
	ProviderLocal  AuthProvider = "local"
	ProviderGoogle AuthProvider = "google"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// User represents the bare minimum for a user in the application.
type User struct {
	ID             string
	Username       string
	Password       *string
	Email          string
	IsActive       bool
	Role           Role
	Bio            string
	ProfilePicture string

	Provider   AuthProvider
	ProviderID string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserSearchFilterOptions struct {
	Username *string // Pointer for optional search
	Email    *string // Pointer for optional search
	Role     *Role   // Pointer to filter by a specific role
	IsActive *bool   // Pointer to filter by active/inactive status
	Provider *AuthProvider

	// AND or OR
	GlobalLogic GlobalLogic

	// Date range for when the user was created
	StartDate *time.Time
	EndDate   *time.Time

	// Pagination
	Page  int64
	Limit int64

	// Sorting
	SortBy    string    // e.g., "username", "email", "createdAt"
	SortOrder SortOrder // ASC or DESC (reusing from blog domain)
}

// IsValid checks if the role is one of the predefined valid roles.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleUser:
		return true
	}
	return false
}

// Validate performs intrinsic validation on the User struct fields.
func (u *User) Validate() error {
	if u.Username == "" {
		return ErrUsernameEmpty
	}
	if len(u.Username) > 50 {
		return ErrUsernameTooLong
	}
	if u.Provider == ProviderLocal {
		if u.Password == nil || *u.Password == "" {
			return ErrPasswordEmpty
		}
		if len(*u.Password) < 8 {
			return ErrPasswordTooShort
		}
	if u.Provider == ProviderLocal {
		if u.Password == nil || *u.Password == "" {
			return ErrPasswordEmpty
		}
		if len(*u.Password) < 8 {
			return ErrPasswordTooShort
		}
	}
	if _, err := mail.ParseAddress(u.Email); err != nil || !emailRegex.MatchString(u.Email) {
		return ErrInvalidEmailFormat
	}
	if !u.Role.IsValid() {
		return ErrInvalidRole
	}
	return nil

}

