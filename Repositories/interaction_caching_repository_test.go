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

// MockInteractionRepository is a mock for domain.IInteractionRepository
type MockInteractionRepository struct {
	mock.Mock
}

func (m *MockInteractionRepository) Create(ctx context.Context, interaction *domain.BlogInteraction) error {
	args := m.Called(ctx, interaction)
	return args.Error(0)
}
func (m *MockInteractionRepository) Get(ctx context.Context, userID, blogID string) (*domain.BlogInteraction, error) {
	args := m.Called(ctx, userID, blogID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.BlogInteraction), args.Error(1)
}
func (m *MockInteractionRepository) GetByID(ctx context.Context, id string) (*domain.BlogInteraction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.BlogInteraction), args.Error(1)
}
func (m *MockInteractionRepository) Update(ctx context.Context, interaction *domain.BlogInteraction) error {
	args := m.Called(ctx, interaction)
	return args.Error(0)
}
func (m *MockInteractionRepository) Delete(ctx context.Context, interactionID string) error {
	args := m.Called(ctx, interactionID)
	return args.Error(0)
}

// --- The Test Suite ---

type CachingInteractionDecoratorSuite struct {
	suite.Suite
	mockRepo    *MockInteractionRepository
	mockCache   *MockCacheService
	cachingRepo domain.IInteractionRepository
}

func (s *CachingInteractionDecoratorSuite) SetupTest() {
	s.mockRepo = new(MockInteractionRepository)
	s.mockCache = new(MockCacheService)
	s.cachingRepo = NewCachingInteractionRepository(s.mockRepo, s.mockCache)
}

func TestCachingInteractionDecoratorSuite(t *testing.T) {
	suite.Run(t, new(CachingInteractionDecoratorSuite))
}

// --- The Actual Tests ---

func (s *CachingInteractionDecoratorSuite) TestGet_CacheMiss() {
	ctx := context.Background()
	userID, blogID := "user123", "blog456"
	cacheKey := "interaction:user:user123:blog:blog456"
	expectedInteraction := &domain.BlogInteraction{UserID: userID, BlogID: blogID, Action: domain.ActionTypeLike}
	interactionBytes, _ := json.Marshal(expectedInteraction)

	// Arrange: Define mock expectations for a cache miss.
	s.mockCache.On("Get", ctx, cacheKey).Return(nil, domain.ErrNotFound).Once()
	s.mockRepo.On("Get", ctx, userID, blogID).Return(expectedInteraction, nil).Once()
	s.mockCache.On("Set", ctx, cacheKey, interactionBytes, 15*time.Minute).Return(nil).Once()

	// Act
	result, err := s.cachingRepo.Get(ctx, userID, blogID)

	// Assert
	s.NoError(err)
	s.Equal(expectedInteraction, result)
	s.mockCache.AssertExpectations(s.T())
	s.mockRepo.AssertExpectations(s.T())
}

func (s *CachingInteractionDecoratorSuite) TestGet_CacheHit() {
	ctx := context.Background()
	userID, blogID := "user123", "blog456"
	cacheKey := "interaction:user:user123:blog:blog456"
	expectedInteraction := &domain.BlogInteraction{UserID: userID, BlogID: blogID, Action: domain.ActionTypeLike}
	interactionBytes, _ := json.Marshal(expectedInteraction)

	// Arrange: Define mock expectations for a cache hit.
	s.mockCache.On("Get", ctx, cacheKey).Return(interactionBytes, nil).Once()

	// Act
	result, err := s.cachingRepo.Get(ctx, userID, blogID)

	// Assert
	s.NoError(err)
	s.Equal(expectedInteraction.Action, result.Action)
	s.mockCache.AssertExpectations(s.T())
	// Assert that the database was NOT hit.
	s.mockRepo.AssertNotCalled(s.T(), "Get", mock.Anything, mock.Anything, mock.Anything)
}

func (s *CachingInteractionDecoratorSuite) TestUpdate_InvalidatesCache() {
	ctx := context.Background()
	interactionToUpdate := &domain.BlogInteraction{UserID: "user123", BlogID: "blog456", Action: domain.ActionTypeDislike}
	cacheKey := "interaction:user:user123:blog:blog456"

	// Arrange
	s.mockRepo.On("Update", ctx, interactionToUpdate).Return(nil).Once()
	s.mockCache.On("Delete", ctx, cacheKey).Return(nil).Once()

	// Act
	err := s.cachingRepo.Update(ctx, interactionToUpdate)

	// Assert
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingInteractionDecoratorSuite) TestDelete_InvalidatesCache() {
	ctx := context.Background()
	interactionID := "interaction789"
	interactionToDelete := &domain.BlogInteraction{ID: interactionID, UserID: "user123", BlogID: "blog456"}
	cacheKey := "interaction:user:user123:blog:blog456"

	// Arrange
	// 1. Expect a call to GetByID to find the userID and blogID for the cache key.
	s.mockRepo.On("GetByID", ctx, interactionID).Return(interactionToDelete, nil).Once()
	// 2. Expect the actual Delete call.
	s.mockRepo.On("Delete", ctx, interactionID).Return(nil).Once()
	// 3. Expect the cache invalidation call.
	s.mockCache.On("Delete", ctx, cacheKey).Return(nil).Once()

	// Act
	err := s.cachingRepo.Delete(ctx, interactionID)

	// Assert
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingInteractionDecoratorSuite) TestCreate_PassThrough() {
	ctx := context.Background()
	interactionToCreate := &domain.BlogInteraction{UserID: "user123", BlogID: "blog456"}

	// Arrange: Expect the call to be passed directly to the wrapped repo.
	s.mockRepo.On("Create", ctx, interactionToCreate).Return(nil).Once()

	// Act
	err := s.cachingRepo.Create(ctx, interactionToCreate)

	// Assert
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	// Assert that the cache was NOT touched for this operation, as per our strategy.
	s.mockCache.AssertNotCalled(s.T(), "Get")
	s.mockCache.AssertNotCalled(s.T(), "Set")
	s.mockCache.AssertNotCalled(s.T(), "Delete")
}
