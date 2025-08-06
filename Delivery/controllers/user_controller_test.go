package controllers_test

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"bytes"
	"context"

	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock UserUsecase ---
type MockUserUsecase struct {
	mock.Mock
}

func (m *MockUserUsecase) Register(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *MockUserUsecase) Login(ctx context.Context, identifier, password string) (string, string, error) {
	args := m.Called(ctx, identifier, password)
	return args.String(0), args.String(1), args.Error(2)
}
func (m *MockUserUsecase) Logout(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}
func (m *MockUserUsecase) RefreshAccessToken(ctx context.Context, oldAccess, oldRefresh string) (string, string, error) {
	args := m.Called(ctx, oldAccess, oldRefresh)
	return args.String(0), args.String(1), args.Error(2)
}
func (m *MockUserUsecase) ForgetPassword(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}
func (m *MockUserUsecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}
func (m *MockUserUsecase) UpdateProfile(ctx context.Context, userID, bio, profilePicURL string) (*domain.User, error) {
	args := m.Called(ctx, userID, bio, profilePicURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserUsecase) ActivateAccount(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockUserUsecase) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)

}

// --- USER ROUTER SETUP HELPER ---
func setupUserRouter(uc usecases.UserUsecase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userController := controllers.NewUserController(uc)

	auth := router.Group("/auth")
	{
		auth.POST("/register", userController.Register)
		auth.GET("/activate", userController.ActivateAccount)
		auth.POST("/login", userController.Login)
		auth.POST("/logout", userController.Logout)
		auth.POST("/refresh", userController.RefreshToken)
	}
	password := router.Group("/password")
	{
		password.POST("/forget", userController.ForgetPassword)
		password.POST("/reset", userController.ResetPassword)
	}
	profile := router.Group("/profile")
	{
		profile.Use(func(c *gin.Context) {
			c.Set("userID", "test-user-id")
			c.Next()
		})
		profile.PUT("", userController.UpdateProfile)
	}
	return router
}

// --- TEST FUNCTIONS ---

func TestUserController_Register(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	t.Run("Success", func(t *testing.T) {
		reqPayload := controllers.RegisterRequest{Username: "test", Email: "test@test.com", Password: "password123"}
		mockUsecase.On("Register", mock.Anything, mock.AnythingOfType("*domain.User")).
			Return(nil).
			Run(func(args mock.Arguments) {
				args.Get(1).(*domain.User).ID = "mock-id-123"
			}).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), `"id":"mock-id-123"`)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("Failure - Email Exists", func(t *testing.T) {
		reqPayload := controllers.RegisterRequest{Username: "test", Email: "exists@test.com", Password: "password123"}
		mockUsecase.On("Register", mock.Anything, mock.AnythingOfType("*domain.User")).Return(domain.ErrEmailExists).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserController_Login(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	t.Run("Success", func(t *testing.T) {
		reqPayload := controllers.LoginRequest{Email: "test@test.com", Password: "password123"}
		mockUsecase.On("Login", mock.Anything, reqPayload.Email, reqPayload.Password).
			Return("new.access.token", "new.refresh.token", nil).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "new.access.token")
		assert.Contains(t, w.Body.String(), "new.refresh.token")
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserController_Logout(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	t.Run("Success", func(t *testing.T) {
		reqPayload := controllers.LogoutRequest{RefreshToken: "valid.refresh.token"}
		mockUsecase.On("Logout", mock.Anything, "valid.refresh.token").Return(nil).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/logout", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Successfully logged out")
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserController_RefreshToken(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	t.Run("Success", func(t *testing.T) {
		reqPayload := controllers.RefreshRequest{AccessToken: "old.access", RefreshToken: "old.refresh"}
		mockUsecase.On("RefreshAccessToken", mock.Anything, "old.access", "old.refresh").
			Return("new.access.token", "new.refresh.token", nil).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "new.access.token")
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserController_ForgetPassword(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	t.Run("Success", func(t *testing.T) {
		reqPayload := controllers.ForgetPasswordRequest{Email: "user@example.com"}
		mockUsecase.On("ForgetPassword", mock.Anything, "user@example.com").Return(nil).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/password/forget", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "a password reset link has been sent")
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserController_ResetPassword(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	t.Run("Success", func(t *testing.T) {
		reqPayload := controllers.ResetPasswordRequest{Token: "valid.token", NewPassword: "new-password123"}
		mockUsecase.On("ResetPassword", mock.Anything, "valid.token", "new-password123").Return(nil).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/password/reset", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "password has been reset successfully")
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserController_ActivateAccount(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	t.Run("Success", func(t *testing.T) {
		mockUsecase.On("ActivateAccount", mock.Anything, "valid.token").Return(nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/auth/activate?token=valid.token", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Account activated successfully")
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserController_UpdateProfile(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	t.Run("Success", func(t *testing.T) {
		reqPayload := controllers.UpdateProfileRequest{Bio: "new bio", ProfilePicURL: "new.url"}
		updatedUser := &domain.User{ID: "test-user-id", Bio: "new bio", ProfilePicture: "new.url"}
		mockUsecase.On("UpdateProfile", mock.Anything, "test-user-id", "new bio", "new.url").
			Return(updatedUser, nil).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"bio":"new bio"`)
		mockUsecase.AssertExpectations(t)
	})
}
