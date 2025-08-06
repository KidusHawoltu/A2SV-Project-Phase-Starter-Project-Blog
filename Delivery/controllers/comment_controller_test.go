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
	domain "A2SV_Starter_Project_Blog/Domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock ICommentUsecase ---
type MockCommentUsecase struct {
	mock.Mock
}

func (m *MockCommentUsecase) CreateComment(ctx context.Context, userID, blogID, content string, parentID *string) (*domain.Comment, error) {
	args := m.Called(ctx, userID, blogID, content, parentID)
	var comment *domain.Comment
	if args.Get(0) != nil {
		comment = args.Get(0).(*domain.Comment)
	}
	return comment, args.Error(1)
}
func (m *MockCommentUsecase) UpdateComment(ctx context.Context, userID, commentID, content string) (*domain.Comment, error) {
	args := m.Called(ctx, userID, commentID, content)
	var comment *domain.Comment
	if args.Get(0) != nil {
		comment = args.Get(0).(*domain.Comment)
	}
	return comment, args.Error(1)
}
func (m *MockCommentUsecase) DeleteComment(ctx context.Context, userID, commentID string) error {
	args := m.Called(ctx, userID, commentID)
	return args.Error(0)
}
func (m *MockCommentUsecase) GetCommentsForBlog(ctx context.Context, blogID string, page, limit int64) ([]*domain.Comment, int64, error) {
	args := m.Called(ctx, blogID, page, limit)
	var comments []*domain.Comment
	if args.Get(0) != nil {
		comments = args.Get(0).([]*domain.Comment)
	}
	return comments, args.Get(1).(int64), args.Error(2)
}
func (m *MockCommentUsecase) GetRepliesForComment(ctx context.Context, parentID string, page, limit int64) ([]*domain.Comment, int64, error) {
	args := m.Called(ctx, parentID, page, limit)
	var comments []*domain.Comment
	if args.Get(0) != nil {
		comments = args.Get(0).([]*domain.Comment)
	}
	return comments, args.Get(1).(int64), args.Error(2)
}

// --- Test Suite Setup ---
type CommentControllerTestSuite struct {
	suite.Suite
}

func (s *CommentControllerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
}

func TestCommentControllerTestSuite(t *testing.T) {
	suite.Run(t, new(CommentControllerTestSuite))
}

// --- Tests ---

