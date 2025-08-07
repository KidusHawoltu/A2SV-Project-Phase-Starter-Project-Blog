package usecases_test

import (
	"context"
	"sync"
	"testing"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Usecases"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockCommentRepository struct {
	mock.Mock
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}
func (m *MockCommentRepository) GetByID(ctx context.Context, commentID string) (*domain.Comment, error) {
	args := m.Called(ctx, commentID)
	var comment *domain.Comment
	if args.Get(0) != nil {
		comment = args.Get(0).(*domain.Comment)
	}
	return comment, args.Error(1)
}
func (m *MockCommentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}
func (m *MockCommentRepository) Anonymize(ctx context.Context, commentID string) error {
	args := m.Called(ctx, commentID)
	return args.Error(0)
}
func (m *MockCommentRepository) FetchByBlogID(ctx context.Context, blogID string, page, limit int64) ([]*domain.Comment, int64, error) {
	args := m.Called(ctx, blogID, page, limit)
	return args.Get(0).([]*domain.Comment), args.Get(1).(int64), args.Error(2)
}
func (m *MockCommentRepository) FetchReplies(ctx context.Context, parentID string, page, limit int64) ([]*domain.Comment, int64, error) {
	args := m.Called(ctx, parentID, page, limit)
	return args.Get(0).([]*domain.Comment), args.Get(1).(int64), args.Error(2)
}
func (m *MockCommentRepository) IncrementReplyCount(ctx context.Context, parentID string, value int) error {
	args := m.Called(ctx, parentID, value)
	return args.Error(0)
}

// --- Test Suite Setup ---
type CommentUsecaseTestSuite struct {
	suite.Suite
	mockBlogRepo    *MockBlogRepository
	mockCommentRepo *MockCommentRepository
	usecase         domain.ICommentUsecase
}

func (s *CommentUsecaseTestSuite) SetupTest() {
	s.mockBlogRepo = new(MockBlogRepository)
	s.mockCommentRepo = new(MockCommentRepository)
	s.usecase = NewCommentUsecase(s.mockBlogRepo, s.mockCommentRepo, 2*time.Second)
}

func TestCommentUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(CommentUsecaseTestSuite))
}

func (s *CommentUsecaseTestSuite) TestCreateComment() {
	ctx := context.Background()
	userID := "user-123"
	blogID := "blog-abc"
	content := "A new comment"

	s.Run("Success - Top Level Comment", func() {
		s.SetupTest()
		var wg sync.WaitGroup
		wg.Add(1) // We expect one goroutine for the blog counter

		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, blogID).Return(&domain.Blog{}, nil).Once()
		s.mockCommentRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Comment")).Return(nil).Once()
		s.mockBlogRepo.On("IncrementCommentCount", mock.Anything, blogID, 1).
			Run(func(args mock.Arguments) { wg.Done() }).Return(nil).Once()

		// Act
		comment, err := s.usecase.CreateComment(ctx, userID, blogID, content, nil)

		// Assert
		s.NoError(err)
		s.NotNil(comment)
		wg.Wait() // Wait for the counter goroutine to finish
		s.mockBlogRepo.AssertExpectations(s.T())
		s.mockCommentRepo.AssertExpectations(s.T())
		s.mockCommentRepo.AssertNotCalled(s.T(), "IncrementReplyCount") // Ensure reply counter is not touched
	})

	s.Run("Success - Reply Comment", func() {
		s.SetupTest()
		var wg sync.WaitGroup
		wg.Add(2) // We expect two goroutines: blog counter and reply counter
		parentID := "parent-xyz"

		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, blogID).Return(&domain.Blog{}, nil).Once()
		s.mockCommentRepo.On("GetByID", mock.Anything, parentID).Return(&domain.Comment{}, nil).Once()
		s.mockCommentRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Comment")).Return(nil).Once()
		s.mockBlogRepo.On("IncrementCommentCount", mock.Anything, blogID, 1).
			Run(func(args mock.Arguments) { wg.Done() }).Return(nil).Once()
		s.mockCommentRepo.On("IncrementReplyCount", mock.Anything, parentID, 1).
			Run(func(args mock.Arguments) { wg.Done() }).Return(nil).Once()

		// Act
		comment, err := s.usecase.CreateComment(ctx, userID, blogID, content, &parentID)

		// Assert
		s.NoError(err)
		s.NotNil(comment)
		wg.Wait()
		s.mockBlogRepo.AssertExpectations(s.T())
		s.mockCommentRepo.AssertExpectations(s.T())
	})

	s.Run("Failure - Blog not found", func() {
		s.SetupTest()
		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, blogID).Return(nil, ErrNotFound).Once()

		// Act
		comment, err := s.usecase.CreateComment(ctx, userID, blogID, content, nil)

		// Assert
		s.Error(err)
		s.ErrorIs(err, ErrNotFound)
		s.Nil(comment)
		s.mockCommentRepo.AssertNotCalled(s.T(), "Create")
	})
}

