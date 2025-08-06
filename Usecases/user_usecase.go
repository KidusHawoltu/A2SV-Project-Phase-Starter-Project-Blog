package usecases

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"
	"context"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

// UserUsecase defines the business logic required for Phase 1 & 2.

// UserRepository defines the persistence operations for a User.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
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
	UpdateProfile(ctx context.Context, userID, bio, profilePicURL string) (*domain.User, error)
	GetProfile(c context.Context, userID string) (*domain.User, error)
}

type userUsecase struct {
	userRepo        UserRepository
	tokenRepo       TokenRepository
	passwordService infrastructure.PasswordService
	jwtService      infrastructure.JWTService
	emailService    infrastructure.EmailService
	contextTimeout  time.Duration
}

func NewUserUsecase(ur UserRepository, ps infrastructure.PasswordService, js infrastructure.JWTService, tr TokenRepository,es infrastructure.EmailService, timeout time.Duration) UserUsecase {
	return &userUsecase{
		userRepo:        ur,
		tokenRepo:       tr,
		passwordService: ps,
		jwtService:      js,
		emailService:    es,
		contextTimeout:  timeout,
	}
}

func (uc *userUsecase) Register(c context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	if err := user.Validate(); err != nil { return err }

	existingUser, _ := uc.userRepo.GetByEmail(ctx, user.Email)
	if existingUser != nil { return domain.ErrEmailExists }

	// Also check for username uniqueness
	existingUser, _ = uc.userRepo.GetByUsername(ctx, user.Username)
	if existingUser != nil {
		return domain.ErrUsernameExists
	}

	hashedPassword, err := uc.passwordService.HashPassword(user.Password)
	if err != nil { return err }
	user.Password = hashedPassword
	user.Role = domain.RoleUser
	user.IsActive = false

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return err
	}

	//After successful creation, generate and send activation token.
	activationToken := &domain.Token{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Type:      domain.TokenTypeActivation,
		Value:     uuid.NewString(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Activation links can last longer
	}

	if err := uc.tokenRepo.Store(ctx, activationToken); err != nil {
		return err
	}

	return uc.emailService.SendActivationEmail(user.Email, user.Username, activationToken.Value)
}

func(uc *userUsecase) ActivateAccount(c context.Context, activationTokenValue string) error {
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

	// Update the user's status.
	user.IsActive = true
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}

	return uc.tokenRepo.Delete(ctx, activationToken.ID)
}

// login is updated to handle refresh token
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
		return "","", err
	}
	if user == nil {
		return  "", "", domain.ErrAuthenticationFailed //user not found
	}

	if !user.IsActive {
		return "","", domain.ErrAccountNotActive
	}

	err = uc.passwordService.ComparePassword(user.Password, password) 
	if err != nil {
		return "","",domain.ErrAuthenticationFailed
	}

	// Generate and return access token
	accessTokenString, accessClaims, err := uc.jwtService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return "","", err
	}
	refreshTokenString, refreshClaims, err := uc.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return "","", err
	}

	accessToken := &domain.Token{
		ID: accessClaims.ID,
		UserID: user.ID,
		Type: domain.TokenTypeAccessToken,
		Value: accessTokenString,
		ExpiresAt: accessClaims.ExpiresAt.Time,
	}

	if err := uc.tokenRepo.Store(ctx, accessToken); err != nil {
		return "", "", err
	}

	refreshToken := &domain.Token{
		ID: refreshClaims.ID,
		UserID: user.ID,
		Type: domain.TokenTypeRefresh,
		Value: refreshTokenString,
		ExpiresAt:refreshClaims.ExpiresAt.Time,
	}
	if err := uc.tokenRepo.Store(ctx, refreshToken); err != nil {
		return "","",err
	}

	return accessTokenString, refreshTokenString, nil

}

// Logout invalidates a session by deleting the refresh token.
func (uc *userUsecase) Logout(c context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	token, err := uc.tokenRepo.GetByValue(ctx, refreshToken)
	if err != nil || token == nil || token.Type != domain.TokenTypeRefresh {
		return nil  // Do not leak info. Act as if logout was successful.
	}

	return uc.tokenRepo.Delete(ctx, token.ID)
}

