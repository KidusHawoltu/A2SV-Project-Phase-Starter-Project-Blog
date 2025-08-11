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

// --- Test-specific Struct ---
// Define a struct to hold the cached result for paginated queries.
// This matchs the unexported one in the implementation file.
type paginatedCommentResult struct {
	Comments []*domain.Comment `json:"comments"`
	Total    int64             `json:"total"`
}

// --- Mock Implementations ---

// MockCommentRepository is a mock for domain.ICommentRepository
type MockCommentRepository struct {
	mock.Mock
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	args := m.Called(ctx, comment)
	return args.Error(0)
}
func (m *MockCommentRepository) GetByID(ctx context.Context, commentID string) (*domain.Comment, error) { /* ... */
	return nil, nil
}
func (m *MockCommentRepository) Update(ctx context.Context, comment *domain.Comment) error { /* ... */
	return nil
}
func (m *MockCommentRepository) Anonymize(ctx context.Context, commentID string) error { /* ... */
	return nil
}
func (m *MockCommentRepository) FetchByBlogID(ctx context.Context, blogID string, page, limit int64) ([]*domain.Comment, int64, error) {
	args := m.Called(ctx, blogID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Comment), args.Get(1).(int64), args.Error(2)
}
func (m *MockCommentRepository) FetchReplies(ctx context.Context, parentID string, page, limit int64) ([]*domain.Comment, int64, error) {
	args := m.Called(ctx, parentID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Comment), args.Get(1).(int64), args.Error(2)
}
func (m *MockCommentRepository) IncrementReplyCount(ctx context.Context, parentID string, value int) error { /* ... */
	return nil
}

// --- The Test Suite ---

type CachingCommentDecoratorSuite struct {
	suite.Suite
	mockRepo    *MockCommentRepository
	mockCache   *MockCacheService
	cachingRepo domain.ICommentRepository
}

func (s *CachingCommentDecoratorSuite) SetupTest() {
	s.mockRepo = new(MockCommentRepository)
	s.mockCache = new(MockCacheService)
	s.cachingRepo = NewCachingCommentRepository(s.mockRepo, s.mockCache)
}

func TestCachingCommentDecoratorSuite(t *testing.T) {
	suite.Run(t, new(CachingCommentDecoratorSuite))
}

// --- The Actual Tests ---

func (s *CachingCommentDecoratorSuite) TestFetchByBlogID_CacheMiss_CachesAnyPage() {
	ctx := context.Background()
	// Test with a page other than 1 to prove the logic works for all pages.
	blogID, page, limit := "blog123", int64(2), int64(20)
	cacheKey := "comments:blog:blog123:page:2:limit:20"
	trackerKey := "tracker:comments:blog:blog123"

	expectedResult := paginatedCommentResult{
		Comments: []*domain.Comment{{ID: "commentOnPage2", BlogID: blogID}},
		Total:    21,
	}
	resultBytes, _ := json.Marshal(expectedResult)

	// --- Arrange: Mock expectations for a cache miss ---
	s.mockCache.On("Get", ctx, cacheKey).Return(nil, domain.ErrNotFound).Once()
	s.mockRepo.On("FetchByBlogID", ctx, blogID, page, limit).Return(expectedResult.Comments, expectedResult.Total, nil).Once()
	s.mockCache.On("Set", ctx, cacheKey, resultBytes, 2*time.Minute).Return(nil).Once()
	s.mockCache.On("AddToSet", ctx, trackerKey, []interface{}{cacheKey}).Return(nil).Once()

	// --- Act ---
	comments, total, err := s.cachingRepo.FetchByBlogID(ctx, blogID, page, limit)

	// --- Assert ---
	s.NoError(err)
	s.Equal(expectedResult.Comments, comments)
	s.Equal(expectedResult.Total, total)
	s.mockCache.AssertExpectations(s.T())
	s.mockRepo.AssertExpectations(s.T())
}

