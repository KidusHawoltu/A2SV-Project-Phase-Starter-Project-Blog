package usecases_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Usecases"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock IAIService ---
// This mock allows us to control the behavior of the external AI service during tests.
type MockAIService struct {
	mock.Mock
}

func (m *MockAIService) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

// --- Test Suite Setup ---
type AIUsecaseTestSuite struct {
	suite.Suite
	mockAIService *MockAIService
	usecase       domain.IAIUsecase
}

func (s *AIUsecaseTestSuite) SetupTest() {
	s.mockAIService = new(MockAIService)
	s.usecase = NewAIUsecase(s.mockAIService, 45*time.Second)
}

func TestAIUsecaseTestSuite(t *testing.T) {
	suite.Run(t, new(AIUsecaseTestSuite))
}

// --- Tests for GenerateBlogIdeas ---

func (s *AIUsecaseTestSuite) TestGenerateBlogIdeas() {
	ctx := context.Background()
	keywords := []string{"go", "rest", "api"}

	s.Run("Success - Valid JSON response", func() {
		s.SetupTest()
		// Arrange:
		// 1. Define the exact prompt we expect the usecase to construct.
		keywordString := `"go", "rest", "api"`
		expectedPrompt := fmt.Sprintf(
			`You are an expert blogger and content strategist. 
		Your task is to generate 5 compelling and unique blog post titles.
		The titles must be based on the following keywords: %s.
		Return the result ONLY as a raw JSON array of strings, with no other text, commentary, or markdown formatting.
		Example response: ["Title 1", "Title 2", "Title 3", "Title 4", "Title 5"]`,
			keywordString,
		)

		// 2. Define the ideal response from the AI.
		aiResponseJSON := `["Building REST APIs in Go", "Go API Best Practices"]`
		expectedIdeas := []string{"Building REST APIs in Go", "Go API Best Practices"}

		// 3. Set the mock expectation.
		s.mockAIService.On("GenerateCompletion", mock.Anything, expectedPrompt).Return(aiResponseJSON, nil).Once()

		// Act
		ideas, err := s.usecase.GenerateBlogIdeas(ctx, keywords)

		// Assert
		s.NoError(err)
		s.Equal(expectedIdeas, ideas)
		s.mockAIService.AssertExpectations(s.T())
	})

	s.Run("Success - Handles markdown in JSON response", func() {
		s.SetupTest()
		// Arrange:
		// Mock the AI returning a valid JSON string wrapped in markdown code fences.
		aiResponseMarkdown := "```json\n[\"Title A\", \"Title B\"]\n```"
		expectedIdeas := []string{"Title A", "Title B"}

		s.mockAIService.On("GenerateCompletion", mock.Anything, mock.Anything).Return(aiResponseMarkdown, nil).Once()

		// Act
		ideas, err := s.usecase.GenerateBlogIdeas(ctx, keywords)

		// Assert
		s.NoError(err)
		s.Equal(expectedIdeas, ideas)
	})

	s.Run("Failure - AI service returns an error", func() {
		s.SetupTest()
		// Arrange
		expectedErr := errors.New("API rate limit exceeded")
		s.mockAIService.On("GenerateCompletion", mock.Anything, mock.Anything).Return("", expectedErr).Once()

		// Act
		ideas, err := s.usecase.GenerateBlogIdeas(ctx, keywords)

		// Assert
		s.Error(err)
		s.ErrorIs(err, expectedErr)
		s.Nil(ideas)
		s.mockAIService.AssertExpectations(s.T())
	})

	s.Run("Failure - AI returns malformed JSON", func() {
		s.SetupTest()
		// Arrange
		malformedJSON := `This is not JSON.`
		s.mockAIService.On("GenerateCompletion", mock.Anything, mock.Anything).Return(malformedJSON, nil).Once()

		// Act
		ideas, err := s.usecase.GenerateBlogIdeas(ctx, keywords)

		// Assert
		s.Error(err)
		s.ErrorIs(err, ErrInternal, "Should return a generic internal error")
		s.Nil(ideas)
		s.mockAIService.AssertExpectations(s.T())
	})

	s.Run("Failure - No keywords provided", func() {
		s.SetupTest()
		// Act
		ideas, err := s.usecase.GenerateBlogIdeas(ctx, []string{})

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrValidation)
		s.Nil(ideas)
		// Crucially, the AI service should not have been called.
		s.mockAIService.AssertNotCalled(s.T(), "GenerateCompletion", mock.Anything, mock.Anything)
	})
}

// --- Tests for RefineBlogPost ---

func (s *AIUsecaseTestSuite) TestRefineBlogPost() {
	ctx := context.Background()
	originalContent := "go is a good lang. it make fast apis."

	s.Run("Success", func() {
		s.SetupTest()
		// Arrange
		expectedRefinedContent := "Go is an excellent language for building high-performance APIs."
		s.mockAIService.On("GenerateCompletion", mock.Anything, mock.Anything).Return(expectedRefinedContent, nil).Once()

		// Act
		refined, err := s.usecase.RefineBlogPost(ctx, originalContent)

		// Assert
		s.NoError(err)
		s.Equal(expectedRefinedContent, refined)
		s.mockAIService.AssertExpectations(s.T())
	})

	s.Run("Success - Handles extra whitespace from AI", func() {
		s.SetupTest()
		// Arrange
		aiResponseWithWhitespace := "\n  Refined content here.  \t"
		expectedCleanContent := "Refined content here."
		s.mockAIService.On("GenerateCompletion", mock.Anything, mock.Anything).Return(aiResponseWithWhitespace, nil).Once()

		// Act
		refined, err := s.usecase.RefineBlogPost(ctx, originalContent)

		// Assert
		s.NoError(err)
		s.Equal(expectedCleanContent, refined)
	})

	s.Run("Failure - Empty content provided", func() {
		s.SetupTest()
		// Act
		refined, err := s.usecase.RefineBlogPost(ctx, "   ") // Content with only whitespace

		// Assert
		s.Error(err)
		s.ErrorIs(err, domain.ErrValidation)
		s.Empty(refined)
		s.mockAIService.AssertNotCalled(s.T(), "GenerateCompletion", mock.Anything, mock.Anything)
	})
}
