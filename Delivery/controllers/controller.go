package controllers

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

//==============================================================================
// DTOs (Data Transfer Objects)
//==============================================================================

// CreateBlogRequest defines the structure for a new blog post request.
type CreateBlogRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

// UpdateBlogRequest defines the structure for updating a blog post.
type UpdateBlogRequest map[string]interface{}

// BlogResponse is the standard format for a single blog post returned to the client.
type BlogResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	Tags      []string  `json:"tags"`
	Views     int64     `json:"views"`
	Likes     int64     `json:"likes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Pagination represents the metadata for a paginated response.
type Pagination struct {
	Total int64 `json:"total"`
	Page  int64 `json:"page"`
	Limit int64 `json:"limit"`
}

// PaginatedBlogResponse is the format for a list of blogs.
type PaginatedBlogResponse struct {
	Data       []BlogResponse `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

//==============================================================================
// Controller
//==============================================================================

// BlogController holds the usecase dependency.
type BlogController struct {
	blogUsecase domain.IBlogUsecase
}

// NewBlogController is the constructor for BlogController.
func NewBlogController(usecase domain.IBlogUsecase) *BlogController {
	return &BlogController{
		blogUsecase: usecase,
	}
}

// --- Handler Methods ---

func (bc *BlogController) Create(c *gin.Context) {
	var req CreateBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body: " + err.Error()})
		return
	}

	userID := c.GetString("userID")

	blog, err := bc.blogUsecase.Create(c.Request.Context(), req.Title, req.Content, userID, req.Tags)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toBlogResponse(blog))
}

func (bc *BlogController) GetByID(c *gin.Context) {
	blogID := c.Param("id")

	blog, err := bc.blogUsecase.GetByID(c.Request.Context(), blogID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toBlogResponse(blog))
}

func (bc *BlogController) Fetch(c *gin.Context) {
	page, err := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'page' parameter"})
		return
	}

	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'limit' parameter"})
		return
	}

	blogs, total, err := bc.blogUsecase.Fetch(c.Request.Context(), page, limit)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaginatedBlogResponse(blogs, total, page, limit))
}

func (bc *BlogController) Update(c *gin.Context) {
	blogID := c.Param("id")
	userID := c.GetString("userID")
	userRole := c.GetString("role")

	var updates UpdateBlogRequest
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body: " + err.Error()})
		return
	}

	updatedBlog, err := bc.blogUsecase.Update(c.Request.Context(), blogID, userID, userRole, updates)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toBlogResponse(updatedBlog))
}

func (bc *BlogController) Delete(c *gin.Context) {
	blogID := c.Param("id")
	userID := c.GetString("userID")
	userRole := c.GetString("role")

	err := bc.blogUsecase.Delete(c.Request.Context(), blogID, userID, userRole)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

//==============================================================================
// Helpers
//==============================================================================

// HandleError centralizes error handling for the controller.
// It maps domain and usecase errors to appropriate HTTP status codes.
func HandleError(c *gin.Context, err error) {
	// Map specific, known errors to HTTP status codes.
	switch {
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, usecases.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "The requested resource was not found"})
	case errors.Is(err, domain.ErrPermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"message": "You do not have permission to perform this action"})
	case errors.Is(err, usecases.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	default:
		// Any other error is treated as an internal server error.
		// It's crucial to log these for debugging.
		log.Printf("Internal Server Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "An unexpected error occurred"})
	}
}

// toBlogResponse maps a domain.Blog entity to its API response representation.
func toBlogResponse(b *domain.Blog) BlogResponse {
	return BlogResponse{
		ID:        b.ID,
		Title:     b.Title,
		Content:   b.Content,
		AuthorID:  b.AuthorID,
		Tags:      b.Tags,
		Views:     b.Views,
		Likes:     b.Likes,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

// toPaginatedBlogResponse maps a slice of blogs and pagination data to the API response format.
func toPaginatedBlogResponse(blogs []*domain.Blog, total, page, limit int64) PaginatedBlogResponse {
	blogResponses := make([]BlogResponse, len(blogs))
	for i, b := range blogs {
		blogResponses[i] = toBlogResponse(b)
	}

	return PaginatedBlogResponse{
		Data: blogResponses,
		Pagination: Pagination{
			Total: total,
			Page:  page,
			Limit: limit,
		},
	}
}
