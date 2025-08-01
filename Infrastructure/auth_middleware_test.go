package infrastructure_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	// Setup a minimal gin router with the middleware for testing
	gin.SetMode(gin.TestMode)

	// Setup a real JWT service
	jwtService := infrastructure.NewJWTService("test-secret", "test-issuer", 1*time.Minute)

	// Create a test user ID and role
	userID := "user-abc-123"
	userRole := domain.RoleUser

	testCases := []struct {
		name           string
		token          string
		expectedStatus int
		expectedBody   string
		setupRequest   func(req *http.Request)
	}{
		{
			name:           "Success - Valid Token",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"userID":"user-abc-123","userRole":"user"}`,
			setupRequest: func(req *http.Request) {
				token, _ := jwtService.GenerateAccessToken(userID, userRole)
				req.Header.Set("Authorization", "Bearer "+token)
			},
		},
		{
			name:           "Failure - No Authorization Header",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authorization header is required"}`,
			setupRequest:   func(req *http.Request) {},
		},
		{
			name:           "Failure - Malformed Header (No Bearer)",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Authorization header format must be Bearer {token}"}`,
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "invalid-token")
			},
		},
		{
			name:           "Failure - Invalid Token (Wrong Secret)",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid or expired token"}`,
			setupRequest: func(req *http.Request) {
				otherJwtService := infrastructure.NewJWTService("wrong-secret", "test-issuer", 1*time.Minute)
				token, _ := otherJwtService.GenerateAccessToken(userID, userRole)
				req.Header.Set("Authorization", "Bearer "+token)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			router := gin.New()
			// Apply the middleware to a test route
			router.GET("/test", infrastructure.AuthMiddleware(jwtService), func(c *gin.Context) {
				// This handler will only be reached if middleware passes
				id, _ := c.Get("userID")
				role, _ := c.Get("userRole")
				c.JSON(http.StatusOK, gin.H{"userID": id, "userRole": role})
			})

			// Create a test request and response recorder
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			tc.setupRequest(req)

			// Act
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}