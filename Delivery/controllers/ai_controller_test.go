package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "A2SV_Starter_Project_Blog/Delivery/controllers"
	usecases "A2SV_Starter_Project_Blog/Usecases"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// --- Mock IAIUsecase ---
type MockAIUsecase struct {
	mock.Mock
}

func (m *MockAIUsecase) GenerateBlogIdeas(ctx context.Context, keywords []string) ([]string, error) {
	args := m.Called(ctx, keywords)
	var ideas []string
	if args.Get(0) != nil {
		ideas = args.Get(0).([]string)
	}
	return ideas, args.Error(1)
}

func (m *MockAIUsecase) RefineBlogPost(ctx context.Context, content string) (string, error) {
	args := m.Called(ctx, content)
	return args.String(0), args.Error(1)
}

// --- Test Suite Setup ---
type AIControllerTestSuite struct {
	suite.Suite
	mockAIUsecase *MockAIUsecase
	router        *gin.Engine
	controller    *AIController
}

func (s *AIControllerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.mockAIUsecase = new(MockAIUsecase)
	s.controller = NewAIController(s.mockAIUsecase)
	s.router = gin.New()

	// The endpoint is protected, so we add a dummy middleware for tests
	// that just sets the userID, mimicking the real auth middleware.
	authMiddleware := func(c *gin.Context) {
		c.Set("userID", "test-user-id")
		c.Next()
	}
	s.router.POST("/ai/suggest", authMiddleware, s.controller.Suggest)
}

func TestAIControllerTestSuite(t *testing.T) {
	suite.Run(t, new(AIControllerTestSuite))
}

// --- Tests ---

func (s *AIControllerTestSuite) TestSuggest_GenerateIdeas() {
	s.Run("Success", func() {
		s.SetupTest()
		// Arrange
		keywords := []string{"go", "performance"}
		expectedIdeas := []string{"High-Performance Go", "Optimizing Go Apps"}

		// Set mock expectation for the usecase
		s.mockAIUsecase.On("GenerateBlogIdeas", mock.Anything, keywords).Return(expectedIdeas, nil).Once()

		// Create the request body
		reqBody := AISuggestRequest{Action: "generate_ideas", Keywords: keywords}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/ai/suggest", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		s.router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code)

		var resp AISuggestResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		s.NoError(err)
		s.Equal(expectedIdeas, resp.Suggestions)
		s.Empty(resp.RefinedContent) // Ensure the other field is not populated

		s.mockAIUsecase.AssertExpectations(s.T())
	})

	s.Run("Failure - Missing keywords", func() {
		s.SetupTest()
		// Arrange: Request body is missing the required 'keywords' field for this action
		reqBody := AISuggestRequest{Action: "generate_ideas"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/ai/suggest", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		s.router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusBadRequest, w.Code)
		// Usecase should not have been called due to controller-level validation
		s.mockAIUsecase.AssertNotCalled(s.T(), "GenerateBlogIdeas", mock.Anything, mock.Anything)
	})
}

func (s *AIControllerTestSuite) TestSuggest_RefineContent() {
	s.Run("Success", func() {
		s.SetupTest()
		// Arrange
		content := "go is fast"
		expectedRefined := "Go is a high-performance programming language."

		s.mockAIUsecase.On("RefineBlogPost", mock.Anything, content).Return(expectedRefined, nil).Once()

		reqBody := AISuggestRequest{Action: "refine_content", Content: content}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/ai/suggest", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		s.router.ServeHTTP(w, req)

		// Assert
		s.Equal(http.StatusOK, w.Code)

		var resp AISuggestResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		s.NoError(err)
		s.Equal(expectedRefined, resp.RefinedContent)
		s.Empty(resp.Suggestions)

		s.mockAIUsecase.AssertExpectations(s.T())
	})
}

func (s *AIControllerTestSuite) TestSuggest_GeneralFailures() {
	s.Run("Failure - Invalid action", func() {
		s.SetupTest()
		// Arrange: The 'action' field has a value not in the 'oneof' validator
		invalidBody := `{"action": "compile_code", "keywords": ["go"]}`
		req := httptest.NewRequest(http.MethodPost, "/ai/suggest", strings.NewReader(invalidBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		s.router.ServeHTTP(w, req)

		// Assert: Gin's validator should catch this
		s.Equal(http.StatusBadRequest, w.Code)
		s.mockAIUsecase.AssertNotCalled(s.T(), "GenerateBlogIdeas")
		s.mockAIUsecase.AssertNotCalled(s.T(), "RefineBlogPost")
	})

	s.Run("Failure - Usecase returns an error", func() {
		s.SetupTest()
		// Arrange
		keywords := []string{"go"}
		expectedErr := usecases.ErrInternal // Simulate a generic internal error

		s.mockAIUsecase.On("GenerateBlogIdeas", mock.Anything, keywords).Return(nil, expectedErr).Once()

		reqBody := AISuggestRequest{Action: "generate_ideas", Keywords: keywords}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/ai/suggest", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		s.router.ServeHTTP(w, req)

		// Assert: Your centralized HandleError should convert this to a 500
		s.Equal(http.StatusInternalServerError, w.Code)
		s.mockAIUsecase.AssertExpectations(s.T())
	})
}
