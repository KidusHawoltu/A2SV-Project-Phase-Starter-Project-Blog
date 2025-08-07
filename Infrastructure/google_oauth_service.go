package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	domain "A2SV_Starter_Project_Blog/Domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// httpClient defines the interface for an HTTP client. This allows us to
// use the standard http.Client in production and a mock client in tests.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// GoogleOAuthService is the concrete implementation. It's now exported
// so we can access its fields in the test file for mocking.
type GoogleOAuthService struct {
	OAuthConfig *oauth2.Config
	HTTPClient  httpClient // Use the interface type
	UserInfoURL string
}

// NewGoogleOAuthService is the constructor. It sets up the default, real dependencies.
func NewGoogleOAuthService(clientID, clientSecret, redirectURI string) (domain.IGoogleOAuthService, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Google OAuth client ID or secret is missing")
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	return &GoogleOAuthService{
		OAuthConfig: config,
		HTTPClient:  &http.Client{},                                  // Default production HTTP client
		UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo", // Default URL
	}, nil
}

// ExchangeCodeForToken remains a thin wrapper. We trust the underlying library.
func (s *GoogleOAuthService) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := s.OAuthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	return token, nil
}

// GetUserInfo is now testable because it uses the struct's fields for its dependencies.
func (s *GoogleOAuthService) GetUserInfo(ctx context.Context, token *oauth2.Token) (*domain.GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	// Use the injected HTTP client.
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned non-200 status for user info: %s", string(body))
	}

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info JSON: %w", err)
	}

	return &domain.GoogleUserInfo{
		ID:                userInfo.ID,
		Email:             userInfo.Email,
		Name:              userInfo.Name,
		ProfilePictureURL: userInfo.Picture,
	}, nil
}
