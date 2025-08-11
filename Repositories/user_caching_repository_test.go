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

// --- Mock Implementations ---

// MockUserRepository is a mock for usecases.UserRepository
type MockUserRepository struct {
	mock.Mock
}

// Implement all methods of the usecases.UserRepository interface
func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *MockUserRepository) FindUserIDsByName(ctx context.Context, authorName string) ([]string, error) {
	args := m.Called(ctx, authorName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockUserRepository) FindByProviderID(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error) {
	args := m.Called(ctx, provider, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) SearchAndFilter(ctx context.Context, opts domain.UserSearchFilterOptions) ([]*domain.User, int64, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.User), args.Get(1).(int64), args.Error(2)
}

// MockCacheService is a mock for domain.ICacheService
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}
func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}
func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}
func (m *MockCacheService) AddToSet(ctx context.Context, key string, members ...interface{}) error {
	args := m.Called(ctx, key, members)
	return args.Error(0)
}
func (m *MockCacheService) GetSetMembers(ctx context.Context, key string) ([]string, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockCacheService) DeleteKeys(ctx context.Context, keys []string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

// --- The Test Suite ---

type CachingUserDecoratorSuite struct {
	suite.Suite
	mockRepo    *MockUserRepository
	mockCache   *MockCacheService
	cachingRepo usecases.UserRepository // The decorator we are testing
}

func (s *CachingUserDecoratorSuite) SetupTest() {
	s.mockRepo = new(MockUserRepository)
	s.mockCache = new(MockCacheService)
	// Inject the mocks into the decorator
	s.cachingRepo = NewCachingUserRepository(s.mockRepo, s.mockCache)
}

func TestCachingUserDecoratorSuite(t *testing.T) {
	suite.Run(t, new(CachingUserDecoratorSuite))
}

// --- The Actual Tests ---

func (s *CachingUserDecoratorSuite) TestGetByID_CacheMiss() {
	ctx := context.Background()
	userID := "user123"
	cacheKey := "user:id:user123"
	expectedUser := &domain.User{ID: userID, Username: "testuser"}
	userBytes, _ := json.Marshal(expectedUser)

	// --- Arrange: Define mock expectations ---
	// 1. Expect a call to cache.Get, and make it return a "not found" error.
	s.mockCache.On("Get", ctx, cacheKey).Return(nil, domain.ErrNotFound).Once()

	// 2. Because of the cache miss, expect a call to the wrapped repo's GetByID.
	s.mockRepo.On("GetByID", ctx, userID).Return(expectedUser, nil).Once()

	// 3. Because we got a result from the repo, expect a call to cache.Set.
	s.mockCache.On("Set", ctx, cacheKey, userBytes, 1*time.Hour).Return(nil).Once()

	// --- Act: Call the method on the decorator ---
	resultUser, err := s.cachingRepo.GetByID(ctx, userID)

	// --- Assert ---
	s.NoError(err)
	s.Equal(expectedUser, resultUser)

	// Verify that all the expected mock calls were made.
	s.mockCache.AssertExpectations(s.T())
	s.mockRepo.AssertExpectations(s.T())
}

func (s *CachingUserDecoratorSuite) TestGetByID_CacheHit() {
	ctx := context.Background()
	userID := "user123"
	cacheKey := "user:id:user123"
	expectedUser := &domain.User{ID: userID, Username: "testuser"}
	userBytes, _ := json.Marshal(expectedUser)

	// --- Arrange: Define mock expectations ---
	// 1. Expect a call to cache.Get, and make it return the user data.
	s.mockCache.On("Get", ctx, cacheKey).Return(userBytes, nil).Once()

	// --- Act ---
	resultUser, err := s.cachingRepo.GetByID(ctx, userID)

	// --- Assert ---
	s.NoError(err)
	s.Equal(expectedUser.Username, resultUser.Username)

	// Verify cache was called.
	s.mockCache.AssertExpectations(s.T())
	// Crucially, assert that GetByID was NEVER called on the wrapped repo.
	s.mockRepo.AssertNotCalled(s.T(), "GetByID", mock.Anything, mock.Anything)
}

func (s *CachingUserDecoratorSuite) TestUpdate_InvalidatesCache() {
	ctx := context.Background()
	userToUpdate := &domain.User{ID: "user123", Username: "updatedName"}
	cacheKey := "user:id:user123"

	// --- Arrange ---
	// 1. Expect a call to the wrapped repo's Update method.
	s.mockRepo.On("Update", ctx, userToUpdate).Return(nil).Once()

	// 2. Because the update succeeded, expect a call to cache.Delete.
	s.mockCache.On("Delete", ctx, cacheKey).Return(nil).Once()

	// --- Act ---
	err := s.cachingRepo.Update(ctx, userToUpdate)

	// --- Assert ---
	s.NoError(err)
	s.mockRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *CachingUserDecoratorSuite) TestPassThroughMethods() {
	ctx := context.Background()

	// This test verifies that methods we don't cache are passed through correctly.
	s.Run("GetByEmail", func() {
		expectedUser := &domain.User{Email: "test@example.com"}
		s.mockRepo.On("GetByEmail", ctx, "test@example.com").Return(expectedUser, nil).Once()

		user, err := s.cachingRepo.GetByEmail(ctx, "test@example.com")

		s.NoError(err)
		s.Equal(expectedUser, user)
		s.mockRepo.AssertExpectations(s.T())
	})

	// Reset mocks for the next subtest
	s.SetupTest()

	s.Run("Create", func() {
		userToCreate := &domain.User{Username: "newuser"}
		s.mockRepo.On("Create", ctx, userToCreate).Return(nil).Once()

		err := s.cachingRepo.Create(ctx, userToCreate)

		s.NoError(err)
		s.mockRepo.AssertExpectations(s.T())
	})

	// You can add more subtests here for other pass-through methods if desired.
}
