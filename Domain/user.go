package domain

import (
	"net/mail" // The standard library's mail package is good for format validation.
	"regexp"
	"time"
)

type Role string

const (
	RoleUser 		Role = "user"
	RoleAdmin 	Role = "admin"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// User represents the bare minimum for a user in the application.
type User struct{
	ID 							string
	Username 				string
	Password 				string
	Email 					string
	IsActive        bool
	Role 						Role
	Bio 						string
	ProfilePicture 	string
	CreatedAt 			time.Time
	UpdatedAt 			time.Time
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
	if u.Password == "" {
		return ErrPasswordEmpty
	}
	// Note: We check password length on the raw password, not the hash.
	if len(u.Password) < 8 {
		return ErrPasswordTooShort
	}
	if _, err := mail.ParseAddress(u.Email); err != nil || !emailRegex.MatchString(u.Email) {
		return ErrInvalidEmailFormat
	}
	if !u.Role.IsValid() {
		return ErrInvalidRole
	}
	return nil
}