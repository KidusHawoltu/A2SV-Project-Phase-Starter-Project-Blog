// Use the _test package suffix for black-box testing.
package usecases_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock Repositories ---

// MockBlogRepository is a mock implementation of the domain.IBlogRepository interface.
// It embeds testify's mock.Mock to track calls and set expectations.
type MockBlogRepository struct {
	mock.Mock
}

// Implement all methods of the interface by delegating to the mock framework.
func (m *MockBlogRepository) Create(ctx context.Context, blog *domain.Blog) error {
	args := m.Called(ctx, blog)
	// We can manipulate the blog object here if the real implementation does.
	// For example, setting the ID.
	if blog != nil {
		blog.ID = "mock-generated-id"
	}
	return args.Error(0)
}

func (m *MockBlogRepository) SearchAndFilter(ctx context.Context, options domain.BlogSearchFilterOptions) ([]*domain.Blog, int64, error) {
	args := m.Called(ctx, options)
	var blogs []*domain.Blog
	if args.Get(0) != nil {
		blogs = args.Get(0).([]*domain.Blog)
	}
	return blogs, args.Get(1).(int64), args.Error(2)
}

func (m *MockBlogRepository) GetByID(ctx context.Context, id string) (*domain.Blog, error) {
	args := m.Called(ctx, id)
	var blog *domain.Blog
	if args.Get(0) != nil {
		blog = args.Get(0).(*domain.Blog)
	}
	return blog, args.Error(1)
}

func (m *MockBlogRepository) Update(ctx context.Context, blog *domain.Blog) error {
	args := m.Called(ctx, blog)
	return args.Error(0)
}

func (m *MockBlogRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Test Suite Setup ---

type BlogUsecaseTestSuite struct {
	suite.Suite
	mockBlogRepo *MockBlogRepository
	mockUserRepo *MockUserRepository // Added mock for user repository
	usecase      domain.IBlogUsecase
}

// SetupTest runs before each test in the suite.
// It creates fresh instances to ensure test isolation.
func (s *BlogUsecaseTestSuite) SetupTest() {
	s.mockBlogRepo = new(MockBlogRepository)
	s.mockUserRepo = new(MockUserRepository) // Initialize the new mock

	// Use a short, fixed timeout for tests.
	// The constructor call is updated to include the user repository.
	s.usecase = usecases.NewBlogUsecase(s.mockBlogRepo, s.mockUserRepo, 2*time.Second)
}

// TestBlogUsecaseTestSuite is the entry point for running the suite.
func TestBlogUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(BlogUsecaseTestSuite))
}

// --- Tests ---

func (s *BlogUsecaseTestSuite) TestCreate() {
	authorID := "user-123"
	mockAuthor := &domain.User{ID: authorID} // Assuming domain.User has an ID field

	s.Run("Success", func() {
		// Arrange
		// 1. Mock the user repository to confirm the author exists.
		s.mockUserRepo.On("GetByID", mock.Anything, authorID).Return(mockAuthor, nil).Once()
		// 2. Mock the blog repository's Create method. It's configured to set the ID.
		s.mockBlogRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Blog")).Return(nil).Once()

		// Act
		blog, err := s.usecase.Create(context.Background(), "A Valid Title", "Valid Content", authorID, nil)

		// Assert
		s.NoError(err)
		s.NotNil(blog)
		s.Equal("mock-generated-id", blog.ID, "The ID should be set by the repository")
		s.mockUserRepo.AssertExpectations(s.T())
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_DomainValidation", func() {
		// No mock setup is needed because the usecase should fail before calling any repository.

		// Act
		blog, err := s.usecase.Create(context.Background(), "", "Content", authorID, nil)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrValidation)
		s.Nil(blog)
		s.mockUserRepo.AssertNotCalled(s.T(), "GetByID") // Verify user repo was NOT called.
		s.mockBlogRepo.AssertNotCalled(s.T(), "Create")  // Verify blog repo was NOT called.
	})

	s.Run("Failure_AuthorNotFound", func() {
		// Arrange: Mock user repo to return (nil, nil) indicating user does not exist.
		s.mockUserRepo.On("GetByID", mock.Anything, authorID).Return(nil, nil).Once()

		// Act
		blog, err := s.usecase.Create(context.Background(), "A Valid Title", "Valid Content", authorID, nil)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrUserNotFound)
		s.Nil(blog)
		s.mockUserRepo.AssertExpectations(s.T())
		s.mockBlogRepo.AssertNotCalled(s.T(), "Create") // Verify blog repo was NOT called.
	})

	s.Run("Failure_UserRepoError", func() {
		// Arrange: Mock user repo to return an unexpected error.
		expectedErr := errors.New("user db connection failed")
		s.mockUserRepo.On("GetByID", mock.Anything, authorID).Return(nil, expectedErr).Once()

		// Act
		blog, err := s.usecase.Create(context.Background(), "A Valid Title", "Valid Content", authorID, nil)

		// Assert
		s.Error(err)
		s.ErrorIs(err, expectedErr)
		s.Nil(blog)
		s.mockUserRepo.AssertExpectations(s.T())
		s.mockBlogRepo.AssertNotCalled(s.T(), "Create") // Verify blog repo was NOT called.
	})

	s.Run("Failure_BlogRepositoryError", func() {
		// Arrange
		// 1. Mock the user lookup to succeed first.
		s.mockUserRepo.On("GetByID", mock.Anything, authorID).Return(mockAuthor, nil).Once()
		// 2. Mock the blog repo to fail on creation.
		s.mockBlogRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Blog")).Return(errors.New("db error")).Once()

		// Act
		blog, err := s.usecase.Create(context.Background(), "A Valid Title", "Valid Content", authorID, nil)

		// Assert
		s.Error(err)
		s.Nil(blog)
		s.mockUserRepo.AssertExpectations(s.T())
		s.mockBlogRepo.AssertExpectations(s.T())
	})
}

