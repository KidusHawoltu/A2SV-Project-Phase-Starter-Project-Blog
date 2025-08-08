package usecases_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"errors"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- MOCK DEFINITIONS ---

type MockUserRepository struct{ mock.Mock }

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
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
func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *MockUserRepository) FindByProviderID(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error) {
	args := m.Called(ctx, provider, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) FindUserIDsByName(ctx context.Context, name string) ([]string, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
func (m *MockUserRepository) SearchAndFilter(ctx context.Context, options domain.UserSearchFilterOptions) ([]*domain.User, int64, error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, int64(args.Int(1)), args.Error(2)
	}
	return args.Get(0).([]*domain.User), int64(args.Int(1)), args.Error(2)
}

type MockTokenRepository struct{ mock.Mock }

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
func (m *MockTokenRepository) Delete(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}
func (m *MockTokenRepository) DeleteByUserID(ctx context.Context, userID string, tokenType domain.TokenType) error {
	args := m.Called(ctx, userID, tokenType)
	return args.Error(0)
}

type MockPasswordService struct{ mock.Mock }

func (m *MockPasswordService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}
func (m *MockPasswordService) ComparePassword(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

type MockJWTService struct{ mock.Mock }

func (m *MockJWTService) GenerateAccessToken(userID string, role domain.Role) (string, *infrastructure.JWTClaims, error) {
	args := m.Called(userID, role)
	if args.Error(2) != nil {
		return args.String(0), nil, args.Error(2)
	}
	return args.String(0), args.Get(1).(*infrastructure.JWTClaims), args.Error(2)
}
func (m *MockJWTService) GenerateRefreshToken(userID string) (string, *infrastructure.JWTClaims, error) {
	args := m.Called(userID)
	if args.Error(2) != nil {
		return args.String(0), nil, args.Error(2)
	}
	return args.String(0), args.Get(1).(*infrastructure.JWTClaims), args.Error(2)
}
func (m *MockJWTService) ValidateToken(tokenString string) (*infrastructure.JWTClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infrastructure.JWTClaims), args.Error(1)
}
func (m *MockJWTService) ParseExpiredToken(tokenString string) (*infrastructure.JWTClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infrastructure.JWTClaims), args.Error(1)
}
func (m *MockJWTService) GetRefreshTokenExpiry() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

type MockEmailService struct{ mock.Mock }

func (m *MockEmailService) SendPasswordResetEmail(to, user, token string) error {
	args := m.Called(to, user, token)
	return args.Error(0)
}
func (m *MockEmailService) SendActivationEmail(to, user, token string) error {
	args := m.Called(to, user, token)
	return args.Error(0)
}

type MockImageUploaderService struct{ mock.Mock }

func (m *MockImageUploaderService) UploadProfilePicture(file multipart.File, header *multipart.FileHeader) (string, error) {
	args := m.Called(file, header)
	return args.String(0), args.Error(1)
}

type mockMultipartFile struct {
	*strings.Reader
}

func (m *mockMultipartFile) Close() error {
	return nil
}

// --- TEST FUNCTIONS ---

func TestUserUsecase_Register(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		mockPassSvc := new(MockPasswordService)
		mockEmailSvc := new(MockEmailService)
		uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, nil, mockTokenRepo, mockEmailSvc, nil, 2*time.Second)

		password := "password123"
		user := &domain.User{Username: "test", Email: "test@test.com", Password: &password}

		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(nil, nil).Once()
		mockUserRepo.On("GetByUsername", mock.Anything, user.Username).Return(nil, nil).Once()
		mockPassSvc.On("HashPassword", *user.Password).Return("hashed_password", nil).Once()
		mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.AnythingOfType("*domain.Token")).Return(nil).Once()
		mockEmailSvc.On("SendActivationEmail", user.Email, user.Username, mock.AnythingOfType("string")).Return(nil).Once()

		err := uc.Register(context.Background(), user)

		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
		mockPassSvc.AssertExpectations(t)
		mockEmailSvc.AssertExpectations(t)
	})
}

