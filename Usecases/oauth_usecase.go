package usecases

import (
	"context"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	infrastructure "A2SV_Starter_Project_Blog/Infrastructure"

	"github.com/google/uuid"
)

// oauthUsecase implements the domain.IOAuthUsecase interface.
type oauthUsecase struct {
	userRepo   UserRepository
	tokenRepo  TokenRepository
	jwtService infrastructure.JWTService
	googleSvc  domain.IGoogleOAuthService
	timeout    time.Duration
}

// NewOAuthUsecase is the constructor for the OAuth usecase.
func NewOAuthUsecase(
	userRepo UserRepository,
	tokenRepo TokenRepository,
	jwtService infrastructure.JWTService,
	googleSvc domain.IGoogleOAuthService,
	timeout time.Duration,
) domain.IOAuthUsecase {
	return &oauthUsecase{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtService: jwtService,
		googleSvc:  googleSvc,
		timeout:    timeout,
	}
}

// HandleGoogleCallback orchestrates the entire Google OAuth2 flow.
func (uc *oauthUsecase) HandleGoogleCallback(c context.Context, code string) (string, string, error) {
	ctx, cancel := context.WithTimeout(c, uc.timeout)
	defer cancel()

	// 1. Exchange the authorization code for an OAuth2 token from Google.
	googleToken, err := uc.googleSvc.ExchangeCodeForToken(ctx, code)
	if err != nil {
		return "", "", err
	}

	// 2. Use the token to get the user's information from Google.
	userInfo, err := uc.googleSvc.GetUserInfo(ctx, googleToken)
	if err != nil {
		return "", "", err
	}

	// 3. The "Find or Create" logic begins. First, check if a user with this Google ID already exists.
	user, err := uc.userRepo.FindByProviderID(ctx, domain.ProviderGoogle, userInfo.ID)
	if err != nil {
		// This is a real DB error, not "not found".
		return "", "", err
	}

	// Scenario A: The Google user already exists in our system (Sign In).
	if user != nil {
		if !user.IsActive {
			return "", "", domain.ErrAccountNotActive
		}
		// The user is valid, proceed to generate our own tokens.
		return uc.generateAndStoreTokenPair(ctx, user)
	}

	// Scenario B: No user with this Google ID. We need to check by email to link accounts or create a new one.
	user, err = uc.userRepo.GetByEmail(ctx, userInfo.Email)
	if err != nil {
		return "", "", err
	}

	// Scenario B1: A user with this email exists but uses local auth.
	// This is an account linking scenario. For now, we'll treat it as an error
	// to prevent security issues. A more advanced implementation could handle linking.
	if user != nil && user.Provider == domain.ProviderLocal {
		return "", "", domain.ErrEmailExists // Or a more specific "please link your account" error.
	}

	// Scenario B2: The user is truly new (Sign Up).
	if user == nil {
		newUser := &domain.User{
			Username:       userInfo.Name, // Or generate a unique username
			Email:          userInfo.Email,
			Password:       nil,  // No password for OAuth users
			IsActive:       true, // Accounts created via Google are active by default
			Role:           domain.RoleUser,
			ProfilePicture: userInfo.ProfilePictureURL,
			Provider:       domain.ProviderGoogle,
			ProviderID:     userInfo.ID,
		}

		// Before creating, ensure the generated username is unique.
		// A simple strategy is to append a few random digits if it's not.
		// (This logic can be expanded).
		existingUser, _ := uc.userRepo.GetByUsername(ctx, newUser.Username)
		if existingUser != nil {
			newUser.Username = newUser.Username + "-" + uuid.NewString()[:4]
		}

		if err := uc.userRepo.Create(ctx, newUser); err != nil {
			return "", "", err
		}

		// The new user has been created, proceed to generate tokens.
		return uc.generateAndStoreTokenPair(ctx, newUser)
	}

	// This case should ideally not be reached if the user with the email
	// is already a Google user, as they would have been found by provider ID.
	// But as a fallback, we treat it as a successful login.
	return uc.generateAndStoreTokenPair(ctx, user)
}

// generateAndStoreTokenPair is a helper to avoid duplicating token generation logic.
// This can be the same helper from your userUsecase.
func (uc *oauthUsecase) generateAndStoreTokenPair(ctx context.Context, user *domain.User) (string, string, error) {
	// Access token
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

	// Refresh token
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
