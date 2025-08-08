package usecases

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	"context"
	"mime/multipart"
	"mime/multipart"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

// UserRepository defines the persistence operations for a User.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	FindUserIDsByName(ctx context.Context, authorName string) ([]string, error)
	FindByProviderID(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error)
	SearchAndFilter(ctx context.Context, options domain.UserSearchFilterOptions) ([]*domain.User, int64, error)
}

type TokenRepository interface {
	Store(ctx context.Context, token *domain.Token) error
	GetByValue(ctx context.Context, tokenValue string) (*domain.Token, error)
	Delete(ctx context.Context, tokenID string) error
	DeleteByUserID(ctx context.Context, userID string, tokenType domain.TokenType) error
}

type UserUsecase interface {
	Register(ctx context.Context, user *domain.User) error
	ActivateAccount(ctx context.Context, activationTokenValue string) error
	Login(ctx context.Context, identifier, password string) (accessToken, refreshToken string, err error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshAccessToken(ctx context.Context, refreshToken, accessToken string) (newAccessToken, newRefreshToken string, err error)

	// Password Management
	ForgetPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, resetToken, newPassword string) error

	//Profile Management
	UpdateProfile(c context.Context, userID, bio string, profilePicFile multipart.File, profilePicHeader *multipart.FileHeader) (*domain.User, error)
	UpdateProfile(c context.Context, userID, bio string, profilePicFile multipart.File, profilePicHeader *multipart.FileHeader) (*domain.User, error)
	GetProfile(c context.Context, userID string) (*domain.User, error)

	// User Management
	SearchAndFilter(ctx context.Context, options domain.UserSearchFilterOptions) ([]*domain.User, int64, error)
	SetUserRole(ctx context.Context, actorUserID string, actorRole domain.Role, targetUserID string, newRole domain.Role) (*domain.User, error)
}

type userUsecase struct {
	userRepo             UserRepository
	tokenRepo            TokenRepository
	passwordService      infrastructure.PasswordService
	jwtService           infrastructure.JWTService
	emailService         infrastructure.EmailService
	imageUploaderService domain.ImageUploaderService
	contextTimeout       time.Duration
}

func NewUserUsecase(ur UserRepository, ps infrastructure.PasswordService, js infrastructure.JWTService, tr TokenRepository, es infrastructure.EmailService, ius domain.ImageUploaderService, timeout time.Duration) UserUsecase {
	return &userUsecase{
		userRepo:             ur,
		tokenRepo:            tr,
		passwordService:      ps,
		jwtService:           js,
		emailService:         es,
		imageUploaderService: ius,
		contextTimeout:       timeout,
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

	existingUser, _ = uc.userRepo.GetByUsername(ctx, user.Username)
	if existingUser != nil {
		return domain.ErrUsernameExists
	}

	hashedPassword, err := uc.passwordService.HashPassword(*(user.Password))
	if err != nil {
		return err
	}
	user.Password = &hashedPassword
	user.Role = domain.RoleUser
	user.IsActive = false
	user.Provider = domain.ProviderLocal

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return err
	}

	activationToken := &domain.Token{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Type:      domain.TokenTypeActivation,
		Value:     uuid.NewString(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := uc.tokenRepo.Store(ctx, activationToken); err != nil {
		return err
	}

	return uc.emailService.SendActivationEmail(user.Email, user.Username, activationToken.Value)
}

func (uc *userUsecase) ActivateAccount(c context.Context, activationTokenValue string) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	activationToken, err := uc.tokenRepo.GetByValue(ctx, activationTokenValue)
	if err != nil || activationToken == nil || activationToken.IsExpired() || activationToken.Type != domain.TokenTypeActivation {
		return domain.ErrInvalidActivationToken
	}

	user, err := uc.userRepo.GetByID(ctx, activationToken.UserID)
	if err != nil || user == nil {
		return domain.ErrUserNotFound
	}

	user.IsActive = true
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}

	return uc.tokenRepo.Delete(ctx, activationToken.ID)
}

func (uc *userUsecase) Login(c context.Context, identifier, password string) (string, string, error) {
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
		return "", "", err
	}
	if user == nil {
		return "", "", domain.ErrAuthenticationFailed
	}

	if user.Provider != domain.ProviderLocal {
		return "", "", domain.ErrOAuthUser
	}

	if !user.IsActive {
		return "", "", domain.ErrAccountNotActive
	}

	err = uc.passwordService.ComparePassword(*(user.Password), password)
	if err != nil {
		return "", "", domain.ErrAuthenticationFailed
	}

	return uc.generateAndStoreTokenPair(ctx, user)
}

func (uc *userUsecase) Logout(c context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	token, err := uc.tokenRepo.GetByValue(ctx, refreshToken)
	if err != nil || token == nil || token.Type != domain.TokenTypeRefresh {
		return nil
	}

	return uc.tokenRepo.Delete(ctx, token.ID)
}

