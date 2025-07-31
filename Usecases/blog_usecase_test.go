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

// --- Mock Repository ---

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

func (m *MockBlogRepository) Fetch(ctx context.Context, page, limit int64) ([]*domain.Blog, int64, error) {
	args := m.Called(ctx, page, limit)
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
	mockRepo *MockBlogRepository
	usecase  domain.IBlogUsecase
}

// SetupTest runs before each test in the suite.
// It creates fresh instances to ensure test isolation.
func (s *BlogUsecaseTestSuite) SetupTest() {
	s.mockRepo = new(MockBlogRepository)
	// Use a short, fixed timeout for tests.
	s.usecase = usecases.NewBlogUsecase(s.mockRepo, 2*time.Second)
}

// TestBlogUsecaseTestSuite is the entry point for running the suite.
func TestBlogUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(BlogUsecaseTestSuite))
}

// --- Tests ---

func (s *BlogUsecaseTestSuite) TestCreate() {
	s.Run("Success", func() {
		// Arrange
		// The mock repo's `Create` method is configured to set the ID.
		s.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Blog")).Return(nil).Once()

		// Act
		blog, err := s.usecase.Create(context.Background(), "A Valid Title", "Valid Content", "user-123", nil)

		// Assert
		s.NoError(err)
		s.NotNil(blog)
		s.Equal("mock-generated-id", blog.ID, "The ID should be set by the repository")
		s.mockRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_DomainValidation", func() {
		// No mock setup is needed because the usecase should fail before calling the repo.

		// Act
		blog, err := s.usecase.Create(context.Background(), "", "Content", "user-123", nil)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrValidation)
		s.Nil(blog)
		s.mockRepo.AssertNotCalled(s.T(), "Create") // Verify repo was NOT called.
	})

	s.Run("Failure_RepositoryError", func() {
		// Arrange
		s.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Blog")).Return(errors.New("db error")).Once()

		// Act
		blog, err := s.usecase.Create(context.Background(), "A Valid Title", "Valid Content", "user-123", nil)

		// Assert
		s.Error(err)
		s.Nil(blog)
		s.mockRepo.AssertExpectations(s.T())
	})
}

func (s *BlogUsecaseTestSuite) TestGetByID() {
	s.Run("Success", func() {
		// Arrange
		mockBlog, _ := domain.NewBlog("Title", "Content", "author", nil)
		mockBlog.ID = "blog-1"
		s.mockRepo.On("GetByID", mock.Anything, "blog-1").Return(mockBlog, nil).Once()

		// Act
		blog, err := s.usecase.GetByID(context.Background(), "blog-1")

		// Assert
		s.NoError(err)
		s.Equal(mockBlog, blog)
		s.mockRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_NotFound", func() {
		// Arrange
		s.mockRepo.On("GetByID", mock.Anything, "not-found-id").Return(nil, usecases.ErrNotFound).Once()

		// Act
		blog, err := s.usecase.GetByID(context.Background(), "not-found-id")

		// Assert
		s.Error(err)
		s.ErrorIs(err, usecases.ErrNotFound)
		s.Nil(blog)
		s.mockRepo.AssertExpectations(s.T())
	})
}

func (s *BlogUsecaseTestSuite) TestDelete() {
	mockBlog, _ := domain.NewBlog("Title", "Content", "owner-id", nil)
	mockBlog.ID = "blog-to-delete"

	s.Run("Success_AsOwner", func() {
		// Arrange
		s.mockRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		s.mockRepo.On("Delete", mock.Anything, mockBlog.ID).Return(nil).Once()

		// Act
		err := s.usecase.Delete(context.Background(), mockBlog.ID, "owner-id", domain.RoleUser)

		// Assert
		s.NoError(err)
		s.mockRepo.AssertExpectations(s.T())
	})

	s.Run("Success_AsAdmin", func() {
		// Arrange
		s.mockRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		s.mockRepo.On("Delete", mock.Anything, mockBlog.ID).Return(nil).Once()

		// Act
		// The admin's ID is different from the owner's, but their role grants permission.
		err := s.usecase.Delete(context.Background(), mockBlog.ID, "admin-id", domain.RoleAdmin)

		// Assert
		s.NoError(err)
		s.mockRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_PermissionDenied", func() {
		// Arrange
		s.mockRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		// We DO NOT mock the "Delete" call because it should never be reached.

		// Act
		err := s.usecase.Delete(context.Background(), mockBlog.ID, "not-the-owner-id", domain.RoleUser)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrPermissionDenied)
		s.mockRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_BlogNotFound", func() {
		// Arrange
		s.mockRepo.On("GetByID", mock.Anything, "not-found-id").Return(nil, usecases.ErrNotFound).Once()

		// Act
		err := s.usecase.Delete(context.Background(), "not-found-id", "any-user", domain.RoleUser)

		// Assert
		s.Error(err)
		s.ErrorIs(err, usecases.ErrNotFound)
		s.mockRepo.AssertExpectations(s.T())
	})
}

func (s *BlogUsecaseTestSuite) TestUpdate() {
	mockBlog, _ := domain.NewBlog("Old Title", "Old Content", "owner-id", nil)
	mockBlog.ID = "blog-to-update"
	updates := map[string]interface{}{"title": "New Valid Title"}

	s.Run("Success", func() {
		// Arrange
		s.mockRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		s.mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Blog")).Return(nil).Once()

		// Act
		updatedBlog, err := s.usecase.Update(context.Background(), mockBlog.ID, "owner-id", domain.RoleUser, updates)

		// Assert
		s.NoError(err)
		s.NotNil(updatedBlog)
		s.Equal("New Valid Title", updatedBlog.Title) // Verify the title was updated.
		s.mockRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_PermissionDenied", func() {
		// Arrange
		s.mockRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()
		// No mock for "Update" as it shouldn't be called.

		// Act
		updatedBlog, err := s.usecase.Update(context.Background(), mockBlog.ID, "not-owner-id", domain.RoleUser, updates)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrPermissionDenied)
		s.Nil(updatedBlog)
		s.mockRepo.AssertExpectations(s.T())
	})

	s.Run("Failure_InvalidUpdateData", func() {
		// Arrange
		invalidUpdates := map[string]interface{}{"title": "  "} // Update to an invalid empty title
		s.mockRepo.On("GetByID", mock.Anything, mockBlog.ID).Return(mockBlog, nil).Once()

		// Act
		updatedBlog, err := s.usecase.Update(context.Background(), mockBlog.ID, "owner-id", domain.RoleUser, invalidUpdates)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrValidation)
		s.Nil(updatedBlog)
		s.mockRepo.AssertExpectations(s.T())
	})
}