func (s *CommentUsecaseTestSuite) TestUpdateComment() {
	ctx := context.Background()
	userID := "user-123"
	commentID := "comment-abc"
	newContent := "This content has been updated."

	s.Run("Success - As Owner", func() {
		s.SetupTest()
		// Arrange
		mockComment := &domain.Comment{ID: commentID, AuthorID: &userID, Content: "Original"}
		s.mockCommentRepo.On("GetByID", mock.Anything, commentID).Return(mockComment, nil).Once()
		s.mockCommentRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Comment")).
			// We can assert that the content was updated before the repo call
			Run(func(args mock.Arguments) {
				commentArg := args.Get(1).(*domain.Comment)
				s.Equal(newContent, commentArg.Content)
			}).
			Return(nil).Once()

		// Act
		updatedComment, err := s.usecase.UpdateComment(ctx, userID, commentID, newContent)

		// Assert
		s.NoError(err)
		s.NotNil(updatedComment)
		s.Equal(newContent, updatedComment.Content)
		s.mockCommentRepo.AssertExpectations(s.T())
	})

	s.Run("Failure - Not the owner", func() {
		s.SetupTest()
		// Arrange
		otherUserID := "user-456"
		mockComment := &domain.Comment{ID: commentID, AuthorID: &otherUserID, Content: "Original"}
		s.mockCommentRepo.On("GetByID", mock.Anything, commentID).Return(mockComment, nil).Once()

		// Act
		updatedComment, err := s.usecase.UpdateComment(ctx, userID, commentID, newContent)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrPermissionDenied)
		s.Nil(updatedComment)
		s.mockCommentRepo.AssertNotCalled(s.T(), "Update")
	})

	s.Run("Failure - Comment is anonymized (deleted)", func() {
		s.SetupTest()
		// Arrange: A comment with a nil AuthorID cannot be updated.
		mockComment := &domain.Comment{ID: commentID, AuthorID: nil, Content: "[deleted]"}
		s.mockCommentRepo.On("GetByID", mock.Anything, commentID).Return(mockComment, nil).Once()

		// Act
		_, err := s.usecase.UpdateComment(ctx, userID, commentID, newContent)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrPermissionDenied)
	})
}

func (s *CommentUsecaseTestSuite) TestDeleteComment() {
	ctx := context.Background()
	userID := "user-123"
	commentID := "comment-abc"
	blogID := "blog-xyz"

	s.Run("Success - As Owner", func() {
		s.SetupTest()
		var wg sync.WaitGroup
		wg.Add(1) // For the blog counter decrement

		// Arrange
		mockComment := &domain.Comment{ID: commentID, BlogID: blogID, AuthorID: &userID}
		s.mockCommentRepo.On("GetByID", mock.Anything, commentID).Return(mockComment, nil).Once()
		s.mockCommentRepo.On("Anonymize", mock.Anything, commentID).Return(nil).Once()
		s.mockBlogRepo.On("IncrementCommentCount", mock.Anything, blogID, -1).
			Run(func(args mock.Arguments) { wg.Done() }).Return(nil).Once()

		// Act
		err := s.usecase.DeleteComment(ctx, userID, commentID)

		// Assert
		s.NoError(err)
		wg.Wait()
		s.mockCommentRepo.AssertExpectations(s.T())
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Failure - Not the owner", func() {
		s.SetupTest()
		otherUserID := "user-456"
		mockComment := &domain.Comment{ID: commentID, BlogID: blogID, AuthorID: &otherUserID}
		s.mockCommentRepo.On("GetByID", mock.Anything, commentID).Return(mockComment, nil).Once()

		// Act
		err := s.usecase.DeleteComment(ctx, userID, commentID)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrPermissionDenied)
		s.mockCommentRepo.AssertNotCalled(s.T(), "Anonymize")
		s.mockBlogRepo.AssertNotCalled(s.T(), "IncrementCommentCount")
	})
}

func (s *CommentUsecaseTestSuite) TestGetCommentsForBlog() {
	ctx := context.Background()
	blogID := "blog-123"
	page, limit := int64(1), int64(10)

	s.Run("Success", func() {
		s.SetupTest()
		// Arrange
		mockComments := []*domain.Comment{{ID: "c1"}, {ID: "c2"}}
		mockTotal := int64(2)
		s.mockCommentRepo.On("FetchByBlogID", mock.Anything, blogID, page, limit).Return(mockComments, mockTotal, nil).Once()

		// Act
		comments, total, err := s.usecase.GetCommentsForBlog(ctx, blogID, page, limit)

		// Assert
		s.NoError(err)
		s.Equal(mockTotal, total)
		s.Equal(mockComments, comments)
		s.mockCommentRepo.AssertExpectations(s.T())
	})
}

func (s *CommentUsecaseTestSuite) TestGetRepliesForComment() {
	ctx := context.Background()
	parentID := "parent-comment-123"
	page, limit := int64(1), int64(10)

	s.Run("Success", func() {
		s.SetupTest()
		// Arrange
		mockReplies := []*domain.Comment{{ID: "r1"}, {ID: "r2"}}
		mockTotal := int64(2)
		s.mockCommentRepo.On("FetchReplies", mock.Anything, parentID, page, limit).Return(mockReplies, mockTotal, nil).Once()

		// Act
		comments, total, err := s.usecase.GetRepliesForComment(ctx, parentID, page, limit)

		// Assert
		s.NoError(err)
		s.Equal(mockTotal, total)
		s.Equal(mockReplies, comments)
		s.mockCommentRepo.AssertExpectations(s.T())
	})
}
