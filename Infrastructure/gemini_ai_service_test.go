package infrastructure_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Infrastructure"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/suite"
)

type GeminiAIServiceSuite struct {
	suite.Suite
	apiKey    string
	modelNmae string
	service   domain.IAIService
}

func (s *GeminiAIServiceSuite) SetupSuite() {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		s.T().Skip("Skipping GeminiAIService integration tests: OPENAI_API_KEY not set.")
	}

	s.apiKey = key
	s.modelNmae = os.Getenv("GEMINI_MODEL")
	if s.modelNmae == "" {
		s.modelNmae = "gemini-2.5-pro"
	}
}

func (s *GeminiAIServiceSuite) SetupTest() {
	service, err := NewGeminiAIService(s.apiKey, s.modelNmae)
	s.Require().NoError(err, "Failed to create a new Gemini AI Service instance")
	s.service = service
}

// TestGeminiAIServiceSuite is the entry point for the test suite.
func TestGeminiAIServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}
	suite.Run(t, new(GeminiAIServiceSuite))
}

// --- Tests ---

func (s *GeminiAIServiceSuite) TestGenerateCompletion() {
	ctx := context.Background()

	s.Run("Success - Simple Factual Prompt", func() {
		// Arrange: A simple prompt with a predictable answer.
		prompt := "What is the capital of France?"

		// Act: Call the real Gemini API.
		response, err := s.service.GenerateCompletion(ctx, prompt)

		// Assert
		s.NoError(err, "The API call should not return an error")
		s.NotEmpty(response, "The response from the API should not be empty")
		s.Contains(strings.ToLower(response), "paris", "The response should contain the correct answer")
	})

	s.Run("Success - Structured JSON Prompt", func() {
		// Arrange
		prompt := `Return a single JSON object with one key "country" and the value "Japan". Do not include any other text or markdown formatting.`

		// Act
		response, err := s.service.GenerateCompletion(ctx, prompt)

		// Assert
		s.NoError(err)
		s.NotEmpty(response)

		// The most important assertion: can we unmarshal the response?
		var data struct {
			Country string `json:"country"`
		}
		// Clean the response just in case the AI adds markdown fences.
		cleanedResponse := strings.Trim(response, " \n\t`json")
		unmarshalErr := json.Unmarshal([]byte(cleanedResponse), &data)

		s.NoError(unmarshalErr, "The AI response should be valid JSON that can be unmarshalled")
		s.Equal("Japan", data.Country, "The unmarshalled data should contain the correct value")
	})
}

// TestNewGeminiAIService_Failures tests the constructor's error handling.
func (s *GeminiAIServiceSuite) TestNewGeminiAIService_Failures() {
	s.Run("Failure - Empty API Key", func() {
		// Act
		service, err := NewGeminiAIService("", "")

		// Assert
		s.Error(err, "Constructor should return an error for an empty API key")
		s.Nil(service, "Service instance should be nil on failure")
	})
}
