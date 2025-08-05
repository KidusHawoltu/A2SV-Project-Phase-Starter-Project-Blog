package controllers

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AISuggestRequest defines the shape of the incoming JSON body for the AI endpoint.
// It uses an 'action' field to determine which usecase to call.
// The Gin `binding` tag provides automatic validation.
type AISuggestRequest struct {
	Action   string   `json:"action" binding:"required,oneof=generate_ideas refine_content"`
	Keywords []string `json:"keywords,omitempty"`
	Content  string   `json:"content,omitempty"`
}

// AISuggestResponse defines the shape of a successful JSON response.
// `omitempty` ensures that only the relevant field is included in the output.
type AISuggestResponse struct {
	Suggestions    []string `json:"suggestions,omitempty"`
	RefinedContent string   `json:"refinedContent,omitempty"`
}

type AIController struct {
	aiUsecase domain.IAIUsecase
}

func NewAIController(usecase domain.IAIUsecase) *AIController {
	return &AIController{
		aiUsecase: usecase,
	}
}

// Suggest is the handler for the POST /ai/suggest endpoint.
func (ac *AIController) Suggest(c *gin.Context) {
	// 1. Bind and validate the incoming JSON. Gin handles the 'action' field validation.
	var req AISuggestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body. 'action' must be 'generate_ideas' or 'refine_content'"})
		return
	}

	// 2. Route to the appropriate usecase based on the action.
	switch req.Action {
	case "generate_ideas":
		if len(req.Keywords) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "The 'keywords' field is required for the 'generate_ideas' action."})
			return
		}

		ideas, err := ac.aiUsecase.GenerateBlogIdeas(c.Request.Context(), req.Keywords)
		if err != nil {
			HandleError(c, err)
			return
		}

		c.JSON(http.StatusOK, AISuggestResponse{Suggestions: ideas})

	case "refine_content":
		if req.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "The 'content' field is required for the 'refine_content' action."})
			return
		}

		refined, err := ac.aiUsecase.RefineBlogPost(c.Request.Context(), req.Content)
		if err != nil {
			HandleError(c, err)
			return
		}

		c.JSON(http.StatusOK, AISuggestResponse{RefinedContent: refined})
	}
}
