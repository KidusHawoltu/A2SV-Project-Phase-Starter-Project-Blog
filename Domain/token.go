package domain

import (
	"time"
)

// TokenType helps differentiate between different kinds of tokens.
type TokenType string

const (
	TokenTypeRefresh       TokenType = "refresh"
	TokenTypePasswordReset TokenType = "password_reset"
	TokenTypeAccessToken   TokenType = "access"
	TokenTypeActivation    TokenType = "activation"
)

// Token represents a temporary token stored for a user.
type Token struct {
	ID       	string
	UserID		string
	Type   		TokenType
	Value			string
	ExpiresAt	time.Time
}

func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}