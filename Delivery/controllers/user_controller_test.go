package controllers_test

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"bytes"
	"context"
	"io"
	"mime/multipart"

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
func (m *MockUserUsecase) UpdateProfile(ctx context.Context, userID, bio string, profilePicFile multipart.File, profilePicHeader *multipart.FileHeader) (*domain.User, error) {
	args := m.Called(ctx, userID, bio, profilePicFile, profilePicHeader)
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
func (m *MockUserUsecase) SearchAndFilter(ctx context.Context, options domain.UserSearchFilterOptions) ([]*domain.User, int64, error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, int64(args.Int(1)), args.Error(2)
	}
	return args.Get(0).([]*domain.User), int64(args.Int(1)), args.Error(2)
}
func (m *MockUserUsecase) SetUserRole(ctx context.Context, actorUserID string, actorRole domain.Role, targetUserID string, newRole domain.Role) (*domain.User, error) {
	args := m.Called(ctx, actorUserID, actorRole, targetUserID, newRole)
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
	admin := router.Group("/admin")
	{
		admin.Use(func(c *gin.Context) {
			c.Set("userID", "admin-id-123")
			c.Set("userRole", domain.RoleAdmin)
			c.Next()
		})
		admin.GET("/users", userController.SearchAndFilter)
		admin.PATCH("/users/:userID/role", userController.SetUserRole)
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

	t.Run("Success - Update bio and profile picture", func(t *testing.T) {
		// Arrange: Create a fresh mock and router for this test
		mockUsecase := new(MockUserUsecase)
		router := setupUserRouter(mockUsecase)

		userID := "test-user-id"
		newBio := "This is my new biography."
		updatedUser := &domain.User{ID: userID, Bio: newBio, ProfilePicture: "http://example.com/new.jpg"}

		// We use mock.Anything for the file parts because mocking the exact file data is complex and not necessary.
		// We just need to ensure the usecase is called with *some* file data.
		mockUsecase.On("UpdateProfile", mock.Anything, userID, newBio, mock.Anything, mock.Anything).
			Return(updatedUser, nil).Once()

		// Create the multipart form body
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Add the 'bio' form field
		writer.WriteField("bio", newBio)

		// Add the 'profilePicture' file field
		part, err := writer.CreateFormFile("profilePicture", "test.jpg")
		assert.NoError(t, err)
		_, err = io.WriteString(part, "dummy image content")
		assert.NoError(t, err)

		// Close the writer to finalize the body
		writer.Close()

		// Act
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/profile", body)
		// Set the Content-Type header to the one generated by the multipart writer
		req.Header.Set("Content-Type", writer.FormDataContentType())

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"bio":"This is my new biography."`)
		assert.Contains(t, w.Body.String(), `"profile_picture":"http://example.com/new.jpg"`)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("Success - Update bio only (no file provided)", func(t *testing.T) {
		// Arrange
		mockUsecase := new(MockUserUsecase)
		router := setupUserRouter(mockUsecase)

		userID := "test-user-id"
		newBio := "Bio only update."
		updatedUser := &domain.User{ID: userID, Bio: newBio, ProfilePicture: "old.url"} // Picture doesn't change

		// The usecase should be called with nil for the file parts
		mockUsecase.On("UpdateProfile", mock.Anything, userID, newBio, nil, (*multipart.FileHeader)(nil)).
			Return(updatedUser, nil).Once()

		// Create a multipart form body with only the 'bio' field
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.WriteField("bio", newBio)
		writer.Close()

		// Act
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPut, "/profile", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("Failure - Invalid form data (not multipart)", func(t *testing.T) {
		// Arrange
		mockUsecase := new(MockUserUsecase)
		router := setupUserRouter(mockUsecase)

		// Act
		w := httptest.NewRecorder()
		// Send a request with a standard content type, which will cause ParseMultipartForm to fail
		req, _ := http.NewRequest(http.MethodPut, "/profile", bytes.NewBufferString("bio=test"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid form data")
		mockUsecase.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestUserController_SearchAndFilter(t *testing.T) {
	mockUsecase := new(MockUserUsecase)
	router := setupUserRouter(mockUsecase)

	// Sample response data from the usecase
	sampleUsers := []*domain.User{
		{ID: "user-1", Username: "admin1", Email: "admin1@test.com", Role: domain.RoleAdmin},
		{ID: "user-2", Username: "user1", Email: "user1@test.com", Role: domain.RoleUser},
	}
	var totalUsers int64 = 2

	t.Run("Success - No Filters", func(t *testing.T) {
		// This tests the default behavior
		expectedOptions := domain.UserSearchFilterOptions{
			Page:        1,
			Limit:       10,
			GlobalLogic: domain.GlobalLogicAND,
			SortOrder:   domain.SortOrderDESC,
		}
		mockUsecase.On("SearchAndFilter", mock.Anything, expectedOptions).
			Return(sampleUsers, int(totalUsers), nil).Once()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/admin/users", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"total":2`)
		assert.Contains(t, w.Body.String(), `"username":"admin1"`)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("Success - With All Filters", func(t *testing.T) {
		username := "testuser"
		email := "test@test.com"
		role := domain.RoleUser
		isActive := true
		provider := domain.ProviderGoogle

		// This is what the controller should parse the query params into
		expectedOptions := domain.UserSearchFilterOptions{
			Username:    &username,
			Email:       &email,
			Role:        &role,
			IsActive:    &isActive,
			Provider:    &provider,
			GlobalLogic: domain.GlobalLogicOR,
			Page:        2,
			Limit:       25,
			SortBy:      "username",
			SortOrder:   domain.SortOrderASC,
		}
		mockUsecase.On("SearchAndFilter", mock.Anything, expectedOptions).
			Return(sampleUsers, int(totalUsers), nil).Once()

		// Build the URL with query parameters
		url := "/admin/users?username=testuser&email=test@test.com&role=user&isActive=true&provider=google&logic=OR&page=2&limit=25&sortBy=username&sortOrder=ASC"

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("Failure - Invalid Page Parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/admin/users?page=invalid", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid 'page' parameter")
	})

	t.Run("Failure - Invalid Role Parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/admin/users?role=invalid", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid 'role' parameter")
	})
}

func TestUserController_SetUserRole(t *testing.T) {
	// Common variables that don't change can stay here
	targetUserID := "user-to-promote"

	t.Run("Success - Promote User", func(t *testing.T) {
		// ARRANGE: Create a fresh mock and router for THIS test.
		mockUsecase := new(MockUserUsecase)
		router := setupUserRouter(mockUsecase)

		updatedUser := &domain.User{ID: targetUserID, Role: domain.RoleAdmin}
		reqPayload := controllers.SetRoleRequest{NewRole: domain.RoleAdmin}
		mockUsecase.On("SetUserRole", mock.Anything, "admin-id-123", domain.RoleAdmin, targetUserID, domain.RoleAdmin).
			Return(updatedUser, nil).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/admin/users/"+targetUserID+"/role", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// ACT
		router.ServeHTTP(w, req)

		// ASSERT
		assert.Equal(t, http.StatusOK, w.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("Failure - Usecase returns permission denied", func(t *testing.T) {
		// ARRANGE: Create a fresh mock and router for THIS test.
		mockUsecase := new(MockUserUsecase)
		router := setupUserRouter(mockUsecase)

		reqPayload := controllers.SetRoleRequest{NewRole: domain.RoleAdmin}
		mockUsecase.On("SetUserRole", mock.Anything, "admin-id-123", domain.RoleAdmin, targetUserID, domain.RoleAdmin).
			Return(nil, domain.ErrPermissionDenied).Once()

		body, _ := json.Marshal(reqPayload)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/admin/users/"+targetUserID+"/role", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// ACT
		router.ServeHTTP(w, req)

		// ASSERT
		assert.Equal(t, http.StatusForbidden, w.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("Failure - Invalid JSON body", func(t *testing.T) {
		// ARRANGE: Create a fresh mock and router for THIS test.
		mockUsecase := new(MockUserUsecase)
		router := setupUserRouter(mockUsecase)
		// NOTE: We do NOT set any ".On()" expectations because the usecase should not be called.

		w := httptest.NewRecorder()
		// Send malformed JSON that will cause ShouldBindJSON to fail
		req, _ := http.NewRequest(http.MethodPatch, "/admin/users/"+targetUserID+"/role", bytes.NewBufferString(`{"newRole":`))
		req.Header.Set("Content-Type", "application/json")

		// ACT
		router.ServeHTTP(w, req)

		// ASSERT
		assert.Equal(t, http.StatusBadRequest, w.Code)
		// This assertion will now work correctly because this specific mock instance was never called.
		mockUsecase.AssertNotCalled(t, "SetUserRole", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}