func TestUserUsecase_Login(t *testing.T) {
	password := "hashed_password"
	user := &domain.User{
		ID:       "user-123",
		Email:    "test@test.com",
		Username: "testuser",
		Password: &password,
		IsActive: true,
		Role:     domain.RoleUser,
		Provider: domain.ProviderLocal,
	}
	accessClaims := &infrastructure.JWTClaims{UserID: user.ID, RegisteredClaims: jwt.RegisteredClaims{ID: "access-jti-123", ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute))}}
	refreshClaims := &infrastructure.JWTClaims{UserID: user.ID, RegisteredClaims: jwt.RegisteredClaims{ID: "refresh-jti-456", ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour))}}

	t.Run("Success - Login with Email", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		mockPassSvc := new(MockPasswordService)
		mockJwtSvc := new(MockJWTService)
		uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, mockJwtSvc, mockTokenRepo, nil, nil, 2*time.Second)
		// Arrange
		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil).Once()
		mockPassSvc.On("ComparePassword", *user.Password, "password123").Return(nil).Once()
		mockJwtSvc.On("GenerateAccessToken", user.ID, user.Role).Return("access.token", accessClaims, nil).Once()
		mockJwtSvc.On("GenerateRefreshToken", user.ID).Return("refresh.token", refreshClaims, nil).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.AnythingOfType("*domain.Token")).Return(nil).Twice()

		access, refresh, err := uc.Login(context.Background(), user.Email, "password123")

		assert.NoError(t, err)
		assert.Equal(t, "access.token", access)
		assert.Equal(t, "refresh.token", refresh)
		mockUserRepo.AssertExpectations(t)
		mockPassSvc.AssertExpectations(t)
		mockJwtSvc.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Success - Login with Username", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		mockPassSvc := new(MockPasswordService)
		mockJwtSvc := new(MockJWTService)
		uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, mockJwtSvc, mockTokenRepo, nil, nil, 2*time.Second)
		// Arrange
		mockUserRepo.On("GetByUsername", mock.Anything, user.Username).Return(user, nil).Once()
		mockPassSvc.On("ComparePassword", *user.Password, "password123").Return(nil).Once()
		mockJwtSvc.On("GenerateAccessToken", user.ID, user.Role).Return("access.token", accessClaims, nil).Once()
		mockJwtSvc.On("GenerateRefreshToken", user.ID).Return("refresh.token", refreshClaims, nil).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.AnythingOfType("*domain.Token")).Return(nil).Twice()

		access, refresh, err := uc.Login(context.Background(), user.Username, "password123")

		assert.NoError(t, err)
		assert.Equal(t, "access.token", access)
		assert.Equal(t, "refresh.token", refresh)
		mockUserRepo.AssertExpectations(t)
		mockPassSvc.AssertExpectations(t)
		mockJwtSvc.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
	})

	t.Run("Failure - Attempt to log in as Google user", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)

		googleUser := &domain.User{ID: "user-456", Email: "googleuser@test.com", IsActive: true, Role: domain.RoleUser, Provider: domain.ProviderGoogle}
		mockUserRepo.On("GetByEmail", mock.Anything, googleUser.Email).Return(googleUser, nil).Once()

		_, _, err := uc.Login(context.Background(), googleUser.Email, "any-password")

		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrOAuthUser)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Failure - User Not Found", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)

		mockUserRepo.On("GetByEmail", mock.Anything, "notfound@test.com").Return(nil, nil).Once()

		_, _, err := uc.Login(context.Background(), "notfound@test.com", "any-password")

		assert.Error(t, err)
		assert.Equal(t, domain.ErrAuthenticationFailed, err)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Failure - Incorrect Password", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockPassSvc := new(MockPasswordService)
		uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, nil, nil, nil, nil, 2*time.Second)

		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil).Once()
		mockPassSvc.On("ComparePassword", *user.Password, "wrong-password").Return(errors.New("crypto error")).Once()

		_, _, err := uc.Login(context.Background(), user.Email, "wrong-password")

		assert.Error(t, err)
		assert.Equal(t, domain.ErrAuthenticationFailed, err)
		mockUserRepo.AssertExpectations(t)
		mockPassSvc.AssertExpectations(t)
	})

	t.Run("Failure - Account Not Active", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)

		inactiveUser := &domain.User{ID: "user-inactive", Email: "inactive@test.com", IsActive: false, Provider: domain.ProviderLocal}
		mockUserRepo.On("GetByEmail", mock.Anything, "inactive@test.com").Return(inactiveUser, nil).Once()

		_, _, err := uc.Login(context.Background(), "inactive@test.com", "any-password")

		assert.Error(t, err)
		assert.Equal(t, domain.ErrAccountNotActive, err)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_Logout(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockTokenRepo := new(MockTokenRepository)
		uc := usecases.NewUserUsecase(nil, nil, nil, mockTokenRepo, nil, nil, 2*time.Second)
		token := &domain.Token{ID: "token-id", UserID: "user-123", Type: domain.TokenTypeRefresh}

		mockTokenRepo.On("GetByValue", mock.Anything, "valid.token").Return(token, nil).Once()
		mockTokenRepo.On("Delete", mock.Anything, token.ID).Return(nil).Once()

		err := uc.Logout(context.Background(), "valid.token")
		assert.NoError(t, err)
		mockTokenRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_ActivateAccount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, mockTokenRepo, nil, nil, 2*time.Second)
		token := &domain.Token{ID: "token-id", UserID: "user-123", Type: domain.TokenTypeActivation, ExpiresAt: time.Now().Add(1 * time.Hour)}
		user := &domain.User{ID: "user-123", IsActive: false}

		mockTokenRepo.On("GetByValue", mock.Anything, "valid.token").Return(token, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, "user-123").Return(user, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool { return u.IsActive })).Return(nil).Once()
		mockTokenRepo.On("Delete", mock.Anything, "token-id").Return(nil).Once()

		err := uc.ActivateAccount(context.Background(), "valid.token")
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_ForgetPassword(t *testing.T) {
	user := &domain.User{ID: "user-123", Email: "user@example.com", Username: "testuser", Provider: domain.ProviderLocal}

	t.Run("Success - Local User", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		mockEmailSvc := new(MockEmailService)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, mockTokenRepo, mockEmailSvc, nil, 2*time.Second)

		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil).Once()
		mockTokenRepo.On("DeleteByUserID", mock.Anything, user.ID, domain.TokenTypePasswordReset).Return(nil).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.AnythingOfType("*domain.Token")).Return(nil).Once()
		mockEmailSvc.On("SendPasswordResetEmail", user.Email, user.Username, mock.Anything).Return(nil).Once()

		err := uc.ForgetPassword(context.Background(), user.Email)
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
		mockEmailSvc.AssertExpectations(t)
	})

	t.Run("Success - Google User (No Action)", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		mockEmailSvc := new(MockEmailService)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, mockTokenRepo, mockEmailSvc, nil, 2*time.Second)
		googleUser := &domain.User{ID: "user-456", Email: "google@example.com", Username: "googleuser", Provider: domain.ProviderGoogle}

		mockUserRepo.On("GetByEmail", mock.Anything, googleUser.Email).Return(googleUser, nil).Once()

		err := uc.ForgetPassword(context.Background(), googleUser.Email)

		assert.NoError(t, err, "Should not return an error for a Google user to prevent email enumeration")
		mockUserRepo.AssertExpectations(t)
		mockTokenRepo.AssertNotCalled(t, "DeleteByUserID")
		mockTokenRepo.AssertNotCalled(t, "Store")
		mockEmailSvc.AssertNotCalled(t, "SendPasswordResetEmail")
	})
}