func (s *CachingCommentDecoratorSuite) TestFetchReplies_CacheHit() {
	ctx := context.Background()
	parentID, page, limit := "parent123", int64(1), int64(10)
	cacheKey := "comments:replies:parent123:page:1:limit:10"

	expectedResult := paginatedCommentResult{
		Comments: []*domain.Comment{{ID: "reply1"}},
		Total:    1,
	}
	resultBytes, _ := json.Marshal(expectedResult)

	// --- Arrange: Mock expectations for a cache hit ---
	s.mockCache.On("Get", ctx, cacheKey).Return(resultBytes, nil).Once()

	// --- Act ---
	comments, total, err := s.cachingRepo.FetchReplies(ctx, parentID, page, limit)

	// --- Assert ---
	s.NoError(err)
	s.Equal(expectedResult.Comments[0].ID, comments[0].ID)
	s.Equal(expectedResult.Total, total)
	s.mockCache.AssertExpectations(s.T())
	// Assert that the database was NOT hit.
	s.mockRepo.AssertNotCalled(s.T(), "FetchReplies", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *CachingCommentDecoratorSuite) TestCreate_TopLevelComment_InvalidatesAllCachedLists() {
	ctx := context.Background()
	newComment := &domain.Comment{ID: "newComment", BlogID: "blog123", ParentID: nil}
	trackerKey := "tracker:comments:blog:blog123"
	// Simulate that multiple pages/limits were cached for this blog.
	keysToInvalidate := []string{
		"comments:blog:blog123:page:1:limit:10",
		"comments:blog:blog123:page:2:limit:10",
		"comments:blog:blog123:page:1:limit:50",
	}

	// --- Arrange ---
	s.mockRepo.On("Create", ctx, newComment).Return(nil).Once()
	// Expect the decorator to ask the cache for all members of the tracker set.
	s.mockCache.On("GetSetMembers", ctx, trackerKey).Return(keysToInvalidate, nil).Once()
	// Expect a call to delete all those keys PLUS the tracker key itself.
	expectedKeysToDelete := append(keysToInvalidate, trackerKey)
	s.mockCache.On("DeleteKeys", ctx, expectedKeysToDelete).Return(nil).Once()

	// --- Act ---
	err := s.cachingRepo.Create(ctx, newComment)

	// --- Assert ---
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingCommentDecoratorSuite) TestCreate_ReplyComment_InvalidatesCache() {
	ctx := context.Background()
	parentID := "parent123"
	newComment := &domain.Comment{ID: "newReply", BlogID: "blog123", ParentID: &parentID}
	trackerKey := "tracker:comments:replies:parent123"
	keysToInvalidate := []string{"comments:replies:parent123:page:1:limit:10"}

	// --- Arrange ---
	s.mockRepo.On("Create", ctx, newComment).Return(nil).Once()
	s.mockCache.On("GetSetMembers", ctx, trackerKey).Return(keysToInvalidate, nil).Once()
	expectedKeysToDelete := append(keysToInvalidate, trackerKey)
	s.mockCache.On("DeleteKeys", ctx, expectedKeysToDelete).Return(nil).Once()

	// --- Act ---
	err := s.cachingRepo.Create(ctx, newComment)

	// --- Assert ---
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingCommentDecoratorSuite) TestCreate_Invalidation_NoCachedKeys() {
	ctx := context.Background()
	newComment := &domain.Comment{ID: "newComment", BlogID: "blog123", ParentID: nil}
	trackerKey := "tracker:comments:blog:blog123"
	// Simulate the case where there are no cached items for this blog yet.
	var emptyKeyList []string

	// --- Arrange ---
	s.mockRepo.On("Create", ctx, newComment).Return(nil).Once()
	// Expect a call to GetSetMembers, which returns an empty list.
	s.mockCache.On("GetSetMembers", ctx, trackerKey).Return(emptyKeyList, nil).Once()

	// --- Act ---
	err := s.cachingRepo.Create(ctx, newComment)

	// --- Assert ---
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
	// Assert that DeleteKeys was NOT called, because there was nothing to delete.
	s.mockCache.AssertNotCalled(s.T(), "DeleteKeys", mock.Anything, mock.Anything)
}
