package repositories_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	// Use dot import for convenience to access exported repository types and functions
	. "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BlogRepositoryTestSuite defines the suite for the blog repository integration tests.
type BlogRepositoryTestSuite struct {
	suite.Suite
	repo          domain.IBlogRepository
	collection    string
	fixedAuthorID primitive.ObjectID // A reusable, valid author ID for tests
}

// SetupTest runs before each test in the suite.
// It initializes a fresh repository instance and a clean collection.
func (s *BlogRepositoryTestSuite) SetupTest() {
	s.collection = "blogs_test"
	s.repo = NewBlogRepository(testDB.Collection(s.collection))
	s.fixedAuthorID = primitive.NewObjectID()
}

// TearDownTest runs after each test in the suite.
// It drops the test collection to ensure test isolation.
func (s *BlogRepositoryTestSuite) TearDownTest() {
	err := testDB.Collection(s.collection).Drop(context.Background())
	s.Require().NoError(err, "Failed to drop test collection")
}

// TestBlogRepository is the entry point for the test suite.
func TestBlogRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}
	suite.Run(t, new(BlogRepositoryTestSuite))
}

// TestCreateAndGetByID asserts that a blog can be created and then retrieved successfully.
func (s *BlogRepositoryTestSuite) TestCreateAndGetByID() {
	ctx := context.Background()
	blog, err := domain.NewBlog("Test Title", "Test Content", s.fixedAuthorID.Hex(), []string{"go", "test"})
	s.Require().NoError(err, "NewBlog should not fail")

	// Act: Create the blog
	err = s.repo.Create(ctx, blog)

	// Assert: Check creation
	s.NoError(err, "Create should succeed")
	s.NotEmpty(blog.ID, "Blog ID should be populated by the Create method")

	// Act: Get the blog by its new ID
	foundBlog, err := s.repo.GetByID(ctx, blog.ID)

	// Assert: Check retrieval
	s.NoError(err, "GetByID should succeed")
	s.NotNil(foundBlog)
	s.Equal(blog.ID, foundBlog.ID)
	s.Equal("Test Title", foundBlog.Title)
	s.Equal(s.fixedAuthorID.Hex(), foundBlog.AuthorID)
	s.Equal([]string{"go", "test"}, foundBlog.Tags)
}

// TestGetByID_NotFound asserts that ErrNotFound is returned for a non-existent ID.
func (s *BlogRepositoryTestSuite) TestGetByID_NotFound() {
	ctx := context.Background()
	nonExistentID := primitive.NewObjectID().Hex()

	foundBlog, err := s.repo.GetByID(ctx, nonExistentID)

	s.Error(err, "GetByID should return an error for a non-existent ID")
	s.ErrorIs(err, usecases.ErrNotFound, "The error should be ErrNotFound")
	s.Nil(foundBlog, "The returned blog should be nil")
}

// TestGetByID_InvalidIDFormat asserts that ErrNotFound is returned for a malformed ID string.
func (s *BlogRepositoryTestSuite) TestGetByID_InvalidIDFormat() {
	ctx := context.Background()
	invalidID := "not-a-valid-object-id"

	foundBlog, err := s.repo.GetByID(ctx, invalidID)

	s.Error(err, "GetByID should return an error for a malformed ID")
	s.ErrorIs(err, usecases.ErrNotFound, "The error should be ErrNotFound")
	s.Nil(foundBlog, "The returned blog should be nil")
}

// TestUpdate asserts that a blog's details can be updated correctly.
func (s *BlogRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()
	// Arrange: Create an initial blog
	blog, _ := domain.NewBlog("Initial Title", "Initial Content", s.fixedAuthorID.Hex(), nil)
	err := s.repo.Create(ctx, blog)
	s.Require().NoError(err)

	// Act: Modify the blog object
	blog.Title = "Updated Title"
	blog.Tags = []string{"updated"}
	blog.UpdatedAt = time.Now()
	err = s.repo.Update(ctx, blog)

	// Assert: Check the update operation
	s.NoError(err, "Update should succeed")

	// Verify: Fetch the blog again and check if the fields are updated
	updatedBlog, err := s.repo.GetByID(ctx, blog.ID)
	s.NoError(err)
	s.Equal("Updated Title", updatedBlog.Title)
	s.Equal([]string{"updated"}, updatedBlog.Tags)
	s.WithinDuration(blog.UpdatedAt, updatedBlog.UpdatedAt, time.Second, "UpdatedAt should be close to what was set")
}

// TestDelete asserts that a blog can be deleted and is no longer retrievable.
func (s *BlogRepositoryTestSuite) TestDelete() {
	ctx := context.Background()
	// Arrange: Create a blog to delete
	blog, _ := domain.NewBlog("To Be Deleted", "Content", s.fixedAuthorID.Hex(), nil)
	err := s.repo.Create(ctx, blog)
	s.Require().NoError(err)

	// Act: Delete the blog
	err = s.repo.Delete(ctx, blog.ID)

	// Assert: Check the delete operation
	s.NoError(err, "Delete should succeed")

	// Verify: Try to fetch the deleted blog
	foundBlog, err := s.repo.GetByID(ctx, blog.ID)
	s.ErrorIs(err, usecases.ErrNotFound, "GetByID after Delete should return ErrNotFound")
	s.Nil(foundBlog)
}

// TestFetch asserts that pagination and sorting work correctly.
func (s *BlogRepositoryTestSuite) TestFetch() {
	ctx := context.Background()
	// Arrange: Create multiple blogs with different timestamps to test sorting
	blog1, _ := domain.NewBlog("Oldest Post", "Content 1", s.fixedAuthorID.Hex(), nil)
	time.Sleep(10 * time.Millisecond) // Ensure timestamps are distinct
	blog2, _ := domain.NewBlog("Middle Post", "Content 2", s.fixedAuthorID.Hex(), nil)
	time.Sleep(10 * time.Millisecond)
	blog3, _ := domain.NewBlog("Newest Post", "Content 3", s.fixedAuthorID.Hex(), nil)

	s.Require().NoError(s.repo.Create(ctx, blog1))
	s.Require().NoError(s.repo.Create(ctx, blog2))
	s.Require().NoError(s.repo.Create(ctx, blog3))

	// Act: Fetch the first page (limit 2)
	blogs, total, err := s.repo.Fetch(ctx, 1, 2)

	// Assert: First page
	s.NoError(err)
	s.Equal(int64(3), total, "Total count should be 3")
	s.Len(blogs, 2, "Should fetch 2 blogs on the first page")
	// Verify sorting (most recent first)
	s.Equal("Newest Post", blogs[0].Title)
	s.Equal("Middle Post", blogs[1].Title)

	// Act: Fetch the second page
	blogs, total, err = s.repo.Fetch(ctx, 2, 2)

	// Assert: Second page
	s.NoError(err)
	s.Equal(int64(3), total, "Total count should still be 3")
	s.Len(blogs, 1, "Should fetch 1 blog on the second page")
	s.Equal("Oldest Post", blogs[0].Title)
}