func (s *BlogUsecaseTestSuite) TestGetByID() {
	s.Run("Success", func() {
		// Arrange
		mockBlog, _ := domain.NewBlog("Title", "Content", "author", nil)
		mockBlog.ID = "blog-1"
		s.mockBlogRepo.On("GetByID", mock.Anything, "blog-1").Return(mockBlog, nil).Once()

		// Act
		blog, err := s.usecase.GetByID(context.Background(), "blog-1")

		// Assert
		s.NoError(err)
		s.Equal(mockBlog, blog)
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_NotFound", func() {
		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, "not-found-id").Return(nil, usecases.ErrNotFound).Once()

		// Act
		blog, err := s.usecase.GetByID(context.Background(), "not-found-id")

		// Assert
		s.Error(err)
		s.ErrorIs(err, usecases.ErrNotFound)
		s.Nil(blog)
		s.mockBlogRepo.AssertExpectations(s.T())
	})
}

func (s *BlogUsecaseTestSuite) TestDelete() {
	mockBlog, _ := domain.NewBlog("Title", "Content", "owner-id", nil)
	mockBlog.ID = "blog-to-delete"

	s.Run("Success_AsOwner", func() {
		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		s.mockBlogRepo.On("Delete", mock.Anything, mockBlog.ID).Return(nil).Once()

		// Act
		err := s.usecase.Delete(context.Background(), mockBlog.ID, "owner-id", domain.RoleUser)

		// Assert
		s.NoError(err)
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Success_AsAdmin", func() {
		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		s.mockBlogRepo.On("Delete", mock.Anything, mockBlog.ID).Return(nil).Once()

		// Act
		// The admin's ID is different from the owner's, but their role grants permission.
		err := s.usecase.Delete(context.Background(), mockBlog.ID, "admin-id", domain.RoleAdmin)

		// Assert
		s.NoError(err)
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_PermissionDenied", func() {
		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		// We DO NOT mock the "Delete" call because it should never be reached.

		// Act
		err := s.usecase.Delete(context.Background(), mockBlog.ID, "not-the-owner-id", domain.RoleUser)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrPermissionDenied)
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_BlogNotFound", func() {
		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, "not-found-id").Return(nil, usecases.ErrNotFound).Once()

		// Act
		err := s.usecase.Delete(context.Background(), "not-found-id", "any-user", domain.RoleUser)

		// Assert
		s.Error(err)
		s.ErrorIs(err, usecases.ErrNotFound)
		s.mockBlogRepo.AssertExpectations(s.T())
	})
}

func (s *BlogUsecaseTestSuite) TestUpdate() {
	mockBlog, _ := domain.NewBlog("Old Title", "Old Content", "owner-id", nil)
	mockBlog.ID = "blog-to-update"
	updates := map[string]interface{}{"title": "New Valid Title"}

	s.Run("Success", func() {
		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		s.mockBlogRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Blog")).Return(nil).Once()

		// Act
		updatedBlog, err := s.usecase.Update(context.Background(), mockBlog.ID, "owner-id", domain.RoleUser, updates)

		// Assert
		s.NoError(err)
		s.NotNil(updatedBlog)
		s.Equal("New Valid Title", updatedBlog.Title) // Verify the title was updated.
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_PermissionDenied", func() {
		// Arrange
		s.mockBlogRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		// No mock for "Update" as it shouldn't be called.

		// Act
		updatedBlog, err := s.usecase.Update(context.Background(), mockBlog.ID, "not-owner-id", domain.RoleUser, updates)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrPermissionDenied)
		s.Nil(updatedBlog)
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_InvalidUpdateData", func() {
		// Arrange
		invalidUpdates := map[string]interface{}{"title": "  "} // Update to an invalid empty title
		s.mockBlogRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()

		// Act
		updatedBlog, err := s.usecase.Update(context.Background(), mockBlog.ID, "owner-id", domain.RoleUser, invalidUpdates)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrValidation)
		s.Nil(updatedBlog)
		s.mockBlogRepo.AssertExpectations(s.T())
	})
}

