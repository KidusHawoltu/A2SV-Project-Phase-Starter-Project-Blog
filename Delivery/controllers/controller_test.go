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

// --- Mock BlogUsecase ---
type MockBlogUsecase struct {
	mock.Mock
}

func (m *MockBlogUsecase) Create(ctx context.Context, title, content, authorID string, tags []string) (*domain.Blog, error) {
	args := m.Called(ctx, title, content, authorID, tags)
	var blog *domain.Blog
	if args.Get(0) != nil {
		blog = args.Get(0).(*domain.Blog)
	}
	return blog, args.Error(1)
}

func (m *MockBlogUsecase) Fetch(ctx context.Context, page, limit int64) ([]*domain.Blog, int64, error) {
	args := m.Called(ctx, page, limit)
	var blogs []*domain.Blog
	if args.Get(0) != nil {
		blogs = args.Get(0).([]*domain.Blog)
	}
	return blogs, args.Get(1).(int64), args.Error(2)
}

func (m *MockBlogUsecase) GetByID(ctx context.Context, id string) (*domain.Blog, error) {
	args := m.Called(ctx, id)
	var blog *domain.Blog
	if args.Get(0) != nil {
		blog = args.Get(0).(*domain.Blog)
	}
	return blog, args.Error(1)
}

func (m *MockBlogUsecase) Update(ctx context.Context, blogID, userID string, userRole domain.Role, updates map[string]interface{}) (*domain.Blog, error) {
	args := m.Called(ctx, blogID, userID, userRole, updates)
	var blog *domain.Blog
	if args.Get(0) != nil {
		blog = args.Get(0).(*domain.Blog)
	}
	return blog, args.Error(1)
}

func (m *MockBlogUsecase) Delete(ctx context.Context, blogID, userID string, userRole domain.Role) error {
	args := m.Called(ctx, blogID, userID, userRole)
	return args.Error(0)
}

// --- User Controller Test Suite ---
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
