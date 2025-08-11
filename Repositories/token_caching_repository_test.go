package repositories_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockTokenRepository is a mock for usecases.TokenRepository
type MockTokenRepository struct {
	mock.Mock
}

// Implement all methods of the usecases.TokenRepository interface
func (m *MockTokenRepository) Store(ctx context.Context, token *domain.Token) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}
func (m *MockTokenRepository) GetByValue(ctx context.Context, tokenValue string) (*domain.Token, error) {
	args := m.Called(ctx, tokenValue)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Token), args.Error(1)
}
func (m *MockTokenRepository) GetByID(ctx context.Context, tokenID string) (*domain.Token, error) {
	args := m.Called(ctx, tokenID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Token), args.Error(1)
}
func (m *MockTokenRepository) Delete(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}
func (m *MockTokenRepository) DeleteByUserID(ctx context.Context, userID string, tokenType domain.TokenType) error {
	args := m.Called(ctx, userID, tokenType)
	return args.Error(0)
}

// --- The Test Suite ---

type CachingTokenDecoratorSuite struct {
	suite.Suite
	mockRepo    *MockTokenRepository
	mockCache   *MockCacheService
	cachingRepo usecases.TokenRepository
}

func (s *CachingTokenDecoratorSuite) SetupTest() {
	s.mockRepo = new(MockTokenRepository)
	s.mockCache = new(MockCacheService)
	// Inject the mocks into the decorator
	s.cachingRepo = NewCachingTokenRepository(s.mockRepo, s.mockCache)
}

func TestCachingTokenDecoratorSuite(t *testing.T) {
	suite.Run(t, new(CachingTokenDecoratorSuite))
}

// --- The Actual Tests ---

func (s *CachingTokenDecoratorSuite) TestGetByValue_CacheMiss() {
	ctx := context.Background()
	tokenValue := "some-refresh-token"
	cacheKey := "token:value:some-refresh-token"
	expectedToken := &domain.Token{Value: tokenValue, ExpiresAt: time.Now().Add(1 * time.Hour)}
	tokenBytes, _ := json.Marshal(expectedToken)

	// Arrange: Define mock expectations for a cache miss.
	s.mockCache.On("Get", ctx, cacheKey).Return(nil, domain.ErrNotFound).Once()
	s.mockRepo.On("GetByValue", ctx, tokenValue).Return(expectedToken, nil).Once()
	// Check that Set is called with a TTL that is close to 1 hour.
	s.mockCache.On("Set", ctx, cacheKey, tokenBytes, mock.AnythingOfType("time.Duration")).Run(func(args mock.Arguments) {
		// Assert that the TTL is within a reasonable range of the expected value.
		duration := args.Get(3).(time.Duration)
		s.InDelta(1*time.Hour, duration, float64(time.Second))
	}).Return(nil).Once()

	// Act
	resultToken, err := s.cachingRepo.GetByValue(ctx, tokenValue)

	// Assert
	s.NoError(err)
	s.Equal(expectedToken, resultToken)
	s.mockCache.AssertExpectations(s.T())
	s.mockRepo.AssertExpectations(s.T())
}

func (s *CachingTokenDecoratorSuite) TestGetByValue_CacheHit() {
	ctx := context.Background()
	tokenValue := "some-refresh-token"
	cacheKey := "token:value:some-refresh-token"
	expectedToken := &domain.Token{Value: tokenValue, ExpiresAt: time.Now().Add(1 * time.Hour)}
	tokenBytes, _ := json.Marshal(expectedToken)

	// Arrange: Define mock expectations for a cache hit.
	s.mockCache.On("Get", ctx, cacheKey).Return(tokenBytes, nil).Once()

	// Act
	resultToken, err := s.cachingRepo.GetByValue(ctx, tokenValue)

	// Assert
	s.NoError(err)
	s.Equal(expectedToken.Value, resultToken.Value)
	s.mockCache.AssertExpectations(s.T())
	// Assert that the database was NOT called.
	s.mockRepo.AssertNotCalled(s.T(), "GetByValue", mock.Anything, mock.Anything)
}

func (s *CachingTokenDecoratorSuite) TestStore_WriteThroughCache() {
	ctx := context.Background()
	tokenToStore := &domain.Token{Value: "new-token", ExpiresAt: time.Now().Add(30 * time.Minute)}
	cacheKey := "token:value:new-token"
	tokenBytes, _ := json.Marshal(tokenToStore)

	// Arrange: Expect a call to the repo's Store, then the cache's Set.
	s.mockRepo.On("Store", ctx, tokenToStore).Return(nil).Once()
	s.mockCache.On("Set", ctx, cacheKey, tokenBytes, mock.AnythingOfType("time.Duration")).Return(nil).Once()

	// Act
	err := s.cachingRepo.Store(ctx, tokenToStore)

	// Assert
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingTokenDecoratorSuite) TestDelete_InvalidatesCache() {
	ctx := context.Background()
	tokenID := "token123"
	tokenValue := "token-value-to-delete"
	cacheKey := "token:value:token-value-to-delete"
	tokenToDelete := &domain.Token{ID: tokenID, Value: tokenValue}

	// Arrange
	// 1. Expect a call to GetByID on the repo to find the token's value.
	s.mockRepo.On("GetByID", ctx, tokenID).Return(tokenToDelete, nil).Once()
	// 2. Expect the actual Delete call on the repo.
	s.mockRepo.On("Delete", ctx, tokenID).Return(nil).Once()
	// 3. Because deletion succeeded, expect an invalidation call to the cache.
	s.mockCache.On("Delete", ctx, cacheKey).Return(nil).Once()

	// Act
	err := s.cachingRepo.Delete(ctx, tokenID)

	// Assert
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingTokenDecoratorSuite) TestDeleteByUserID_PassThrough() {
	ctx := context.Background()
	userID := "user123"
	tokenType := domain.TokenTypeRefresh

	// Arrange: Expect the call to be passed directly to the wrapped repo.
	s.mockRepo.On("DeleteByUserID", ctx, userID, tokenType).Return(nil).Once()

	// Act
	err := s.cachingRepo.DeleteByUserID(ctx, userID, tokenType)

	// Assert
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	// Assert that the cache was NOT touched for this operation.
	s.mockCache.AssertNotCalled(s.T(), "Delete", mock.Anything, mock.Anything)
}
