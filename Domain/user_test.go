package domain_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper function to create a valid user for tests, reducing boilerplate.
func createValidUser() *domain.User {
	return &domain.User{
		Username: "johndoe",
		Password: "a_strong_password",
		Email: "johndoe@example.com",
		Role: domain.RoleUser,
	}
}

func TestUser_validate(t *testing.T) {
	// We use table-driven tests to check multiple scenarios cleanly.
	testCase := []struct {
		name          string      // A descriptive name for the test case
		userModifier  func(u *domain.User) // A function to modify a valid user to make it invalid
		expectedError error       // The error we expect to get back
	}{
			{
			name: "valid user",
			userModifier: func (u *domain.User)  {
				// No modification needed, it's already valid
			},
			expectedError: nil,
		},
		{
			name: "Empty username",
			userModifier: func (u *domain.User)  {
				u.Username = ""
			},
			expectedError: domain.ErrUsernameEmpty,
		},
		{
				name: "Username Too Long",
				userModifier: func(u *domain.User) {
					u.Username = "this_is_a_very_long_username_that_is_definitely_over_fifty_characters"
				},
				expectedError: domain.ErrUsernameTooLong,
			},
			{
				name: "Empty Password",
				userModifier: func(u *domain.User) {
					u.Password = ""
				},
				expectedError: domain.ErrPasswordEmpty,
			},
			{
				name: "Password Too Short",
				userModifier: func(u *domain.User) {
					u.Password = "short"
				},
				expectedError: domain.ErrPasswordTooShort,
			},
			{
				name: "Invalid Email Format - No @",
				userModifier: func(u *domain.User) {
					u.Email = "invalid-email.com"
				},
				expectedError: domain.ErrInvalidEmailFormat,
			},
			{
				name: "Invalid Email Format - No Domain",
				userModifier: func(u *domain.User) {
					u.Email = "invalid@email"
				},
				expectedError: domain.ErrInvalidEmailFormat,
			},
			{
				name: "Invalid Role",
				userModifier: func(u *domain.User) {
					u.Role = "not_a_valid_role"
				},
				expectedError: domain.ErrInvalidRole,
			},
	}
	// Loop through all test cases
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange: Create a valid user and then modify it for the specific test case
			user := createValidUser()
			tc.userModifier(user)

			// Act: Run the validation method
			err := user.Validate()

			// Assert: Check if the error returned is the one we expected
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	testCases := []struct {
		name     string
		role     domain.Role
		expected bool
	}{
		{
			name:     "Valid Role User",
			role:     domain.RoleUser,
			expected: true,
		},
		{
			name:     "Valid Role Admin",
			role:     domain.RoleAdmin,
			expected: true,
		},
		{
			name:     "Invalid Role",
			role:     "moderator",
			expected: false,
		},
		{
			name:     "Empty Role",
			role:     "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			isValid := tc.role.IsValid()

			// Assert
			assert.Equal(t, tc.expected, isValid)
		})
	}
}