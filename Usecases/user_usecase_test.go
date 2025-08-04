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

// --- Mocks ---

type MockUserRepository struct {
	mock.Mock
}

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

func (m *MockUserRepository) FindUserIDsByName(ctx context.Context, name string) ([]string, error) {
	args := m.Called(ctx, name)
	return args.Get(0).([]string), args.Error(1)
}

type MockPasswordService struct {
	mock.Mock
}

func (m *MockPasswordService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}
func (m *MockPasswordService) ComparePassword(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

type MockJWTService struct {
	mock.Mock
}

// ValidateToken implements infrastructure.JWTService.
func (m *MockJWTService) ValidateToken(tokenString string) (*infrastructure.JWTClaims, error) {
	panic("unimplemented")
}

func (m *MockJWTService) GenerateAccessToken(userID string, role domain.Role) (string, error) {
	args := m.Called(userID, role)
	return args.String(0), args.Error(1)
}

// --- Tests ---

func TestUserUsecase_Register(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockPassSvc := new(MockPasswordService)
	// We don't need JWT service for register, so we can pass nil or a nil mock

	uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, nil, 2*time.Second)

	t.Run("Success", func(t *testing.T) {
		user := &domain.User{Username: "test", Email: "test@test.com", Password: "password123", Role: domain.RoleUser}

		// Setup expectations
		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(nil, nil).Once()
		mockPassSvc.On("HashPassword", user.Password).Return("hashed_password", nil).Once()
		mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil).Once()

		// Act
		err := uc.Register(context.Background(), user)

		// Assert
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
		mockPassSvc.AssertExpectations(t)
	})

	t.Run("Email already exists", func(t *testing.T) {
		existingUser := &domain.User{
			Email:    "exists@test.com",
			Username: "test user",
			Password: "12345678j",
			Role:     domain.RoleUser,
		}

		mockUserRepo.On("GetByEmail", mock.Anything, existingUser.Email).Return(existingUser, nil).Once()

		err := uc.Register(context.Background(), existingUser)

		assert.Equal(t, domain.ErrEmailExists, err)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserUsecase_Login(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockPassSvc := new(MockPasswordService)
	mockJwtSvc := new(MockJWTService)

	uc := usecases.NewUserUsecase(mockUserRepo, mockPassSvc, mockJwtSvc, 2*time.Second)
	user := &domain.User{ID: "user-123", Email: "test@test.com", Username: "testuser", Password: "hashed_password", Role: domain.RoleUser}

	t.Run("Success with Email", func(t *testing.T) {
		mockUserRepo.On("GetByEmail", mock.Anything, user.Email).Return(user, nil).Once()
		mockPassSvc.On("ComparePassword", user.Password, "password123").Return(nil).Once()
		mockJwtSvc.On("GenerateAccessToken", user.ID, user.Role).Return("valid.token", nil).Once()

		token, err := uc.Login(context.Background(), user.Email, "password123")
		assert.NoError(t, err)
		assert.Equal(t, "valid.token", token)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Success with Username", func(t *testing.T) {
		mockUserRepo.On("GetByUsername", mock.Anything, user.Username).Return(user, nil).Once()
		mockPassSvc.On("ComparePassword", user.Password, "password123").Return(nil).Once()
		mockJwtSvc.On("GenerateAccessToken", user.ID, user.Role).Return("valid.token", nil).Once()

		token, err := uc.Login(context.Background(), user.Username, "password123")
		assert.NoError(t, err)
		assert.Equal(t, "valid.token", token)
		mockUserRepo.AssertExpectations(t)

	})
}
