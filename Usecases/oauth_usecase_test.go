package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	. "A2SV_Starter_Project_Blog/Usecases"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

// --- Mocks for all dependencies ---
// We reuse the MockUserRepository and MockTokenRepository from other tests.
// A new MockGoogleOAuthService is needed.

type MockGoogleOAuthService struct {
	mock.Mock
}

func (m *MockGoogleOAuthService) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oauth2.Token), args.Error(1)
}
func (m *MockGoogleOAuthService) GetUserInfo(ctx context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GoogleUserInfo), args.Error(1)
}

// --- Test Suite Setup ---
type OAuthUsecaseTestSuite struct {
	suite.Suite
	mockUserRepo   *MockUserRepository
	mockTokenRepo  *MockTokenRepository
	mockJwtService *MockJWTService
	mockGoogleSvc  *MockGoogleOAuthService
	usecase        domain.IOAuthUsecase
}

func (s *OAuthUsecaseTestSuite) SetupTest() {
	s.mockUserRepo = new(MockUserRepository)
	s.mockTokenRepo = new(MockTokenRepository)
	s.mockJwtService = new(MockJWTService)
	s.mockGoogleSvc = new(MockGoogleOAuthService)

	s.usecase = NewOAuthUsecase(
		s.mockUserRepo,
		s.mockTokenRepo,
		s.mockJwtService,
		s.mockGoogleSvc,
		2*time.Second,
	)
}

func TestOAuthUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(OAuthUsecaseTestSuite))
}

// --- Tests ---

func (s *OAuthUsecaseTestSuite) TestHandleGoogleCallback() {
	ctx := context.Background()
	authCode := "valid-auth-code"
	googleToken := &oauth2.Token{AccessToken: "google-access-token"}
	googleUserInfo := &domain.GoogleUserInfo{
		ID:    "google-user-id-123",
		Email: "test@google.com",
		Name:  "Test User",
	}

	// Common mock setup for successful token generation
	setupTokenGenerationMocks := func(s *OAuthUsecaseTestSuite, userID string) {
		accessClaims := &infrastructure.JWTClaims{RegisteredClaims: jwt.RegisteredClaims{ID: "access-jti", ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute))}}
		refreshClaims := &infrastructure.JWTClaims{RegisteredClaims: jwt.RegisteredClaims{ID: "refresh-jti", ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Hour))}}
		s.mockJwtService.On("GenerateAccessToken", userID, domain.RoleUser).Return("our-access-token", accessClaims, nil).Once()
		s.mockJwtService.On("GenerateRefreshToken", userID).Return("our-refresh-token", refreshClaims, nil).Once()
		s.mockTokenRepo.On("Store", mock.Anything, mock.AnythingOfType("*domain.Token")).Return(nil).Twice()
	}

	s.Run("Success - Sign In existing Google user", func() {
		s.SetupTest()
		// Arrange
		existingUser := &domain.User{ID: "our-user-id-abc", Role: domain.RoleUser, IsActive: true}
		s.mockGoogleSvc.On("ExchangeCodeForToken", mock.Anything, authCode).Return(googleToken, nil).Once()
		s.mockGoogleSvc.On("GetUserInfo", mock.Anything, googleToken).Return(googleUserInfo, nil).Once()
		s.mockUserRepo.On("FindByProviderID", mock.Anything, domain.ProviderGoogle, googleUserInfo.ID).Return(existingUser, nil).Once()
		setupTokenGenerationMocks(s, existingUser.ID)

		// Act
		accessToken, refreshToken, err := s.usecase.HandleGoogleCallback(ctx, authCode)

		// Assert
		s.NoError(err)
		s.Equal("our-access-token", accessToken)
		s.Equal("our-refresh-token", refreshToken)
		s.mockGoogleSvc.AssertExpectations(s.T())
		s.mockUserRepo.AssertExpectations(s.T())
		s.mockJwtService.AssertExpectations(s.T())
	})

	s.Run("Success - Sign Up new Google user", func() {
		s.SetupTest()
		generatedUserID := "new-generated-id" // Arrange
		s.mockGoogleSvc.On("ExchangeCodeForToken", mock.Anything, authCode).Return(googleToken, nil).Once()
		s.mockGoogleSvc.On("GetUserInfo", mock.Anything, googleToken).Return(googleUserInfo, nil).Once()
		// 1. FindByProviderID returns not found
		s.mockUserRepo.On("FindByProviderID", mock.Anything, domain.ProviderGoogle, googleUserInfo.ID).Return(nil, nil).Once()
		// 2. GetByEmail also returns not found
		s.mockUserRepo.On("GetByEmail", mock.Anything, googleUserInfo.Email).Return(nil, nil).Once()
		// 3. GetByUsername to check for name conflicts also returns not found
		s.mockUserRepo.On("GetByUsername", mock.Anything, googleUserInfo.Name).Return(nil, nil).Once()
		// 4. Expect Create to be called
		s.mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).
			// 2. Use .Run() to simulate the repository's behavior of setting the ID.
			Run(func(args mock.Arguments) {
				userArg := args.Get(1).(*domain.User)
				userArg.ID = generatedUserID // Simulate setting the ID.
			}).
			Return(nil).Once()
		setupTokenGenerationMocks(s, generatedUserID) // User ID is generated by repo

		// Act
		_, _, err := s.usecase.HandleGoogleCallback(ctx, authCode)

		// Assert
		s.NoError(err)
		s.mockGoogleSvc.AssertExpectations(s.T())
		s.mockUserRepo.AssertExpectations(s.T())
	})

	s.Run("Failure - Email already exists with local provider", func() {
		s.SetupTest()
		// Arrange
		existingLocalUser := &domain.User{Provider: domain.ProviderLocal}
		s.mockGoogleSvc.On("ExchangeCodeForToken", mock.Anything, authCode).Return(googleToken, nil).Once()
		s.mockGoogleSvc.On("GetUserInfo", mock.Anything, googleToken).Return(googleUserInfo, nil).Once()
		// 1. FindByProviderID returns not found
		s.mockUserRepo.On("FindByProviderID", mock.Anything, domain.ProviderGoogle, googleUserInfo.ID).Return(nil, nil).Once()
		// 2. GetByEmail finds an existing local user
		s.mockUserRepo.On("GetByEmail", mock.Anything, googleUserInfo.Email).Return(existingLocalUser, nil).Once()

		// Act
		_, _, err := s.usecase.HandleGoogleCallback(ctx, authCode)

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrEmailExists)
		s.mockUserRepo.AssertExpectations(s.T())
		// Ensure Create is not called
		s.mockUserRepo.AssertNotCalled(s.T(), "Create")
	})

	s.Run("Failure - Google service fails to exchange code", func() {
		s.SetupTest()
		// Arrange
		expectedErr := errors.New("invalid_grant")
		s.mockGoogleSvc.On("ExchangeCodeForToken", mock.Anything, authCode).Return(nil, expectedErr).Once()

		// Act
		_, _, err := s.usecase.HandleGoogleCallback(ctx, authCode)

		// Assert
		s.Error(err)
		s.ErrorIs(err, expectedErr)
		// Ensure no other methods were called
		s.mockGoogleSvc.AssertNotCalled(s.T(), "GetUserInfo")
		s.mockUserRepo.AssertNotCalled(s.T(), "FindByProviderID")
	})
}
