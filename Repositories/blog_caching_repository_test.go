package repositories_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Repositories"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock Implementations ---

// MockBlogRepository is a mock for domain.IBlogRepository
type MockBlogRepository struct {
	mock.Mock
}

// Implement all methods of the domain.IBlogRepository interface
func (m *MockBlogRepository) Create(ctx context.Context, blog *domain.Blog) error {
	args := m.Called(ctx, blog)
	return args.Error(0)
}
func (m *MockBlogRepository) GetByID(ctx context.Context, id string) (*domain.Blog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Blog), args.Error(1)
}
func (m *MockBlogRepository) Update(ctx context.Context, blog *domain.Blog) error {
	args := m.Called(ctx, blog)
	return args.Error(0)
}
func (m *MockBlogRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockBlogRepository) SearchAndFilter(ctx context.Context, opts domain.BlogSearchFilterOptions) ([]*domain.Blog, int64, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Blog), args.Get(1).(int64), args.Error(2)
}
func (m *MockBlogRepository) IncrementLikes(ctx context.Context, blogID string, value int) error {
	args := m.Called(ctx, blogID, value)
	return args.Error(0)
}
func (m *MockBlogRepository) IncrementDislikes(ctx context.Context, blogID string, value int) error {
	args := m.Called(ctx, blogID, value)
	return args.Error(0)
}
func (m *MockBlogRepository) IncrementViews(ctx context.Context, blogID string) error {
	args := m.Called(ctx, blogID)
	return args.Error(0)
}
func (m *MockBlogRepository) IncrementCommentCount(ctx context.Context, blogID string, value int) error {
	args := m.Called(ctx, blogID, value)
	return args.Error(0)
}
func (m *MockBlogRepository) UpdateInteractionCounts(ctx context.Context, blogID string, likesInc, dislikesInc int) error {
	args := m.Called(ctx, blogID, likesInc, dislikesInc)
	return args.Error(0)
}

// --- The Test Suite ---

type CachingBlogDecoratorSuite struct {
	suite.Suite
	mockRepo    *MockBlogRepository
	mockCache   *MockCacheService
	cachingRepo domain.IBlogRepository // The decorator we are testing
}

func (s *CachingBlogDecoratorSuite) SetupTest() {
	s.mockRepo = new(MockBlogRepository)
	s.mockCache = new(MockCacheService)
	// Inject the mocks into the decorator
	s.cachingRepo = NewCachingBlogRepository(s.mockRepo, s.mockCache)
}

func TestCachingBlogDecoratorSuite(t *testing.T) {
	suite.Run(t, new(CachingBlogDecoratorSuite))
}

// --- The Actual Tests ---

func (s *CachingBlogDecoratorSuite) TestGetByID_CacheMiss() {
	ctx := context.Background()
	blogID := "blog123"
	cacheKey := "blog:id:blog123"
	expectedBlog := &domain.Blog{ID: blogID, Title: "A Great Post"}
	blogBytes, _ := json.Marshal(expectedBlog)

	// Arrange: Define mock expectations for a cache miss scenario.
	s.mockCache.On("Get", ctx, cacheKey).Return(nil, domain.ErrNotFound).Once()
	s.mockRepo.On("GetByID", ctx, blogID).Return(expectedBlog, nil).Once()
	s.mockCache.On("Set", ctx, cacheKey, blogBytes, 5*time.Minute).Return(nil).Once()

	// Act: Call the method on the decorator.
	resultBlog, err := s.cachingRepo.GetByID(ctx, blogID)

	// Assert: Check the result and verify all mock expectations were met.
	s.NoError(err)
	s.Equal(expectedBlog, resultBlog)
	s.mockCache.AssertExpectations(s.T())
	s.mockRepo.AssertExpectations(s.T())
}

func (s *CachingBlogDecoratorSuite) TestGetByID_CacheHit() {
	ctx := context.Background()
	blogID := "blog123"
	cacheKey := "blog:id:blog123"
	expectedBlog := &domain.Blog{ID: blogID, Title: "A Great Post"}
	blogBytes, _ := json.Marshal(expectedBlog)

	// Arrange: Define mock expectations for a cache hit.
	s.mockCache.On("Get", ctx, cacheKey).Return(blogBytes, nil).Once()

	// Act
	resultBlog, err := s.cachingRepo.GetByID(ctx, blogID)

	// Assert
	s.NoError(err)
	s.Equal(expectedBlog.Title, resultBlog.Title)
	s.mockCache.AssertExpectations(s.T())
	// Crucially, assert that the database was NOT called.
	s.mockRepo.AssertNotCalled(s.T(), "GetByID", mock.Anything, mock.Anything)
}

func (s *CachingBlogDecoratorSuite) TestUpdate_InvalidatesCache() {
	ctx := context.Background()
	blogToUpdate := &domain.Blog{ID: "blog123", Title: "An Updated Post"}
	cacheKey := "blog:id:blog123"

	// Arrange: Expect a call to the repo's Update and the cache's Delete.
	s.mockRepo.On("Update", ctx, blogToUpdate).Return(nil).Once()
	s.mockCache.On("Delete", ctx, cacheKey).Return(nil).Once()

	// Act
	err := s.cachingRepo.Update(ctx, blogToUpdate)

	// Assert
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingBlogDecoratorSuite) TestDelete_InvalidatesCache() {
	ctx := context.Background()
	blogID := "blog123"
	cacheKey := "blog:id:blog123"

	// Arrange: Expect a call to the repo's Delete and the cache's Delete.
	s.mockRepo.On("Delete", ctx, blogID).Return(nil).Once()
	s.mockCache.On("Delete", ctx, cacheKey).Return(nil).Once()

	// Act
	err := s.cachingRepo.Delete(ctx, blogID)

	// Assert
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingBlogDecoratorSuite) TestPassThrough_SearchAndFilter() {
	ctx := context.Background()
	opts := domain.BlogSearchFilterOptions{Page: 1, Limit: 10}
	expectedBlogs := []*domain.Blog{{ID: "blog1"}}
	expectedTotal := int64(1)

	// Arrange: Expect the call to be passed directly to the wrapped repo.
	s.mockRepo.On("SearchAndFilter", ctx, opts).Return(expectedBlogs, expectedTotal, nil).Once()

	// Act
	blogs, total, err := s.cachingRepo.SearchAndFilter(ctx, opts)

	// Assert
	s.NoError(err)
	s.Equal(expectedBlogs, blogs)
	s.Equal(expectedTotal, total)
	s.mockRepo.AssertExpectations(s.T())
	// Assert that the cache was NOT touched for this operation.
	s.mockCache.AssertNotCalled(s.T(), "Get")
	s.mockCache.AssertNotCalled(s.T(), "Set")
	s.mockCache.AssertNotCalled(s.T(), "Delete")
}
