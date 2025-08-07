package usecases

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type AIUsecase struct {
	aiService      domain.IAIService
	contextTimeout time.Duration
}

func NewAIUsecase(aiService domain.IAIService, timeOut time.Duration) domain.IAIUsecase {
	return &AIUsecase{
		aiService:      aiService,
		contextTimeout: timeOut,
	}
}

func (ai *AIUsecase) GenerateBlogIdeas(ctx context.Context, keywords []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, ai.contextTimeout)
	defer cancel()

	// 1. Validate input
	if len(keywords) == 0 {
		return nil, domain.ErrValidation
	}

	// 2. Prompt Engineering: This is the core logic.
	// We give the AI a role, a task, the data, and crucially, a required output format.
	keywordString := `"` + strings.Join(keywords, `", "`) + `"`
	prompt := fmt.Sprintf(
		`You are an expert blogger and content strategist. 
		Your task is to generate 5 compelling and unique blog post titles.
		The titles must be based on the following keywords: %s.
		Return the result ONLY as a raw JSON array of strings, with no other text, commentary, or markdown formatting.
		Example response: ["Title 1", "Title 2", "Title 3", "Title 4", "Title 5"]`,
		keywordString,
	)

	// 3. Call the external AI service via our interface.
	aiResponse, err := ai.aiService.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// 4. Process the response.
	// We expect a raw JSON string, so we need to unmarshal it.
	var ideas []string
	// The AI might sometimes wrap the JSON in markdown code fences (```json ... ```), so we clean it.
	cleanedResponse := strings.Trim(aiResponse, " \n\t`json")
	if err := json.Unmarshal([]byte(cleanedResponse), &ideas); err != nil {
		// This error means the AI did not follow our output format instructions.
		log.Printf("Failed to unmarshal AI response. Raw response: %s", aiResponse)
		return nil, fmt.Errorf("%w: failed to parse AI response for blog ideas", ErrInternal)
	}

	return ideas, nil
}

func (ai *AIUsecase) RefineBlogPost(ctx context.Context, content string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, ai.contextTimeout)
	defer cancel()

	// 1. Validate input
	if strings.TrimSpace(content) == "" {
		return "", domain.ErrValidation
	}

	// 2. Prompt Engineering: Give the AI a clear role and set of instructions.
	prompt := fmt.Sprintf(
		`You are an expert copy editor and writer for a blog.
		Your task is to refine the following blog post content. Your goals are to:
		1. Improve clarity and readability.
		2. Fix any spelling and grammatical errors.
		3. Enhance the tone to be more professional and engaging.
		4. Ensure technical accuracy where possible, without adding new information.
		Do not add new sections or radically change the original meaning.
		Return ONLY the refined text, with no other commentary, explanations, or markdown formatting.

		Original content to refine:
		---
		%s`,
		content,
	)

	// 3. Call the external AI service.
	refinedContent, err := ai.aiService.GenerateCompletion(ctx, prompt)
	if err != nil {
		return "", err
	}

	// 4. Process the response.
	// Here, we expect the raw string back, so we just need to clean up any extra whitespace.
	return strings.TrimSpace(refinedContent), nil
}