func (uc *userUsecase) RefreshAccessToken(c context.Context, refreshToken, accessToken string) (string, string, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	dbRefreshToken, err := uc.tokenRepo.GetByValue(ctx, refreshToken)
	if err != nil || dbRefreshToken == nil || dbRefreshToken.IsExpired() || dbRefreshToken.Type != domain.TokenTypeRefresh {
		return "", "", domain.ErrAuthenticationFailed
	}

	AccessClaims, err := uc.jwtService.ParseExpiredToken(accessToken)
	if err != nil || AccessClaims == nil {
		return "", "", domain.ErrAuthenticationFailed
	}

	dbAccessToken, err := uc.tokenRepo.GetByValue(ctx, accessToken)
	if err != nil || dbAccessToken == nil {
		uc.tokenRepo.DeleteByUserID(ctx, dbRefreshToken.UserID, domain.TokenTypeRefresh)
		uc.tokenRepo.DeleteByUserID(ctx, dbAccessToken.UserID, domain.TokenTypeAccessToken)
		return "", "", domain.ErrAuthenticationFailed
	}

	if dbRefreshToken.UserID != AccessClaims.UserID || dbAccessToken.UserID != dbRefreshToken.UserID {
		return "", "", domain.ErrAuthenticationFailed
	}

	uc.tokenRepo.Delete(ctx, dbAccessToken.ID)
	uc.tokenRepo.Delete(ctx, dbRefreshToken.ID)

	user, err := uc.userRepo.GetByID(ctx, dbRefreshToken.UserID)
	if err != nil || user == nil {
		return "", "", domain.ErrUserNotFound
	}

	return uc.generateAndStoreTokenPair(ctx, user)
}

func (uc *userUsecase) ForgetPassword(c context.Context, email string) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return nil
	}

	if user.Provider != domain.ProviderLocal {
		return nil
	}

	if err := uc.tokenRepo.DeleteByUserID(ctx, user.ID, domain.TokenTypePasswordReset); err != nil {
		return err
	}
	resetToken := &domain.Token{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Type:      domain.TokenTypePasswordReset,
		Value:     uuid.NewString(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := uc.tokenRepo.Store(ctx, resetToken); err != nil {
		return err
	}

	return uc.emailService.SendPasswordResetEmail(user.Email, user.Username, resetToken.Value)
}

func (uc *userUsecase) ResetPassword(c context.Context, resetTokenValue, newPassword string) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	resetToken, err := uc.tokenRepo.GetByValue(ctx, resetTokenValue)
	if err != nil || resetToken == nil || resetToken.Type != domain.TokenTypePasswordReset || resetToken.IsExpired() {
		return domain.ErrInvalidResetToken
	}

	user, err := uc.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil || user == nil {
		return domain.ErrUserNotFound
	}
	if user.Provider != domain.ProviderLocal {
		return domain.ErrOAuthUser
	}
	if len(newPassword) < 8 {
		return domain.ErrPasswordTooShort
	}

	hashedPassword, err := uc.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = &hashedPassword
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}
	return uc.tokenRepo.Delete(ctx, resetToken.ID)
}

func (uc *userUsecase) UpdateProfile(c context.Context, userID, bio string, profilePicFile multipart.File, profilePicHeader *multipart.FileHeader) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound
	}

	user.Bio = bio

	if profilePicFile != nil {
		imageURL, err := uc.imageUploaderService.UploadProfilePicture(profilePicFile, profilePicHeader)
		if err != nil {
			return nil, err
		}
		user.ProfilePicture = imageURL
	}
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *userUsecase) GetProfile(c context.Context, userID string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

func (uc *userUsecase) generateAndStoreTokenPair(ctx context.Context, user *domain.User) (string, string, error) {
	accessToken, accessClaims, err := uc.jwtService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return "", "", err
	}

	accessTokenModel := &domain.Token{
		ID:        accessClaims.ID,
		UserID:    user.ID,
		Type:      domain.TokenTypeAccessToken,
		Value:     accessToken,
		ExpiresAt: accessClaims.ExpiresAt.Time,
	}

	if err := uc.tokenRepo.Store(ctx, accessTokenModel); err != nil {
		return "", "", err
	}

	refreshToken, refreshClaims, err := uc.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}
	refreshTokenModel := &domain.Token{
		ID:        refreshClaims.ID,
		UserID:    user.ID,
		Type:      domain.TokenTypeRefresh,
		Value:     refreshToken,
		ExpiresAt: refreshClaims.ExpiresAt.Time,
	}

	if err := uc.tokenRepo.Store(ctx, refreshTokenModel); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (uc *userUsecase) SearchAndFilter(c context.Context, options domain.UserSearchFilterOptions) ([]*domain.User, int64, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	if options.Page <= 0 {
		options.Page = 1
	}
	if options.Limit <= 0 {
		options.Limit = 10
	}
	if options.Limit > 100 {
		options.Limit = 100
	}

	return uc.userRepo.SearchAndFilter(ctx, options)
}

func (uc *userUsecase) SetUserRole(ctx context.Context, actorUserID string, actorRole domain.Role, targetUserID string, newRole domain.Role) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, uc.contextTimeout)
	defer cancel()

	if actorRole != domain.RoleAdmin {
		return nil, domain.ErrPermissionDenied
	}

	if actorUserID == targetUserID {
		return nil, domain.ErrCannotChangeOwnRole
	}

	if !newRole.IsValid() {
		return nil, domain.ErrInvalidRole
	}

	targetUser, err := uc.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	if targetUser.Role == newRole {
		return targetUser, nil
	}

	targetUser.Role = newRole
	targetUser.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, targetUser); err != nil {
		return nil, err
	}

	return targetUser, nil
}
