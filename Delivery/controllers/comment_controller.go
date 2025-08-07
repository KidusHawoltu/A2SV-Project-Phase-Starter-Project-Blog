package controllers

import (
	"net/http"
	"strconv"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"

	"github.com/gin-gonic/gin"
)

// --- Request DTOs (Data Transfer Objects) ---

// CreateCommentRequest defines the expected body for creating a comment.
type CreateCommentRequest struct {
	Content  string  `json:"content" binding:"required"`
	ParentID *string `json:"parentId,omitempty"`
}

// UpdateCommentRequest defines the expected body for updating a comment.
type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

// --- Response DTOs ---

type CommentResponse struct {
	ID         string    `json:"id"`
	BlogID     string    `json:"blogId"`
	AuthorID   *string   `json:"authorId,omitempty"` // Can be null for deleted comments
	ParentID   *string   `json:"parentId,omitempty"`
	Content    string    `json:"content"`
	ReplyCount int64     `json:"replyCount"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type PaginatedCommentResponse struct {
	Data       []CommentResponse `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

type CommentController struct {
	commentUsecase domain.ICommentUsecase
}

func NewCommentController(usecase domain.ICommentUsecase) *CommentController {
	return &CommentController{
		commentUsecase: usecase,
	}
}

func (cc *CommentController) CreateComment(c *gin.Context) {
	blogID := c.Param("blogID")
	userID := c.GetString("userID") // From auth middleware

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request: 'content' is required"})
		return
	}

	comment, err := cc.commentUsecase.CreateComment(c.Request.Context(), userID, blogID, req.Content, req.ParentID)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toCommentResponse(comment))
}

func (cc *CommentController) UpdateComment(c *gin.Context) {
	commentID := c.Param("commentID")
	userID := c.GetString("userID")

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request: 'content' is required"})
		return
	}

	comment, err := cc.commentUsecase.UpdateComment(c.Request.Context(), userID, commentID, req.Content)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toCommentResponse(comment))
}

func (cc *CommentController) DeleteComment(c *gin.Context) {
	commentID := c.Param("commentID")
	userID := c.GetString("userID")

	if err := cc.commentUsecase.DeleteComment(c.Request.Context(), userID, commentID); err != nil {
		HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (cc *CommentController) GetCommentsForBlog(c *gin.Context) {
	blogID := c.Param("blogID")

	// Parse pagination parameters
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	comments, total, err := cc.commentUsecase.GetCommentsForBlog(c.Request.Context(), blogID, page, limit)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaginatedCommentResponse(comments, total, page, limit))
}

func (cc *CommentController) GetRepliesForComment(c *gin.Context) {
	commentID := c.Param("commentID")

	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	replies, total, err := cc.commentUsecase.GetRepliesForComment(c.Request.Context(), commentID, page, limit)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaginatedCommentResponse(replies, total, page, limit))
}

func toCommentResponse(c *domain.Comment) CommentResponse {
	return CommentResponse{
		ID:         c.ID,
		BlogID:     c.BlogID,
		AuthorID:   c.AuthorID,
		ParentID:   c.ParentID,
		Content:    c.Content,
		ReplyCount: c.ReplyCount,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

func toPaginatedCommentResponse(comments []*domain.Comment, total, page, limit int64) PaginatedCommentResponse {
	commentResponses := make([]CommentResponse, len(comments))
	for i, c := range comments {
		commentResponses[i] = toCommentResponse(c)
	}
	return PaginatedCommentResponse{
		Data: commentResponses,
		Pagination: Pagination{
			Total: total,
			Page:  page,
			Limit: limit,
		},
	}
}
