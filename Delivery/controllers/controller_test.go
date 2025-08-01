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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock Usecase ---

type MockBlogUsecase struct {
	mock.Mock
}

// Implement the IBlogUsecase interface
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
func (m *MockBlogUsecase) Update(ctx context.Context, blogID, userID, userRole string, updates map[string]interface{}) (*domain.Blog, error) {
	args := m.Called(ctx, blogID, userID, userRole, updates)
	var blog *domain.Blog
	if args.Get(0) != nil {
		blog = args.Get(0).(*domain.Blog)
	}
	return blog, args.Error(1)
}
func (m *MockBlogUsecase) Delete(ctx context.Context, blogID, userID, userRole string) error {
	args := m.Called(ctx, blogID, userID, userRole)
	return args.Error(0)
}

// --- Test Suite Setup ---

type BlogControllerTestSuite struct {
	suite.Suite
}

// SetupTest is now empty, as all setup is per-sub-test.
func (s *BlogControllerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
}

func TestBlogControllerTestSuite(t *testing.T) {
	suite.Run(t, new(BlogControllerTestSuite))
}

// --- Tests ---

func (s *BlogControllerTestSuite) TestCreate() {
	authMiddleware := func(c *gin.Context) { c.Set("userID", "user-123"); c.Next() }

	s.Run("Success", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.POST("/blogs", authMiddleware, controller.Create)

		mockBlog, _ := domain.NewBlog("Test Title", "Test Content", "user-123", nil)
		mockBlog.ID = "new-blog-id"
		mockUsecase.On("Create", mock.Anything, "Test Title", "Test Content", "user-123", mock.Anything).Return(mockBlog, nil).Once()

		reqBody := controllers.CreateBlogRequest{Title: "Test Title", Content: "Test Content"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/blogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusCreated, w.Code)
		var resp controllers.BlogResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		s.Equal("new-blog-id", resp.ID)
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure_GinBindingError", func() {
		// This test checks if the controller correctly handles a failure
		// from Gin's built-in validator (the `binding:"required"` tag).

		// Arrange
		mockUsecase := new(MockBlogUsecase) // Mock is created but not used
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.POST("/blogs", authMiddleware, controller.Create)

		reqBody := controllers.CreateBlogRequest{Title: "", Content: "Content"} // Invalid body
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/blogs", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusBadRequest, w.Code)
		// We assert that the usecase was NEVER called.
		mockUsecase.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func (s *BlogControllerTestSuite) TestGetByID() {
	s.Run("Success", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.GET("/blogs/:id", controller.GetByID)

		mockBlog, _ := domain.NewBlog("Found Title", "Found Content", "author-id", nil)
		mockBlog.ID = "found-id"
		mockUsecase.On("GetByID", mock.Anything, "found-id").Return(mockBlog, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/blogs/found-id", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure_NotFound", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.GET("/blogs/:id", controller.GetByID)

		mockUsecase.On("GetByID", mock.Anything, "not-found-id").Return(nil, usecases.ErrNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/blogs/not-found-id", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusNotFound, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})
}

func (s *BlogControllerTestSuite) TestDelete() {
	authMiddleware := func(c *gin.Context) { c.Set("userID", "user-123"); c.Set("role", domain.RoleUser); c.Next() }

	s.Run("Success", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.DELETE("/blogs/:id", authMiddleware, controller.Delete)

		mockUsecase.On("Delete", mock.Anything, "blog-to-delete", "user-123", "").Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/blogs/blog-to-delete", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusNoContent, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure_PermissionDenied", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.DELETE("/blogs/:id", authMiddleware, controller.Delete)

		mockUsecase.On("Delete", mock.Anything, "blog-to-delete", "user-123", "").Return(domain.ErrPermissionDenied).Once()

		req := httptest.NewRequest(http.MethodDelete, "/blogs/blog-to-delete", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusForbidden, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})
}