// RefreshAccessToken issues a new access token from a valid refresh token.
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
		return "","",domain.ErrAuthenticationFailed
	}

	if dbRefreshToken.UserID != AccessClaims.UserID || dbAccessToken.UserID != dbRefreshToken.UserID {
		return "", "", domain.ErrAuthenticationFailed // mismatched user IDs.
	}

	//Everything checks out. Invalidate the old tokens.
	uc.tokenRepo.Delete(ctx, dbAccessToken.ID)
	uc.tokenRepo.Delete(ctx, dbRefreshToken.ID)

	//Generate and store a new pair of tokens.
	user, err := uc.userRepo.GetByID(ctx, dbRefreshToken.UserID)
	if err != nil || user == nil {
		return "", "", domain.ErrUserNotFound
	}

	return uc.generateAndStoreTokenPair(ctx, user)
}

// ForgotPassword generates a reset token and sends it via email.
func (uc *userUsecase) ForgetPassword(c context.Context, email string) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	user, err:= uc.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil{
		return nil  // Prevent email enumeration attacks.
	}

	// Delete previous reset tokens for this user.
	if err := uc.tokenRepo.DeleteByUserID(ctx, user.ID,domain.TokenTypePasswordReset); err != nil {
		return err
	}
	resetToken := &domain.Token{
		ID: uuid.NewString(),
		UserID: user.ID,
		Type: domain.TokenTypePasswordReset,
		Value: uuid.NewString(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := uc.tokenRepo.Store(ctx, resetToken); err != nil {
		return err
	}

	return uc.emailService.SendPasswordResetEmail(user.Email, user.Username, resetToken.ID)
}

// ResetPassword validates the token and updates the password.
func (uc *userUsecase) ResetPassword(c context.Context, resetTokenValue, newPassword string) error {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	resetToken, err := uc.tokenRepo.GetByValue(ctx, resetTokenValue)
	if err != nil || resetToken == nil || resetToken.Type!= domain.TokenTypePasswordReset || resetToken.IsExpired() {
		return domain.ErrInvalidResetToken
	}

	user, err := uc.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil || user == nil {
		return domain.ErrUserNotFound
	}
	if len(newPassword) < 8 {
		return domain.ErrPasswordTooShort
	}

	hashedPassword, err := uc.passwordService.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}
	// Invalidate the token after use.
	return uc.tokenRepo.Delete(ctx, resetToken.ID)
}

// UpdateProfile allows a user to update their own bio and profile picture URL.
func (uc *userUsecase) UpdateProfile(c context.Context, userID, bio, profilePicURL string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(c, uc.contextTimeout)
	defer cancel()

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, domain.ErrUserNotFound
	}

	user.Bio = bio
	user.ProfilePicture = profilePicURL
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

// helper
func (uc *userUsecase) generateAndStoreTokenPair(ctx context.Context, user *domain.User) (string, string, error) {
	// Access token
	accessToken, accessClaims, err := uc.jwtService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return "", "", err
	}

	accessTokenModel := &domain.Token {
		ID: accessClaims.ID,
		UserID: user.ID,
		Type: domain.TokenTypeAccessToken,
		Value: accessToken,
		ExpiresAt: accessClaims.ExpiresAt.Time,
	}

	if err := uc.tokenRepo.Store(ctx,accessTokenModel); err != nil {
		return "", "", err
	}

	// Refresh token
	refreshToken, refreshClaims, err := uc.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		return "","", err
	}
	refreshTokenModel := &domain.Token {
		ID: refreshClaims.ID,
		UserID: user.ID,
		Type: domain.TokenTypeRefresh,
		Value: refreshToken,
		ExpiresAt: refreshClaims.ExpiresAt.Time,
	}

	if err := uc.tokenRepo.Store(ctx, refreshTokenModel); err != nil {
		return "","", err
	}

	return accessToken, refreshToken, nil
}