func (s *BlogUsecaseTestSuite) TestSearchAndFilter() {
	authorName := "John Doe"
	authorIDs := []string{"user-123", "user-456"}

	s.Run("Success_WithoutAuthorName", func() {
		// This tests the simple case where no author name is provided,
		// and the call is passed directly to the blog repository.
		// Arrange
		opts := domain.BlogSearchFilterOptions{Page: 1, Limit: 10}
		s.mockBlogRepo.On("SearchAndFilter", mock.Anything, opts).Return([]*domain.Blog{}, int64(0), nil).Once()

		// Act
		blogs, total, err := s.usecase.SearchAndFilter(context.Background(), opts)

		// Assert
		s.NoError(err)
		s.NotNil(blogs)
		s.Equal(int64(0), total)
		s.mockUserRepo.AssertNotCalled(s.T(), "FindUserIDsByName") // Crucially, userRepo is not called
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Success_WithAuthorName", func() {
		// This tests the main orchestration path: converting an author's name to IDs.
		// Arrange
		opts := domain.BlogSearchFilterOptions{
			AuthorName: &authorName,
			Page:       1,
			Limit:      10,
		}

		// Mock the user repo to successfully find IDs
		s.mockUserRepo.On("FindUserIDsByName", mock.Anything, authorName).Return(authorIDs, nil).Once()

		// The usecase should modify the options struct before passing it to the blog repo
		expectedRepoOpts := opts
		expectedRepoOpts.AuthorIDs = authorIDs
		s.mockBlogRepo.On("SearchAndFilter", mock.Anything, expectedRepoOpts).Return([]*domain.Blog{}, int64(0), nil).Once()

		// Act
		_, _, err := s.usecase.SearchAndFilter(context.Background(), opts)

		// Assert
		s.NoError(err)
		s.mockUserRepo.AssertExpectations(s.T())
		s.mockBlogRepo.AssertExpectations(s.T())
	})

	s.Run("Success_ShortCircuit_WhenAuthorNotFound_AND_GlobalLogicIsAND", func() {
		// This tests the critical optimization path for AND logic.
		// Arrange
		opts := domain.BlogSearchFilterOptions{
			AuthorName:  &authorName,
			GlobalLogic: domain.GlobalLogicAND, // Explicitly set AND logic
		}

		// Mock the user repo to find NO users
		s.mockUserRepo.On("FindUserIDsByName", mock.Anything, authorName).Return([]string{}, nil).Once()

		// Act
		blogs, total, err := s.usecase.SearchAndFilter(context.Background(), opts)

		// Assert
		s.NoError(err)
		s.Empty(blogs) // Should return an empty slice
		s.Equal(int64(0), total)
		s.mockUserRepo.AssertExpectations(s.T())
		// The blog repo should NEVER be called in this case.
		s.mockBlogRepo.AssertNotCalled(s.T(), "SearchAndFilter")
	})

	s.Run("Success_NoShortCircuit_WhenAuthorNotFound_AND_GlobalLogicIsOR", func() {
		// This tests the crucial edge case where we MUST continue with an OR query.
		// Arrange
		opts := domain.BlogSearchFilterOptions{
			AuthorName:  &authorName,
			GlobalLogic: domain.GlobalLogicOR, // Set OR logic
		}

		// Mock the user repo to find NO users
		s.mockUserRepo.On("FindUserIDsByName", mock.Anything, authorName).Return([]string{}, nil).Once()

		// The usecase should add an EMPTY slice of AuthorIDs to the options
		expectedRepoOpts := opts
		expectedRepoOpts.AuthorIDs = []string{}
		expectedRepoOpts.Page = 1
		expectedRepoOpts.Limit = 10
		s.mockBlogRepo.On("SearchAndFilter", mock.Anything, expectedRepoOpts).Return([]*domain.Blog{}, int64(0), nil).Once()

		// Act
		_, _, err := s.usecase.SearchAndFilter(context.Background(), opts)

		// Assert
		s.NoError(err)
		s.mockUserRepo.AssertExpectations(s.T())
		s.mockBlogRepo.AssertExpectations(s.T()) // The blog repo MUST be called
	})

	s.Run("Failure_WhenUserRepoFails", func() {
		// Arrange
		opts := domain.BlogSearchFilterOptions{AuthorName: &authorName}
		expectedErr := errors.New("user db down")
		s.mockUserRepo.On("FindUserIDsByName", mock.Anything, authorName).Return([]string{}, expectedErr).Once()

		// Act
		blogs, total, err := s.usecase.SearchAndFilter(context.Background(), opts)

		// Assert
		s.Error(err)
		s.ErrorIs(err, usecases.ErrInternal) // The usecase should wrap the specific error
		s.Nil(blogs)
		s.Equal(int64(0), total)
		s.mockUserRepo.AssertExpectations(s.T())
		s.mockBlogRepo.AssertNotCalled(s.T(), "SearchAndFilter")
	})
}
