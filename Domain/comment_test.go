package domain_test

import (
	"strings"
	"testing"
	"time"

	. "A2SV_Starter_Project_Blog/Domain"

	"github.com/stretchr/testify/suite"
)

type CommentTestSuite struct {
	suite.Suite
}

func TestCommentTestSuite(t *testing.T) {
	suite.Run(t, new(CommentTestSuite))
}

func (s *CommentTestSuite) TestNewComment() {

	// --- Test the "Happy Path" (Success Case) ---

	s.Run("Success - Top Level Comment", func() {
		// Arrange
		blogID := "blog-123"
		authorID := "user-abc"
		content := "  This is a great post!  " // Test content trimming

		// Act
		comment, err := NewComment(blogID, authorID, content, nil)

		// Assert
		s.NoError(err, "NewComment should not return an error for valid input")
		s.NotNil(comment, "Comment should not be nil on success")

		s.Equal(blogID, comment.BlogID)
		s.Require().NotNil(comment.AuthorID, "AuthorID should not be nil")
		s.Equal(authorID, *comment.AuthorID)
		s.Equal("This is a great post!", comment.Content, "Content should be trimmed")

		s.Nil(comment.ParentID, "ParentID should be nil for a top-level comment")
		s.Equal(int64(0), comment.ReplyCount, "ReplyCount should be initialized to 0")

		s.WithinDuration(time.Now(), comment.CreatedAt, 2*time.Second, "CreatedAt should be recent")
		s.Equal(comment.CreatedAt, comment.UpdatedAt, "UpdatedAt should equal CreatedAt on creation")
	})

	s.Run("Success - Reply Comment", func() {
		// Arrange
		parentID := "parent-comment-xyz"

		// Act
		comment, err := NewComment("blog-123", "user-abc", "A valid reply", &parentID)

		// Assert
		s.NoError(err)
		s.NotNil(comment)
		s.Require().NotNil(comment.ParentID, "ParentID should not be nil for a reply")
		s.Equal(parentID, *comment.ParentID)
	})

	// --- Test the Validation Failure Cases ---

	s.Run("Failure - Invalid Inputs", func() {
		// We can test multiple failure cases in a structured way.
		testCases := []struct {
			name     string
			blogID   string
			authorID string
			content  string
			parentID *string
		}{
			{name: "Empty BlogID", blogID: " ", authorID: "user-1", content: "c"},
			{name: "Empty AuthorID", blogID: "b-1", authorID: " ", content: "c"},
			{name: "Empty Content", blogID: "b-1", authorID: "user-1", content: " "},
			{name: "Content Too Long", blogID: "b-1", authorID: "user-1", content: strings.Repeat("a", 5001)},
		}

		for _, tc := range testCases {
			s.T().Run(tc.name, func(t *testing.T) {
				// Act
				comment, err := NewComment(tc.blogID, tc.authorID, tc.content, tc.parentID)

				// Assert
				s.Error(err, "NewComment should return an error for invalid input")
				s.ErrorIs(err, ErrValidation, "The error should be a domain validation error")
				s.Nil(comment, "The comment should be nil on failure")
			})
		}
	})

	s.Run("Edge Case - Invalid ParentID", func() {
		// Arrange: An invalid parentID (e.g., just whitespace) should be treated as a nil parentID.
		invalidParentID := "   "

		// Act
		comment, err := NewComment("blog-123", "user-abc", "Valid content", &invalidParentID)

		// Assert
		s.NoError(err)
		s.NotNil(comment)
		s.Nil(comment.ParentID, "A ParentID with only whitespace should be ignored, resulting in a top-level comment")
	})
}
