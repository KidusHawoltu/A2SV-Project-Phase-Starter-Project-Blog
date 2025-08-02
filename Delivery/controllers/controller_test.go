package controllers_test

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	domain "A2SV_Starter_Project_Blog/Domain"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"context"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock UserUsecase ---
type MockUserUsecase struct {
	mock.Mock
}

func (m *MockUserUsecase) Register(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserUsecase) Login(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

// --- Test Suite ---
type UserControllerTestSuite struct {
	suite.Suite
}

func (s *UserControllerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
}

func TestUserControllerTestSuite(t *testing.T) {
	suite.Run(t, new(UserControllerTestSuite))
}

func (s *UserControllerTestSuite) TestRegister() {
	s.Run("Success", func() {
		mockUC := new(MockUserUsecase)
		controller := controllers.NewUserController(mockUC)
		router := gin.New()
		router.POST("/register", controller.Register)

		reqBody := controllers.RegisterRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "securepass",
		}
		user := &domain.User{
			Username: reqBody.Username,
			Email:    reqBody.Email,
			Password: reqBody.Password,
			Role:     domain.RoleUser,
		}
		mockUC.On("Register", mock.Anything, user).Return(nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		s.Equal(http.StatusCreated, w.Code)
		mockUC.AssertExpectations(s.T())
	})
}

func (s *UserControllerTestSuite) TestLogin() {
	s.Run("Success", func() {
		mockUC := new(MockUserUsecase)
		controller := controllers.NewUserController(mockUC)
		router := gin.New()
		router.POST("/login", controller.Login)

		reqBody := controllers.LoginRequest{
			Email:    "test@example.com",
			Password: "securepass",
		}
		mockUC.On("Login", mock.Anything, reqBody.Email, reqBody.Password).Return("mock-token", nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.Contains(w.Body.String(), "mock-token")
		mockUC.AssertExpectations(s.T())
	})
}

func (s *UserControllerTestSuite) TestGetProfile() {
	s.Run("Success", func() {
		mockUC := new(MockUserUsecase)
		controller := controllers.NewUserController(mockUC)
		router := gin.New()
		group := router.Group("/profile")
		group.Use(func(c *gin.Context) {
			c.Set("userID", "user-456")
			c.Next()
		})
		group.GET("", controller.GetProfile)

		req := httptest.NewRequest(http.MethodGet, "/profile", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		s.Equal(http.StatusOK, w.Code)
		s.Contains(w.Body.String(), "user-456")
	})
}
