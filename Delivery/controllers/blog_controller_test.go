package controllers_test

import (
	"A2SV_Starter_Project_Blog/Delivery/controllers"
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"context"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

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

func (m *MockBlogUsecase) SearchAndFilter(ctx context.Context, options domain.BlogSearchFilterOptions) ([]*domain.Blog, int64, error) {
	args := m.Called(ctx, options)
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

func (m *MockBlogUsecase) InteractWithBlog(ctx context.Context, blogID, userID string, action domain.ActionType) error {
	args := m.Called(ctx, blogID, userID, action)
	return args.Error(0)
}

// --- Blog ControllerTest Suite Setup ---

type BlogControllerTestSuite struct {
	suite.Suite
}

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
	authMiddleware := func(c *gin.Context) { c.Set("userID", "user-123"); c.Set("role", string(domain.RoleUser)); c.Next() }

	s.Run("Success", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.DELETE("/blogs/:id", authMiddleware, controller.Delete)

		mockUsecase.On("Delete", mock.Anything, "blog-to-delete", "user-123", domain.RoleUser).Return(nil).Once()

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

		mockUsecase.On("Delete", mock.Anything, "blog-to-delete", "user-123", domain.RoleUser).Return(domain.ErrPermissionDenied).Once()

		req := httptest.NewRequest(http.MethodDelete, "/blogs/blog-to-delete", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusForbidden, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})
}

func (s *BlogControllerTestSuite) TestUpdate() {
	// Middleware to simulate an authenticated user making the request
	authMiddleware := func(c *gin.Context) {
		c.Set("userID", "user-123")
		c.Set("role", string(domain.RoleUser))
		c.Next()
	}

	s.Run("Success", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.PUT("/blogs/:id", authMiddleware, controller.Update)

		// The data we expect to send in the request body
		updatePayload := map[string]interface{}{"title": "Updated Title"}

		// The blog object we expect the usecase to return
		mockUpdatedBlog, _ := domain.NewBlog("Updated Title", "Original Content", "user-123", nil)
		mockUpdatedBlog.ID = "blog-to-update"

		// Set the mock expectation
		mockUsecase.On("Update", mock.Anything, "blog-to-update", "user-123", domain.RoleUser, updatePayload).Return(mockUpdatedBlog, nil).Once()

		// Create the HTTP request
		body, _ := json.Marshal(updatePayload)
		req := httptest.NewRequest(http.MethodPut, "/blogs/blog-to-update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code, "Expected status OK")
		var resp controllers.BlogResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		s.Equal("Updated Title", resp.Title, "Response title should be updated")
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure_UsecaseError", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.PUT("/blogs/:id", authMiddleware, controller.Update)

		updatePayload := map[string]interface{}{"title": "Updated Title"}
		mockUsecase.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrPermissionDenied).Once()

		body, _ := json.Marshal(updatePayload)
		req := httptest.NewRequest(http.MethodPut, "/blogs/some-id", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusForbidden, w.Code, "Expected status Forbidden for permission denied")
		mockUsecase.AssertExpectations(s.T())
	})
}

