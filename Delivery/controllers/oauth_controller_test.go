package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "A2SV_Starter_Project_Blog/Delivery/controllers"
	usecases "A2SV_Starter_Project_Blog/Usecases"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock IOAuthUsecase ---
type MockOAuthUsecase struct {
	mock.Mock
}

func (m *MockOAuthUsecase) HandleGoogleCallback(ctx context.Context, code string) (string, string, error) {
	args := m.Called(ctx, code)
	return args.String(0), args.String(1), args.Error(2)
}

// --- Test Suite Setup ---
type OAuthControllerTestSuite struct {
	suite.Suite
	mockOAuthUsecase *MockOAuthUsecase
	router           *gin.Engine
	controller       *OAuthController
}

func (s *OAuthControllerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.mockOAuthUsecase = new(MockOAuthUsecase)
	s.controller = NewOAuthController(s.mockOAuthUsecase)
	s.router = gin.New()
	// Register the endpoint for the test
	s.router.POST("/auth/google/callback", s.controller.HandleGoogleCallback)
}

func TestOAuthControllerTestSuite(t *testing.T) {
	suite.Run(t, new(OAuthControllerTestSuite))
}

// --- Tests ---

func (s *OAuthControllerTestSuite) TestHandleGoogleCallback() {
	s.Run("Success", func() {
		s.SetupTest()
		// Arrange
		authCode := "valid-google-auth-code"
		expectedAccessToken := "our.app.access.token"
		expectedRefreshToken := "our.app.refresh.token"

		// Set the mock expectation for the usecase
		s.mockOAuthUsecase.On("HandleGoogleCallback", mock.Anything, authCode).
			Return(expectedAccessToken, expectedRefreshToken, nil).
			Once()

		// Create the request body
		reqBody := GoogleCallbackRequest{Code: authCode}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/google/callback", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		s.router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code)

		var resp AuthTokensResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		s.NoError(err)
		s.Equal(expectedAccessToken, resp.AccessToken)
		s.Equal(expectedRefreshToken, resp.RefreshToken)

		s.mockOAuthUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure - Missing code in request body", func() {
		s.SetupTest()
		// Arrange: Send an empty JSON object
		invalidBody := `{}`
		req := httptest.NewRequest(http.MethodPost, "/auth/google/callback", strings.NewReader(invalidBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		s.router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusBadRequest, w.Code, "Expected Bad Request due to validation failure")
		// The usecase should NOT have been called.
		s.mockOAuthUsecase.AssertNotCalled(s.T(), "HandleGoogleCallback", mock.Anything, mock.Anything)
	})

	s.Run("Failure - Usecase returns an error", func() {
		s.SetupTest()
		// Arrange
		authCode := "code-that-will-fail"

		// Set the mock to return a conflict error (e.g., email already exists with local provider)
		s.mockOAuthUsecase.On("HandleGoogleCallback", mock.Anything, authCode).
			Return("", "", usecases.ErrConflict).
			Once()

		reqBody := GoogleCallbackRequest{Code: authCode}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/google/callback", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		s.router.ServeHTTP(w, req)

		// Assert
		// Assuming your HandleError function maps ErrConflict to 409 Conflict
		s.Equal(http.StatusConflict, w.Code)
		s.mockOAuthUsecase.AssertExpectations(s.T())
	})
}
