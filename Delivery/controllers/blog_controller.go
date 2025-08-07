package controllers

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CreateBlogRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Tags    []string `json:"tags"`
}

type UpdateBlogRequest map[string]interface{}

type InteractBlogRequest struct {
	Action domain.ActionType `json:"action" binding:"required,oneof=like dislike"`
}

type BlogResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	AuthorID  string    `json:"author_id"`
	Tags      []string  `json:"tags"`
	Views     int64     `json:"views"`
	Likes     int64     `json:"likes"`
	Dislikes  int64     `json:"dislikes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Pagination struct {
	Total int64 `json:"total"`
	Page  int64 `json:"page"`
	Limit int64 `json:"limit"`
}

type PaginatedBlogResponse struct {
	Data       []BlogResponse `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

type BlogController struct {
	blogUsecase domain.IBlogUsecase
}

func NewBlogController(usecase domain.IBlogUsecase) *BlogController {
	return &BlogController{
		blogUsecase: usecase,
	}
}

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
	blogID := c.Param("blogID")

	blog, err := bc.blogUsecase.GetByID(c.Request.Context(), blogID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toBlogResponse(blog))
}

func (bc *BlogController) SearchAndFilter(c *gin.Context) {
	options := domain.BlogSearchFilterOptions{
		GlobalLogic: domain.GlobalLogicAND, // Default to AND logic for filers
		TagLogic:    domain.GlobalLogicOR,  // Default to OR logic for tags
		SortOrder:   domain.SortOrderDESC,  // Default to newest first
	}

	// 2. Parse all optional query parameters from the request.

	// Pagination
	page, err := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'page' parameter"})
		return
	}
	options.Page = page

	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil || limit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'limit' parameter"})
		return
	}
	options.Limit = limit

	// Global Options
	if strings.ToUpper(c.Query("logic")) == string(domain.GlobalLogicOR) {
		options.GlobalLogic = domain.GlobalLogicOR
	}

	// Search criteria (using pointers for optional fields)
	if title := c.Query("title"); title != "" {
		options.Title = &title
	}
	if authorName := c.Query("authorName"); authorName != "" {
		options.AuthorName = &authorName
	}

	// Tag filtering
	if tagStr := c.Query("tags"); tagStr != "" {
		options.Tags = strings.Split(tagStr, ",")
	}
	if strings.ToUpper(c.Query("tagLogic")) == string(domain.GlobalLogicAND) {
		options.TagLogic = domain.GlobalLogicAND
	}

	// Date range filtering (using pointers)
	// Example format: ?startDate=2023-10-27T10:00:00Z
	if startDateStr := c.Query("startDate"); startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			options.StartDate = &t
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'startDate' format. Use RFC3339 (e.g., 2023-10-27T10:00:00Z)"})
			return
		}
	}
	if endDateStr := c.Query("endDate"); endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			options.EndDate = &t
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid 'endDate' format. Use RFC3339 (e.g., 2023-10-27T10:00:00Z)"})
			return
		}
	}

	// Sorting
	options.SortBy = c.Query("sortBy") // e.g., "date", "popularity", "title"
	if strings.ToUpper(c.Query("sortOrder")) == string(domain.SortOrderASC) {
		options.SortOrder = domain.SortOrderASC
	}

	// 3. Call the usecase with the populated options struct.
	blogs, total, err := bc.blogUsecase.SearchAndFilter(c.Request.Context(), options)
	if err != nil {
		HandleError(c, err)
		return
	}

	// 4. Return the paginated response.
	c.JSON(http.StatusOK, toPaginatedBlogResponse(blogs, total, options.Page, options.Limit))
}

func (bc *BlogController) Update(c *gin.Context) {
	blogID := c.Param("blogID")
	userID := c.GetString("userID")
	userRole := domain.Role(c.GetString("role"))

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
	blogID := c.Param("blogID")
	userID := c.GetString("userID")
	userRole := domain.Role(c.GetString("role"))

	err := bc.blogUsecase.Delete(c.Request.Context(), blogID, userID, userRole)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (bc *BlogController) InteractWithBlog(c *gin.Context) {
	// 1. Parse required parameters from the URL and context.
	blogID := c.Param("blogID")
	userID := c.GetString("userID") // Set by the authentication middleware.

	// 2. Bind and validate the JSON request body.
	var req InteractBlogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request: 'action' field is required and must be 'like' or 'dislike'"})
		return
	}

	// 3. Call the single, consolidated usecase method with the parsed data.
	err := bc.blogUsecase.InteractWithBlog(c.Request.Context(), blogID, userID, req.Action)
	if err != nil {
		// The usecase will return errors like ErrNotFound, which HandleError will correctly process.
		HandleError(c, err)
		return
	}

	// 4. On success, return a 200 OK status with no body.
	c.Status(http.StatusOK)
}

// ===========================================
// HELPERS
// ===========================================

func HandleError(c *gin.Context, err error) {
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
		log.Printf("Internal Server Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "An unexpected error occurred"})
	}
}

func toBlogResponse(b *domain.Blog) BlogResponse {
	return BlogResponse{
		ID:        b.ID,
		Title:     b.Title,
		Content:   b.Content,
		AuthorID:  b.AuthorID,
		Tags:      b.Tags,
		Views:     b.Views,
		Likes:     b.Likes,
		Dislikes:  b.Dislikes,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}
}

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