func (s *BlogControllerTestSuite) TestSearchAndFilter() {
	s.Run("Success_DefaultOptions", func() {
		// This test verifies that a simple request with no parameters
		// calls the usecase with the correct default options.
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.GET("/blogs", controller.SearchAndFilter)

		// Define the exact options struct we expect the controller to build
		expectedOptions := domain.BlogSearchFilterOptions{
			Page:        1,
			Limit:       10,
			GlobalLogic: domain.GlobalLogicAND, // Default
			TagLogic:    domain.GlobalLogicOR,  // Default
			SortOrder:   domain.SortOrderDESC,  // Default
		}

		// The mock data the usecase will return
		mockBlogs := []*domain.Blog{{ID: "1", Title: "Blog 1"}}
		totalCount := int64(1)

		// Set the mock expectation: The usecase must be called with our expectedOptions
		mockUsecase.On("SearchAndFilter", mock.Anything, expectedOptions).Return(mockBlogs, totalCount, nil).Once()

		// Act
		req := httptest.NewRequest(http.MethodGet, "/blogs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code)
		var resp controllers.PaginatedBlogResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		s.Len(resp.Data, 1)
		s.Equal(totalCount, resp.Pagination.Total)
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Success_WithAllQueryParameters", func() {
		// This test verifies that the controller correctly parses all possible
		// query parameters and constructs the options struct.
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.GET("/blogs", controller.SearchAndFilter)

		// Prepare pointer values for our expected struct
		title := "Test"
		authorName := "John"
		startDate, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")

		// Define the exact, fully-populated options struct we expect
		expectedOptions := domain.BlogSearchFilterOptions{
			Page:        2,
			Limit:       20,
			Title:       &title,
			AuthorName:  &authorName,
			Tags:        []string{"go", "api"},
			TagLogic:    domain.GlobalLogicAND,
			GlobalLogic: domain.GlobalLogicOR,
			StartDate:   &startDate,
			SortBy:      "title",
			SortOrder:   domain.SortOrderASC,
		}

		mockUsecase.On("SearchAndFilter", mock.Anything, expectedOptions).Return([]*domain.Blog{}, int64(0), nil).Once()

		// Create a URL with all the corresponding query parameters
		url := "/blogs?page=2&limit=20&title=Test&authorName=John&tags=go,api&tagLogic=AND&logic=OR&startDate=2023-01-01T00:00:00Z&sortBy=title&sortOrder=ASC"

		// Act
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code)
		mockUsecase.AssertExpectations(s.T()) // This is the most important assertion
	})

	s.Run("Failure_InvalidDateParameter", func() {
		// This test ensures that if a parameter is badly formatted,
		// the controller fails early and doesn't call the usecase.
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.GET("/blogs", controller.SearchAndFilter)

		// Create a request with a non-RFC3339 date
		req := httptest.NewRequest(http.MethodGet, "/blogs?startDate=2023-01-01", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusBadRequest, w.Code)
		// Usecase should NOT have been called
		mockUsecase.AssertNotCalled(s.T(), "SearchAndFilter", mock.Anything, mock.Anything)
	})
}

func (s *BlogControllerTestSuite) TestInteractWithBlog() {
	// Middleware to simulate an authenticated user.
	authMiddleware := func(c *gin.Context) {
		c.Set("userID", "user-123")
		c.Next()
	}

	s.Run("Success - Like action", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.POST("/blogs/:id/interact", authMiddleware, controller.InteractWithBlog)

		blogID := "blog-abc"
		action := domain.ActionTypeLike

		// Expect the usecase to be called with the correct parameters.
		mockUsecase.On("InteractWithBlog", mock.Anything, blogID, "user-123", action).Return(nil).Once()

		// Create the request body.
		reqBody := controllers.InteractBlogRequest{Action: action}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/blogs/"+blogID+"/interact", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure - Invalid action in body", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.POST("/blogs/:id/interact", authMiddleware, controller.InteractWithBlog)

		// Create a request with an invalid action string.
		invalidBody := `{"action": "invalid-action"}`
		req := httptest.NewRequest(http.MethodPost, "/blogs/some-id/interact", strings.NewReader(invalidBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusBadRequest, w.Code, "Expected Bad Request due to validation failure")
		// The usecase should NOT have been called.
		mockUsecase.AssertNotCalled(s.T(), "InteractWithBlog", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	s.Run("Failure - Usecase returns an error", func() {
		// Arrange
		mockUsecase := new(MockBlogUsecase)
		controller := controllers.NewBlogController(mockUsecase)
		router := gin.New()
		router.POST("/blogs/:id/interact", authMiddleware, controller.InteractWithBlog)

		blogID := "not-found-id"
		action := domain.ActionTypeLike

		// Expect the usecase to be called and to return an error.
		mockUsecase.On("InteractWithBlog", mock.Anything, blogID, "user-123", action).Return(usecases.ErrNotFound).Once()

		reqBody := controllers.InteractBlogRequest{Action: action}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/blogs/"+blogID+"/interact", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusNotFound, w.Code, "Expected Not Found status from HandleError")
		mockUsecase.AssertExpectations(s.T())
	})
}