package domain_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// BlogDomainTestSuite remains the same
type BlogDomainTestSuite struct {
	suite.Suite
}

// TestBlogDomainTestSuite runner remains the same
func TestBlogDomainTestSuite(t *testing.T) {
	suite.Run(t, new(BlogDomainTestSuite))
}

// TestNewBlog_Success is updated to handle the new return signature.
func (s *BlogDomainTestSuite) TestNewBlog_Success() {
	// Arrange
	title := "My First Blog Post"
	content := "This is the content of the blog."
	authorID := "user-12345"
	tags := []string{"go", "clean-architecture", "testing"}

	// Act: Call the factory and check for an error.
	blog, err := domain.NewBlog(title, content, authorID, tags)

	// Assert
	s.NoError(err, "Creating a blog with valid data should not produce an error")
	s.NotNil(blog, "The created blog should not be nil on success")

	// The rest of the assertions for the happy path remain the same.
	s.Equal(title, blog.Title)
	s.Equal(content, blog.Content)
	s.Equal(authorID, blog.AuthorID)
	s.Equal(tags, blog.Tags)
	s.Empty(blog.ID)
	s.Equal(int64(0), blog.Views)
	s.Equal(int64(0), blog.Likes)
	s.Equal(int64(0), blog.Dislikes)
	s.WithinDuration(time.Now().UTC(), blog.CreatedAt, 2*time.Second)
	s.Equal(blog.CreatedAt, blog.UpdatedAt)
}

func (s *BlogDomainTestSuite) TestNewBlog_ValidationFailure() {
	// Define a table of test cases to avoid repetitive code.
	testCases := []struct {
		name     string
		title    string
		content  string
		authorID string
	}{
		{
			name:     "EmptyTitle",
			title:    "", // Invalid
			content:  "Some content",
			authorID: "user-1",
		},
		{
			name:     "WhitespaceTitle",
			title:    "   ", // Invalid
			content:  "Some content",
			authorID: "user-1",
		},
		{
			name:     "EmptyContent",
			title:    "A Title",
			content:  "", // Invalid
			authorID: "user-1",
		},
		{
			name:     "WhitespaceContent",
			title:    "A Title",
			content:  " \t\n ", // Invalid
			authorID: "user-1",
		},
		{
			name:     "EmptyAuthorID",
			title:    "A Title",
			content:  "Some content",
			authorID: "", // Invalid
		},
		{
			name:     "WhitespaceAuthorID",
			title:    "A Title",
			content:  "Some content",
			authorID: "  ", // Invalid
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			blog, err := domain.NewBlog(tc.title, tc.content, tc.authorID, nil)

			// Assert: Check that the correct error is returned and the object is nil.
			s.Error(err, "Should return an error for invalid input")
			s.ErrorIs(err, domain.ErrValidation, "The error should be a domain validation error")
			s.Nil(blog, "The blog object should be nil on validation failure")
		})
	}
}

// TestNewBlog_ValidEdgeCases tests valid but non-standard inputs.
func (s *BlogDomainTestSuite) TestNewBlog_ValidEdgeCases() {
	s.Run("WithNilTags", func() {
		blog, err := domain.NewBlog("Title", "Content", "author-id", nil)

		s.NoError(err, "Nil tags are valid and should not cause an error")
		s.NotNil(blog)
		s.Nil(blog.Tags, "If input tags are nil, the blog's tags should also be nil")
	})

	s.Run("WithEmptyTagsSlice", func() {
		blog, err := domain.NewBlog("Title", "Content", "author-id", []string{})

		s.NoError(err, "An empty tags slice is valid and should not cause an error")
		s.NotNil(blog)
		s.NotNil(blog.Tags, "Tags slice should be initialized, not nil")
		s.Len(blog.Tags, 0, "Tags slice should be empty")
	})
}
