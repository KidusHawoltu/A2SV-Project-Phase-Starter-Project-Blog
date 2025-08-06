package infrastructure

import (
	"context"
	"fmt"
	"log"

	domain "A2SV_Starter_Project_Blog/Domain"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiAIService struct {
	model *genai.GenerativeModel
}

func NewGeminiAIService(apiKey, modelName string) (domain.IAIService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is missing")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create new genai client: %w", err)
	}

	model := client.GenerativeModel(modelName)

	return &GeminiAIService{
		model: model,
	}, nil
}

// GenerateCompletion sends a prompt to the Gemini API and returns the text response.
func (s *GeminiAIService) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	// 1. Send the prompt to the model.
	resp, err := s.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content from Gemini: %w", err)
	}

	// 2. Process the response to extract the text.
	// A valid response should have at least one "candidate" (possible answer).
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Println("Gemini response was empty or had no content parts.")
		return "", fmt.Errorf("received an empty response from the AI service")
	}

	// 3. Extract the text from the first part of the first candidate.
	// We are expecting a simple text response.
	firstPart := resp.Candidates[0].Content.Parts[0]
	if text, ok := firstPart.(genai.Text); ok {
		return string(text), nil
	}

	// This is a fallback case if the AI returns a non-text part unexpectedly.
	log.Printf("Gemini returned a non-text part: %T", firstPart)
	return "", fmt.Errorf("received an unexpected response type from the AI service")
}
