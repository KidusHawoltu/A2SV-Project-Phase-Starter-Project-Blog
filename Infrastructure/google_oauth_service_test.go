package infrastructure_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "A2SV_Starter_Project_Blog/Infrastructure"

	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type GoogleOAuthServiceUnitTestSuite struct {
	suite.Suite
}

// TestGoogleOAuthServiceUnitTestSuite is the entry point for the test suite.
func TestGoogleOAuthServiceUnitTestSuite(t *testing.T) {
	suite.Run(t, new(GoogleOAuthServiceUnitTestSuite))
}

// TestNewGoogleOAuthService_Failures tests the constructor's error handling.
func (s *GoogleOAuthServiceUnitTestSuite) TestNewGoogleOAuthService_Failures() {
	s.Run("Failure - Empty Client ID", func() {
		service, err := NewGoogleOAuthService("", "secret", "uri")
		s.Error(err)
		s.Nil(service)
	})
	s.Run("Failure - Empty Client Secret", func() {
		service, err := NewGoogleOAuthService("id", "", "uri")
		s.Error(err)
		s.Nil(service)
	})
}

func (s *GoogleOAuthServiceUnitTestSuite) TestGetUserInfo() {
	s.Run("Success - Valid response from server", func() {
		// Arrange:
		// 1. Create a mock HTTP server that will act as the Google UserInfo endpoint.
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Assert that the request sent by our service is correct.
			s.Equal("GET", r.Method)
			s.Equal("Bearer test-access-token", r.Header.Get("Authorization"))

			// Send back a valid JSON response.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"id":"12345","email":"test@example.com","name":"Test User","picture":"http://pic.url"}`)
		}))
		defer mockServer.Close()

		// 2. Create an instance of our service.
		service, err := NewGoogleOAuthService("id", "secret", "uri")
		s.Require().NoError(err)

		// 3. Inject our mock dependencies into the service instance.
		// We type-assert to the concrete struct to access its fields.
		concreteService := service.(*GoogleOAuthService)
		concreteService.UserInfoURL = mockServer.URL     // Point to our mock server
		concreteService.HTTPClient = mockServer.Client() // Use the mock server's client

		// 4. Create a dummy token for the test.
		token := &oauth2.Token{AccessToken: "test-access-token"}

		// Act
		userInfo, err := service.GetUserInfo(context.Background(), token)

		// Assert
		s.NoError(err)
		s.NotNil(userInfo)
		s.Equal("12345", userInfo.ID)
		s.Equal("test@example.com", userInfo.Email)
		s.Equal("Test User", userInfo.Name)
		s.Equal("http://pic.url", userInfo.ProfilePictureURL)
	})

	s.Run("Failure - Server returns non-200 status", func() {
		// Arrange:
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate an error from Google's API.
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, `{"error": "invalid_token"}`)
		}))
		defer mockServer.Close()

		service, _ := NewGoogleOAuthService("id", "secret", "uri")
		concreteService := service.(*GoogleOAuthService)
		concreteService.UserInfoURL = mockServer.URL
		concreteService.HTTPClient = mockServer.Client()

		token := &oauth2.Token{AccessToken: "expired-token"}

		// Act
		userInfo, err := service.GetUserInfo(context.Background(), token)

		// Assert
		s.Error(err, "Should return an error when status code is not 200")
		s.Nil(userInfo)
		s.Contains(err.Error(), "invalid_token", "Error message should contain the body from the server")
	})

	s.Run("Failure - Server returns malformed JSON", func() {
		// Arrange:
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `this is not valid json`) // Send back garbage
		}))
		defer mockServer.Close()

		service, _ := NewGoogleOAuthService("id", "secret", "uri")
		concreteService := service.(*GoogleOAuthService)
		concreteService.UserInfoURL = mockServer.URL
		concreteService.HTTPClient = mockServer.Client()

		token := &oauth2.Token{AccessToken: "any-token"}

		// Act
		userInfo, err := service.GetUserInfo(context.Background(), token)

		// Assert
		s.Error(err, "Should return an error when JSON unmarshalling fails")
		s.Nil(userInfo)
		s.Contains(err.Error(), "failed to unmarshal user info JSON")
	})
}
