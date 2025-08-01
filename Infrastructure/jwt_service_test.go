package infrastructure_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	"testing"
	"time"

	// <-- CHANGE THIS
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService(t *testing.T) {
	secret := "my-super-secret-key-for-testing"
	issuer := "test-issuer"
	ttl := 15 * time.Minute

	jwtService := infrastructure.NewJWTService(secret, issuer, ttl)
	userID := "user-123"
	userRole := domain.RoleUser

	t.Run("Generate and Validate a valid token", func(t *testing.T) {
		// Act: Generate
		tokenString, err := jwtService.GenerateAccessToken(userID, userRole)
		require.NoError(t, err)
		require.NotEmpty(t, tokenString)

		// Act: Validate
		claims, err := jwtService.ValidateToken(tokenString)
		require.NoError(t, err)
		require.NotNil(t, claims)

		// Assert
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, userRole, claims.Role)
		assert.Equal(t, issuer, claims.Issuer)
	})

	t.Run("Fail to validate token with wrong secret", func(t *testing.T) {
		// Arrange
		tokenString, _ := jwtService.GenerateAccessToken(userID, userRole)
		otherService := infrastructure.NewJWTService("another-secret", issuer, ttl)

		// Act
		_, err := otherService.ValidateToken(tokenString)

		// Assert
		assert.Error(t, err)
	})
}