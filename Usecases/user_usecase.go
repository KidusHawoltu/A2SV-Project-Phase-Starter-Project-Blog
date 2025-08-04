package usecases

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	"context"
	"net/mail"
	"time"
)

// UserUsecase defines the business logic required for Phase 1 & 2.

// UserRepository defines the persistence operations for a User.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	FindUserIDsByName(ctx context.Context, authorName string) ([]string, error)
}
type UserUsecase interface {
	Register(ctx context.Context, user *domain.User) error
	Login(ctx context.Context, identifier, password string) (accessToken string, err error)
}

type userUsecase struct {
	userRepo        UserRepository
	passwordService infrastructure.PasswordService
	jwtService      infrastructure.JWTService
	contextTimeout  time.Duration
}

func NewUserUsecase(ur UserRepository, ps infrastructure.PasswordService, js infrastructure.JWTService, timeout time.Duration) UserUsecase {
	return &userUsecase{
		userRepo:        ur,
		passwordService: ps,
		jwtService:      js,
		contextTimeout:  timeout,
	}
}

func (uc *userUsecase) Register(c context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	if err := user.Validate(); err != nil {
		return err
	}

	existingUser, _ := uc.userRepo.GetByEmail(ctx, user.Email)
	if existingUser != nil {
		return domain.ErrEmailExists
	}

	hashedPassword, err := uc.passwordService.HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashedPassword
	user.Role = domain.RoleUser

	return uc.userRepo.Create(ctx, user)
}

func (uc *userUsecase) Login(c context.Context, identifier, password string) (string, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	var user *domain.User
	var err error

	if _, mailErr := mail.ParseAddress(identifier); mailErr == nil {
		user, err = uc.userRepo.GetByEmail(ctx, identifier)
	} else {
		user, err = uc.userRepo.GetByUsername(ctx, identifier)
	}

	if err != nil {
		return "", err
	}
	if user == nil {
		return "", domain.ErrAuthenticationFailed //user not found
	}

	// Generate and return access token
	return uc.jwtService.GenerateAccessToken(user.ID, user.Role)
}
