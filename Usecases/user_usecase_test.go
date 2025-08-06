package usecases_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- MOCKS FOR ALL PHASE 3 DEPENDENCIES ---

type MockUserRepository struct{ mock.Mock }

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

type MockTokenRepository struct{ mock.Mock }

func (m *MockTokenRepository) Store(ctx context.Context, token *domain.Token) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}
func (m *MockTokenRepository) GetByValue(ctx context.Context, tokenValue string) (*domain.Token, error) {
	args := m.Called(ctx, tokenValue)
	if args.Get(0) == nil { return nil, args.Error(1) }
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
	// Return nil for claims if the mock is set up to return an error, to avoid nil pointer issues.
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

// THIS IS THE METHOD THAT WAS MISSING
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

// --- TEST FUNCTIONS ---

// Your correct test function
func TestUserUsecase_Register(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockJWTService := new(MockJWTService)
	mockTokenRepo := new(MockTokenRepository)
	mockPassSvc := new(MockPasswordService)
	mockEmailSvc := new(MockEmailService)
	uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, mockJWTService, mockTokenRepo, mockEmailSvc, 2*time.Second)

	t.Run("Success", func(t *testing.T) {
		user := &domain.User{Username: "test", Email: "test@test.com", Password: "password123"}
		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(nil, nil).Once()
		mockUserRepo.On("GetByUsername", mock.Anything, user.Username).Return(nil, nil).Once()
		mockPassSvc.On("HashPassword", user.Password).Return("hashed_password", nil).Once()
		mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.AnythingOfType("*domain.Token")).Return(nil).Once()
		mockEmailSvc.On("SendActivationEmail", user.Email, user.Username, mock.AnythingOfType("string")).Return(nil).Once()

		err := uc.Register(context.Background(), user)
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
	})
}

// Your correct test function
func TestUserUsecase_Login(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	mockPassSvc := new(MockPasswordService)
	mockJwtSvc := new(MockJWTService)
	uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, mockJwtSvc, mockTokenRepo, nil, 2*time.Second)
	user := &domain.User{ID: "user-123", Email: "test@test.com", Password: "hashed_password", IsActive: true}

	t.Run("Success", func(t *testing.T) {
		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil).Once()
		mockPassSvc.On("ComparePassword", user.Password, "password123").Return(nil).Once()
		mockJwtSvc.On("GenerateAccessToken", mock.Anything, mock.Anything).Return("access.token", &infrastructure.JWTClaims{}, nil).Once()
		mockJwtSvc.On("GenerateRefreshToken", mock.Anything).Return("refresh.token", &infrastructure.JWTClaims{}, nil).Once()
		mockJwtSvc.On("GetRefreshTokenExpiry").Return(1 * time.Hour).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.Anything).Return(nil).Twice()

		access, refresh, err := uc.Login(context.Background(), user.Email, "password123")
		assert.NoError(t, err)
		assert.Equal(t, "access.token", access)
		assert.Equal(t, "refresh.token", refresh)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Failure - Account Not Active", func(t *testing.T) {
		inactiveUser := &domain.User{ID: "user-inactive", IsActive: false}
		mockUserRepo.On("GetByEmail", mock.Anything, "inactive@test.com").Return(inactiveUser, nil).Once()
		_, _, err := uc.Login(context.Background(), "inactive@test.com", "any-password")
		assert.Error(t, err)
		assert.Equal(t, domain.ErrAccountNotActive, err)
		mockUserRepo.AssertExpectations(t)
	})
}

// --- ADDED TESTS FOR COMPLETENESS ---

func TestUserUsecase_Logout(t *testing.T) {
	mockTokenRepo := new(MockTokenRepository)
	uc := usecases.NewUserUsecase(nil, nil, nil, mockTokenRepo, nil, 2*time.Second)
	token := &domain.Token{ID: "token-id", UserID: "user-123", Type: domain.TokenTypeRefresh}

	t.Run("Success", func(t *testing.T) {
		mockTokenRepo.On("GetByValue", mock.Anything, "valid.token").Return(token, nil).Once()
		mockTokenRepo.On("DeleteByUserID", mock.Anything, "user-123", domain.TokenTypeRefresh).Return(nil).Once()
		mockTokenRepo.On("DeleteByUserID", mock.Anything, "user-123", domain.TokenTypeAccessToken).Return(nil).Once()

		err := uc.Logout(context.Background(), "valid.token")
		assert.NoError(t, err)
		mockTokenRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_ActivateAccount(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, mockTokenRepo, nil, 2*time.Second)
	token := &domain.Token{ID: "token-id", UserID: "user-123", Type: domain.TokenTypeActivation}
	user := &domain.User{ID: "user-123", IsActive: false}

	t.Run("Success", func(t *testing.T) {
		mockTokenRepo.On("GetByValue", mock.Anything, "valid.token").Return(token, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, "user-123").Return(user, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool { return u.IsActive })).Return(nil).Once()
		mockTokenRepo.On("Delete", mock.Anything, "token-id").Return(nil).Once()

		err := uc.ActivateAccount(context.Background(), "valid.token")
		assert.NoError(t, err)
		mockTokenRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_ForgotPassword(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	mockEmailSvc := new(MockEmailService)
	uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, mockTokenRepo, mockEmailSvc, 2*time.Second)
	user := &domain.User{ID: "user-123", Email: "user@example.com", Username: "testuser"}

	t.Run("Success", func(t *testing.T) {
		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil).Once()
		mockTokenRepo.On("DeleteByUserID", mock.Anything, user.ID, domain.TokenTypePasswordReset).Return(nil).Once()
		mockTokenRepo.On("Store", mock.Anything, mock.AnythingOfType("*domain.Token")).Return(nil).Once()
		mockEmailSvc.On("SendPasswordResetEmail", user.Email, user.Username, mock.Anything).Return(nil).Once()

		err := uc.ForgetPassword(context.Background(), user.Email)
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_ResetPassword(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockTokenRepo := new(MockTokenRepository)
	mockPassSvc := new(MockPasswordService)
	uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, nil, mockTokenRepo, nil, 2*time.Second)
	token := &domain.Token{ID: "token-id", UserID: "user-123", Type: domain.TokenTypePasswordReset}
	user := &domain.User{ID: "user-123"}

	t.Run("Success", func(t *testing.T) {
		mockTokenRepo.On("GetByValue", mock.Anything, "valid.token").Return(token, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, "user-123").Return(user, nil).Once()
		mockPassSvc.On("HashPassword", "new-password123").Return("new_hashed_password", nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool { return u.Password == "new_hashed_password" })).Return(nil).Once()
		mockTokenRepo.On("Delete", mock.Anything, "token-id").Return(nil).Once()

		err := uc.ResetPassword(context.Background(), "valid.token", "new-password123")
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_UpdateProfile(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	uc := usecases.NewUserUsecase(mockUserRepo, nil, nil, nil, nil, 2*time.Second)
	user := &domain.User{ID: "user-123"}
	
	t.Run("Success", func(t *testing.T) {
		mockUserRepo.On("GetByID", mock.Anything, "user-123").Return(user, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
		
		updatedUser, err := uc.UpdateProfile(context.Background(), "user-123", "new bio", "new.url")
		
		assert.NoError(t, err)
		assert.Equal(t, "new bio", updatedUser.Bio)
		assert.Equal(t, "new.url", updatedUser.ProfilePicture)
		mockUserRepo.AssertExpectations(t)
	})
}