func TestUserUsecase_ResetPassword(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockTokenRepo := new(MockTokenRepository)
		mockPassSvc := new(MockPasswordService)
		uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, nil, mockTokenRepo, nil, nil, 2*time.Second)
		token := &domain.Token{ID: "token-id", UserID: "user-123", Type: domain.TokenTypePasswordReset, ExpiresAt: time.Now().Add(1 * time.Hour)}
		user := &domain.User{ID: "user-123", Provider: domain.ProviderLocal}

		mockTokenRepo.On("GetByValue", mock.Anything, "valid.token").Return(token, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, "user-123").Return(user, nil).Once()
		mockPassSvc.On("HashPassword", "new-password123").Return("new_hashed_password", nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool { return *u.Password == "new_hashed_password" })).Return(nil).Once()
		mockTokenRepo.On("Delete", mock.Anything, "token-id").Return(nil).Once()

		err := uc.ResetPassword(context.Background(), "valid.token", "new-password123")
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
		mockTokenRepo.AssertExpectations(t)
		mockPassSvc.AssertExpectations(t)
	})
}

func TestUserUsecase_UpdateProfile(t *testing.T) {
	userID := "user-123"

	t.Run("Success - Update bio only", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		user := &domain.User{ID: userID, Bio: "old bio", ProfilePicture: "old.url"}

		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.Bio == "new bio" && u.ProfilePicture == "old.url"
		})).Return(nil).Once()

		updatedUser, err := uc.UpdateProfile(context.Background(), userID, "new bio", nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "new bio", updatedUser.Bio)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Success - Update profile picture only", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockImageUploader := new(MockImageUploaderService)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, mockImageUploader, 2*time.Second)
		newImageURL := "http://example.com/new_image.jpg"
		userForTest := &domain.User{ID: userID, Bio: "old bio", ProfilePicture: "old.url"}

		mockUserRepo.On("GetByID", mock.Anything, userID).Return(userForTest, nil).Once()
		mockImageUploader.On("UploadProfilePicture", mock.AnythingOfType("*usecases_test.mockMultipartFile"), mock.AnythingOfType("*multipart.FileHeader")).Return(newImageURL, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.ProfilePicture == newImageURL && u.Bio == "old bio"
		})).Return(nil).Once()

		mockFile := &mockMultipartFile{Reader: strings.NewReader("dummy content")}
		mockHeader := &multipart.FileHeader{Filename: "test.jpg"}

		updatedUser, err := uc.UpdateProfile(context.Background(), userID, "old bio", mockFile, mockHeader)
		assert.NoError(t, err)
		assert.NotNil(t, updatedUser)
		assert.Equal(t, newImageURL, updatedUser.ProfilePicture)
		mockUserRepo.AssertExpectations(t)
		mockImageUploader.AssertExpectations(t)
	})

	t.Run("Failure - Image upload service returns an error", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockImageUploader := new(MockImageUploaderService)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, mockImageUploader, 2*time.Second)
		user := &domain.User{ID: userID}
		expectedErr := errors.New("upload failed")

		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
		mockImageUploader.On("UploadProfilePicture", mock.Anything, mock.Anything).Return("", expectedErr).Once()

		mockFile := &mockMultipartFile{Reader: strings.NewReader("dummy content")}
		mockHeader := &multipart.FileHeader{Filename: "test.jpg"}

		_, err := uc.UpdateProfile(context.Background(), userID, "any bio", mockFile, mockHeader)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockUserRepo.AssertExpectations(t)
		mockImageUploader.AssertExpectations(t)
		mockUserRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("Failure - User not found", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(nil, domain.ErrUserNotFound).Once()

		_, err := uc.UpdateProfile(context.Background(), userID, "any bio", nil, nil)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_SearchAndFilter(t *testing.T) {
	t.Run("Success - Basic Search with Defaults", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		expectedUsers := []*domain.User{{ID: "user-1"}, {ID: "user-2"}}
		var expectedTotal int64 = 15
		inputOptions := domain.UserSearchFilterOptions{Page: 0, Limit: 0}
		expectedOptions := domain.UserSearchFilterOptions{Page: 1, Limit: 10}

		mockUserRepo.On("SearchAndFilter", mock.Anything, expectedOptions).Return(expectedUsers, int(expectedTotal), nil).Once()
		users, total, err := uc.SearchAndFilter(context.Background(), inputOptions)

		assert.NoError(t, err)
		assert.Equal(t, expectedTotal, total)
		assert.Len(t, users, 2)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Success - Search with Specific Pagination", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		expectedUsers := []*domain.User{{ID: "user-1"}, {ID: "user-2"}}
		var expectedTotal int64 = 15
		inputOptions := domain.UserSearchFilterOptions{Page: 2, Limit: 20}

		mockUserRepo.On("SearchAndFilter", mock.Anything, inputOptions).Return(expectedUsers, int(expectedTotal), nil).Once()
		users, total, err := uc.SearchAndFilter(context.Background(), inputOptions)

		assert.NoError(t, err)
		assert.Equal(t, expectedTotal, total)
		assert.Equal(t, expectedUsers, users)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Success - Max Limit is Enforced", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		expectedUsers := []*domain.User{{ID: "user-1"}, {ID: "user-2"}}
		var expectedTotal int64 = 15
		inputOptions := domain.UserSearchFilterOptions{Page: 1, Limit: 500}
		expectedOptions := domain.UserSearchFilterOptions{Page: 1, Limit: 100}

		mockUserRepo.On("SearchAndFilter", mock.Anything, expectedOptions).Return(expectedUsers, int(expectedTotal), nil).Once()
		_, _, err := uc.SearchAndFilter(context.Background(), inputOptions)

		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Failure - Repository Returns an Error", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		inputOptions := domain.UserSearchFilterOptions{Page: 1, Limit: 10}
		expectedError := errors.New("database connection failed")

		mockUserRepo.On("SearchAndFilter", mock.Anything, inputOptions).Return(nil, 0, expectedError).Once()
		users, total, err := uc.SearchAndFilter(context.Background(), inputOptions)

		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.Nil(t, users)
		assert.Zero(t, total)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_SetUserRole(t *testing.T) {
	adminUser := &domain.User{ID: "admin-123", Role: domain.RoleAdmin}
	regularUser := &domain.User{ID: "user-456", Role: domain.RoleUser}

	t.Run("Success - Admin promotes a User", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		targetUser := &domain.User{ID: "target-789", Role: domain.RoleUser}

		mockUserRepo.On("GetByID", mock.Anything, "target-789").Return(targetUser, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.ID == "target-789" && u.Role == domain.RoleAdmin
		})).Return(nil).Once()

		updatedUser, err := uc.SetUserRole(context.Background(), adminUser.ID, adminUser.Role, targetUser.ID, domain.RoleAdmin)
		assert.NoError(t, err)
		assert.NotNil(t, updatedUser)
		assert.Equal(t, domain.RoleAdmin, updatedUser.Role)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Success - Admin demotes another Admin", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		targetUser := &domain.User{ID: "target-admin-000", Role: domain.RoleAdmin}

		mockUserRepo.On("GetByID", mock.Anything, "target-admin-000").Return(targetUser, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.ID == "target-admin-000" && u.Role == domain.RoleUser
		})).Return(nil).Once()

		updatedUser, err := uc.SetUserRole(context.Background(), adminUser.ID, adminUser.Role, targetUser.ID, domain.RoleUser)
		assert.NoError(t, err)
		assert.NotNil(t, updatedUser)
		assert.Equal(t, domain.RoleUser, updatedUser.Role)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Success - No update needed if role is already correct", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		targetUser := &domain.User{ID: "target-789", Role: domain.RoleAdmin}

		mockUserRepo.On("GetByID", mock.Anything, "target-789").Return(targetUser, nil).Once()

		updatedUser, err := uc.SetUserRole(context.Background(), adminUser.ID, adminUser.Role, targetUser.ID, domain.RoleAdmin)
		assert.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, updatedUser.Role)
		mockUserRepo.AssertExpectations(t)
		mockUserRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("Failure - Actor is not an Admin", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		_, err := uc.SetUserRole(context.Background(), regularUser.ID, regularUser.Role, "any-target-id", domain.RoleAdmin)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPermissionDenied)
	})

	t.Run("Failure - Admin tries to change their own role", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		_, err := uc.SetUserRole(context.Background(), adminUser.ID, adminUser.Role, adminUser.ID, domain.RoleUser)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrCannotChangeOwnRole)
	})

	t.Run("Failure - Target user not found", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		mockUserRepo.On("GetByID", mock.Anything, "non-existent-id").Return(nil, domain.ErrUserNotFound).Once()
		_, err := uc.SetUserRole(context.Background(), adminUser.ID, adminUser.Role, "non-existent-id", domain.RoleAdmin)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Failure - Invalid new role provided", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		_, err := uc.SetUserRole(context.Background(), adminUser.ID, adminUser.Role, regularUser.ID, domain.Role("super-user"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidRole)
	})

	t.Run("Failure - Repository fails on Update", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, nil, 2*time.Second)
		targetUser := &domain.User{ID: "target-789", Role: domain.RoleUser}
		expectedError := errors.New("database write error")
		mockUserRepo.On("GetByID", mock.Anything, "target-789").Return(targetUser, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.Anything).Return(expectedError).Once()
		_, err := uc.SetUserRole(context.Background(), adminUser.ID, adminUser.Role, targetUser.ID, domain.RoleAdmin)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		mockUserRepo.AssertExpectations(t)
	})
}
