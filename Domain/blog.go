package domain

import (
	"strings"
	"time"
)

type Blog struct {
	ID            string
	Title         string
	Content       string
	AuthorID      string
	Tags          []string
	Views         int64
	Likes         int64
	Dislikes      int64
	CommentsCount int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type GlobalLogic string
type SortOrder string
type ActionType string

const (
	GlobalLogicOR  GlobalLogic = "OR"
	GlobalLogicAND GlobalLogic = "AND"

	SortOrderASC  SortOrder = "ASC"
	SortOrderDESC SortOrder = "DESC"

	ActionTypeLike    ActionType = "like"
	ActionTypeDislike ActionType = "dislike"
)

type BlogSearchFilterOptions struct {
	Title      *string
	AuthorName *string
	AuthorIDs  []string
	// AND or OR
	GlobalLogic GlobalLogic

	// List of tags
	Tags []string
	// AND or OR
	TagLogic GlobalLogic

	StartDate *time.Time
	EndDate   *time.Time

	Page  int64
	Limit int64

	SortBy string
	// ASC or DESC
	SortOrder SortOrder
}

type BlogInteraction struct {
	ID        string
	UserID    string
	BlogID    string
	Action    ActionType
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewBlog(title, content string, authorID string, tags []string) (*Blog, error) {
	if strings.TrimSpace(title) == "" {
		return nil, ErrValidation
	}
	if strings.TrimSpace(content) == "" {
		return nil, ErrValidation
	}
	if strings.TrimSpace(authorID) == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()

	return &Blog{
		Title:         title,
		Content:       content,
		AuthorID:      authorID,
		Tags:          tags,
		Views:         0, // Initialize views to 0
		Likes:         0, // Initialize likes to 0
		Dislikes:      0, // Initialize dislikes to 0
		CommentsCount: 0, // Initialize comments to 0
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}
