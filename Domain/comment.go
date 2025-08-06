package domain

import (
	"strings"
	"time"
)

type Comment struct {
	ID       string
	BlogID   string
	AuthorID *string
	Content  string

	ParentID *string // nil for top level comments

	ReplyCount int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewComment(blogID, authorID, content string, parentID *string) (*Comment, error) {
	if strings.TrimSpace(blogID) == "" {
		return nil, ErrValidation
	}
	if strings.TrimSpace(authorID) == "" {
		return nil, ErrValidation
	}
	if strings.TrimSpace(content) == "" {
		return nil, ErrValidation
	}

	const maxCommentLength = 5000
	if len(content) > maxCommentLength {
		return nil, ErrValidation
	}

	// if parentId is invlid, save the comment as a top level comment
	var validParentID *string
	if parentID != nil && strings.TrimSpace(*parentID) != "" {
		validParentID = parentID
	}

	now := time.Now().UTC()
	newAuthorId := authorID

	return &Comment{
		BlogID:   blogID,
		AuthorID: &newAuthorId,
		Content:  strings.TrimSpace(content),
		ParentID: validParentID,

		ReplyCount: 0,

		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
