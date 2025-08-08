package domain_test

import (
	"testing"

	. "A2SV_Starter_Project_Blog/Domain"

	"github.com/stretchr/testify/suite"
)

// --- Test Suite Setup ---
type UserDomainTestSuite struct {
	suite.Suite
}

func TestUserDomainTestSuite(t *testing.T) {
	suite.Run(t, new(UserDomainTestSuite))
}

// --- Tests ---

// helper function to create a valid local user for tests.
func createValidLocalUser() *User {
	password := "a_strong_password"
	return &User{
		Username: "johndoe",
		Password: &password,
		Email:    "johndoe@example.com",
		Role:     RoleUser,
		Provider: ProviderLocal,
	}
}

func (s *UserDomainTestSuite) TestUser_Validate() {
	testCases := []struct {
		name          string
		userModifier  func(u *User)
		expectedError error
	}{
		// --- LOCAL USER VALIDATION CASES ---
		{
			name:          "Valid local user",
			userModifier:  func(u *User) { /* No modification needed */ },
			expectedError: nil,
		},
		{
			name:          "Local user with empty password",
			userModifier:  func(u *User) { *u.Password = "" },
			expectedError: ErrPasswordEmpty,
		},
		{
			name:          "Local user with nil password",
			userModifier:  func(u *User) { u.Password = nil },
			expectedError: ErrPasswordEmpty,
		},
		{
			name:          "Local user with password too short",
			userModifier:  func(u *User) { shortPass := "short"; u.Password = &shortPass },
			expectedError: ErrPasswordTooShort,
		},

		// --- GENERIC VALIDATION CASES ---
		{
			name:          "Empty username",
			userModifier:  func(u *User) { u.Username = "" },
			expectedError: ErrUsernameEmpty,
		},
		{
			name:          "Invalid email",
			userModifier:  func(u *User) { u.Email = "invalid-email" },
			expectedError: ErrInvalidEmailFormat,
		},

		// --- GOOGLE USER VALIDATION CASES ---
		{
			name: "Valid Google user with nil password",
			userModifier: func(u *User) {
				u.Provider = ProviderGoogle
				u.ProviderID = "google-user-id-123"
				u.Password = nil // Google users have no password
			},
			expectedError: nil,
		},
		{
			name: "Valid Google user with empty password pointer",
			userModifier: func(u *User) {
				emptyPass := ""
				u.Provider = ProviderGoogle
				u.ProviderID = "google-user-id-123"
				u.Password = &emptyPass // This is also valid
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Arrange: Create a valid local user and then modify it.
			user := createValidLocalUser()
			tc.userModifier(user)

			// Act
			err := user.Validate()

			// Assert
			s.ErrorIs(err, tc.expectedError)
		})
	}
}

func (s *UserDomainTestSuite) TestRole_IsValid() {
	s.Run("Valid roles", func() {
		s.True(RoleUser.IsValid())
		s.True(RoleAdmin.IsValid())
	})
	s.Run("Invalid roles", func() {
		s.False(Role("moderator").IsValid())
		s.False(Role("").IsValid())
	})
}