func (s *CommentControllerTestSuite) TestCreateComment() {
	authMiddleware := func(c *gin.Context) { c.Set("userID", "user-123"); c.Next() }

	s.Run("Success - Top Level Comment", func() {
		mockUsecase := new(MockCommentUsecase)
		controller := NewCommentController(mockUsecase)
		router := gin.New()
		router.POST("/blogs/:blogID/comments", authMiddleware, controller.CreateComment)

		blogID := "blog-abc"
		reqBody := CreateCommentRequest{Content: "Great post!"}
		mockReturnedComment := &domain.Comment{ID: "new-comment-id", Content: reqBody.Content}

		mockUsecase.On("CreateComment", mock.Anything, "user-123", blogID, reqBody.Content, (*string)(nil)).Return(mockReturnedComment, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/blogs/"+blogID+"/comments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		s.Equal(http.StatusCreated, w.Code)
		var resp CommentResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		s.Equal("new-comment-id", resp.ID)
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Success - Reply Comment", func() {
		// Note: This test simulates calling the same controller method but for a reply route.
		mockUsecase := new(MockCommentUsecase)
		controller := NewCommentController(mockUsecase)
		router := gin.New()
		// Your router might use a different path for replies, but it calls the same handler.
		router.POST("/comments/:commentID/replies", authMiddleware, controller.CreateComment)

		parentID := "parent-xyz"
		reqBody := CreateCommentRequest{Content: "Good point!", ParentID: &parentID}
		mockReturnedComment := &domain.Comment{ID: "new-reply-id", Content: reqBody.Content, ParentID: &parentID}

		// The CreateComment usecase is smart enough to handle the parentID, but the controller
		// doesn't know about the blogID from the URL in this case. In a real scenario, the
		// usecase would fetch the parent comment to find the blogID. We'll pass an empty string
		// for blogID to simulate this route's behavior.
		mockUsecase.On("CreateComment", mock.Anything, "user-123", "", reqBody.Content, &parentID).Return(mockReturnedComment, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/comments/"+parentID+"/replies", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		s.Equal(http.StatusCreated, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure - Invalid JSON body", func() {
		mockUsecase := new(MockCommentUsecase)
		controller := NewCommentController(mockUsecase)
		router := gin.New()
		router.POST("/blogs/:blogID/comments", authMiddleware, controller.CreateComment)

		// Body is missing the required 'content' field
		invalidBody := `{"parentId": "some-id"}`
		req := httptest.NewRequest(http.MethodPost, "/blogs/some-blog/comments", strings.NewReader(invalidBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		s.Equal(http.StatusBadRequest, w.Code)
		mockUsecase.AssertNotCalled(s.T(), "CreateComment")
	})
}

func (s *CommentControllerTestSuite) TestUpdateComment() {
	authMiddleware := func(c *gin.Context) { c.Set("userID", "user-123"); c.Next() }

	s.Run("Success", func() {
		mockUsecase := new(MockCommentUsecase)
		controller := NewCommentController(mockUsecase)
		router := gin.New()
		router.PUT("/comments/:commentID", authMiddleware, controller.UpdateComment)

		commentID := "comment-abc"
		reqBody := UpdateCommentRequest{Content: "Updated content."}
		mockReturnedComment := &domain.Comment{ID: commentID, Content: reqBody.Content}

		mockUsecase.On("UpdateComment", mock.Anything, "user-123", commentID, reqBody.Content).Return(mockReturnedComment, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/comments/"+commentID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		s.Equal(http.StatusOK, w.Code)
		var resp CommentResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		s.Equal(reqBody.Content, resp.Content)
		mockUsecase.AssertExpectations(s.T())
	})
}

func (s *CommentControllerTestSuite) TestGetCommentsForBlog() {
	// This is a public endpoint, no auth middleware needed.
	s.Run("Success", func() {
		// Arrange
		mockUsecase := new(MockCommentUsecase)
		controller := NewCommentController(mockUsecase)
		router := gin.New()
		router.GET("/blogs/:blogID/comments", controller.GetCommentsForBlog)

		blogID := "blog-abc"
		mockComments := []*domain.Comment{{ID: "c1"}, {ID: "c2"}}

		// Expect a call with default page=1, limit=10
		mockUsecase.On("GetCommentsForBlog", mock.Anything, blogID, int64(1), int64(10)).Return(mockComments, int64(2), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/blogs/"+blogID+"/comments", nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code)
		var resp PaginatedCommentResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		s.Len(resp.Data, 2)
		s.Equal(int64(2), resp.Pagination.Total)
		mockUsecase.AssertExpectations(s.T())
	})
}

func (s *CommentControllerTestSuite) TestDeleteComment() {
	authMiddleware := func(c *gin.Context) { c.Set("userID", "user-123"); c.Next() }

	s.Run("Success", func() {
		// Arrange
		mockUsecase := new(MockCommentUsecase)
		controller := NewCommentController(mockUsecase)
		router := gin.New()
		router.DELETE("/comments/:commentID", authMiddleware, controller.DeleteComment)

		commentID := "comment-to-delete"
		mockUsecase.On("DeleteComment", mock.Anything, "user-123", commentID).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/comments/"+commentID, nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusNoContent, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure - Permission Denied", func() {
		// Arrange
		mockUsecase := new(MockCommentUsecase)
		controller := NewCommentController(mockUsecase)
		router := gin.New()
		router.DELETE("/comments/:commentID", authMiddleware, controller.DeleteComment)

		commentID := "comment-owned-by-other"
		mockUsecase.On("DeleteComment", mock.Anything, "user-123", commentID).Return(domain.ErrPermissionDenied).Once()

		req := httptest.NewRequest(http.MethodDelete, "/comments/"+commentID, nil)
		w := httptest.NewRecorder()

		// Act
		router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusForbidden, w.Code)
		mockUsecase.AssertExpectations(s.T())
	})
}

func (s *CommentControllerTestSuite) TestGetRepliesForComment() {
	// Public endpoint
	s.Run("Success", func() {
		mockUsecase := new(MockCommentUsecase)
		controller := NewCommentController(mockUsecase)
		router := gin.New()
		router.GET("/comments/:commentID/replies", controller.GetRepliesForComment)

		parentID := "parent-abc"
		mockReplies := []*domain.Comment{{ID: "r1"}, {ID: "r2"}}
		
		// Test with specific pagination query params
		mockUsecase.On("GetRepliesForComment", mock.Anything, parentID, int64(2), int64(5)).Return(mockReplies, int64(2), nil).Once()
		
		req := httptest.NewRequest(http.MethodGet, "/comments/"+parentID+"/replies?page=2&limit=5", nil)
		w := httptest.NewRecorder()
		
		router.ServeHTTP(w, req)

		s.Equal(http.StatusOK, w.Code)
		var resp PaginatedCommentResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		s.Len(resp.Data, 2)
		mockUsecase.AssertExpectations(s.T())
	})
}
