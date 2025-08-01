package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrValidation = errors.New("validation error")
)

type Blog struct {
	ID        string
	Title     string
	Content   string
	AuthorID  string
	Tags      []string
	Views     int64
	Likes     int64
	Dislikes  int64
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
		Title:     title,
		Content:   content,
		AuthorID:  authorID,
		Tags:      tags,
		Views:     0, // Initialize views to 0
		Likes:     0, // Initialize likes to 0
		Dislikes:  0, // Initialize dislikes to 0
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
