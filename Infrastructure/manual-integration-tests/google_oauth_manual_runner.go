//go:build manual
// +build manual

// This file is part of the infrastructure package, but it is not a standard test file.
// It is a runnable program designed for manually testing the Google OAuth2 flow.
//
// To run this test:
// 1. Ensure your .env file has the correct GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URI.
// 2. Execute the following command from the project root:
//    go run -tags=manual ./Infrastructure/manual-integration-tests/google_oauth_manual_runner.go
//
// The 'manual' build tag ensures this file is not included in your regular `go test` runs.

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"A2SV_Starter_Project_Blog/Infrastructure"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	// --- Setup ---
	if err := godotenv.Load("../.env"); err != nil {
		if err := godotenv.Load(".env"); err != nil {
			log.Fatal("Error loading .env file. Make sure it exists in the project root.")
		}
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")

	if clientID == "" || clientSecret == "" || redirectURI == "" {
		log.Fatal("Please set GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URI in your .env file.")
	}

	// We are testing the real infrastructure service here.
	service, err := infrastructure.NewGoogleOAuthService(clientID, clientSecret, redirectURI)
	if err != nil {
		log.Fatalf("Failed to create Google OAuth Service: %v", err)
	}

	// --- Step 1: Generate and Print the Auth URL ---
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
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Println("--- Manual Google OAuth2 Integration Test ---")
	fmt.Printf("\n1. Copy and paste this URL into your browser:\n\n%s\n\n", authURL)
	fmt.Println("2. Log in with your Google account and grant consent.")
	fmt.Println("3. You will be redirected to a 'This site can’t be reached' page. This is NORMAL.")
	fmt.Println("4. Copy the entire 'code' value from the URL in your browser's address bar.")
	fmt.Print("5. Paste the code here and press Enter: ")

	// --- Step 2: Read the Authorization Code from the User ---
	reader := bufio.NewReader(os.Stdin)
	pastedCode, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read code from stdin: %v", err)
	}
	pastedCode = strings.TrimSpace(pastedCode)
	if pastedCode == "" {
		log.Fatal("Authorization code cannot be empty.")
	}
	code, err := url.QueryUnescape(pastedCode)
	if err != nil {
		log.Fatalf("Failed to URL-decode the pasted code: %v", err)
	}
	// --- Step 3: Execute the Service Methods ---
	fmt.Println("\n--> Exchanging code for token...")
	ctx := context.Background()
	token, err := service.ExchangeCodeForToken(ctx, code)
	if err != nil {
		log.Fatalf("FAIL: ExchangeCodeForToken failed: %v", err)
	}
	fmt.Println("--> Token received successfully!")

	fmt.Println("--> Fetching user info...")
	userInfo, err := service.GetUserInfo(ctx, token)
	if err != nil {
		log.Fatalf("FAIL: GetUserInfo failed: %v", err)
	}
	fmt.Println("--> User info received successfully!")

	// --- Step 4: Print Results ---
	fmt.Println("\n----------------------------------")
	fmt.Println("✅ PASS: Test Succeeded!")
	fmt.Printf("   User ID: %s\n", userInfo.ID)
	fmt.Printf("      Name: %s\n", userInfo.Name)
	fmt.Printf("     Email: %s\n", userInfo.Email)
	fmt.Printf("   Picture: %s\n", userInfo.ProfilePictureURL)
	fmt.Println("----------------------------------")
}
