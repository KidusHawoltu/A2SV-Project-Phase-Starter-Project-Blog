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

func (s *BlogRepositoryTestSuite) TestSearchAndFilter() {
	ctx := context.Background()
	author1 := primitive.NewObjectID()
	author2 := primitive.NewObjectID()

	// Arrange: Create a diverse set of blogs to test against.
	blogsToCreate := []*domain.Blog{
		{Title: "Golang Basics", AuthorID: author1.Hex(), Tags: []string{"go", "beginner"}, CreatedAt: time.Now().Add(-5 * 24 * time.Hour), Views: 100, Likes: 10},     // 0
		{Title: "Advanced Golang", AuthorID: author1.Hex(), Tags: []string{"go", "advanced"}, CreatedAt: time.Now().Add(-4 * 24 * time.Hour), Views: 200, Likes: 20},   // 1
		{Title: "Intro to Docker", AuthorID: author2.Hex(), Tags: []string{"docker", "devops"}, CreatedAt: time.Now().Add(-3 * 24 * time.Hour), Views: 150, Likes: 15}, // 2
		{Title: "Docker with Go", AuthorID: author2.Hex(), Tags: []string{"docker", "go"}, CreatedAt: time.Now().Add(-2 * 24 * time.Hour), Views: 300, Likes: 5},       // 3
		{Title: "REST APIs", AuthorID: author1.Hex(), Tags: []string{"api", "backend"}, CreatedAt: time.Now().Add(-1 * 24 * time.Hour), Views: 50, Likes: 30},          // 4
	}

	for _, blog := range blogsToCreate {
		err := s.repo.Create(ctx, blog)
		s.Require().NoError(err)
	}

	// Helper to extract IDs for easier comparison
	getBlogIDs := func(blogs []*domain.Blog) []string {
		ids := make([]string, len(blogs))
		for i, b := range blogs {
			ids[i] = b.ID
		}
		return ids
	}

	s.Run("No Filters", func() {
		opts := domain.BlogSearchFilterOptions{Page: 1, Limit: 10}
		blogs, total, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Len(blogs, 5)
		s.Equal(int64(5), total)
	})

	s.Run("Filter by Title", func() {
		title := "golang"
		opts := domain.BlogSearchFilterOptions{Title: &title, Page: 1, Limit: 10}
		blogs, total, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Len(blogs, 2)
		s.Equal(int64(2), total)
		s.ElementsMatch([]string{blogsToCreate[0].ID, blogsToCreate[1].ID}, getBlogIDs(blogs))
	})

	s.Run("Filter by AuthorIDs", func() {
		opts := domain.BlogSearchFilterOptions{AuthorIDs: []string{author2.Hex()}, Page: 1, Limit: 10}
		blogs, total, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Len(blogs, 2)
		s.Equal(int64(2), total)
		s.ElementsMatch([]string{blogsToCreate[2].ID, blogsToCreate[3].ID}, getBlogIDs(blogs))
	})

	s.Run("Filter by Tags with OR logic", func() {
		opts := domain.BlogSearchFilterOptions{Tags: []string{"advanced", "api"}, TagLogic: domain.GlobalLogicOR, Page: 1, Limit: 10}
		blogs, _, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Len(blogs, 2)
		s.ElementsMatch([]string{blogsToCreate[1].ID, blogsToCreate[4].ID}, getBlogIDs(blogs))
	})

	s.Run("Filter by Tags with AND logic", func() {
		opts := domain.BlogSearchFilterOptions{Tags: []string{"docker", "go"}, TagLogic: domain.GlobalLogicAND, Page: 1, Limit: 10}
		blogs, _, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Len(blogs, 1)
		s.Equal(blogsToCreate[3].ID, blogs[0].ID)
	})

	s.Run("Filter by Date Range", func() {
		startDate := time.Now().Add(-3 * 24 * time.Hour).Add(-time.Hour) // 3 days and 1 hour ago
		endDate := time.Now().Add(-1 * 24 * time.Hour).Add(time.Hour)    // 1 day ago plus 1 hour
		opts := domain.BlogSearchFilterOptions{StartDate: &startDate, EndDate: &endDate, Page: 1, Limit: 10}
		blogs, _, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Len(blogs, 3)
		s.ElementsMatch([]string{blogsToCreate[2].ID, blogsToCreate[3].ID, blogsToCreate[4].ID}, getBlogIDs(blogs))
	})

	s.Run("Global AND Logic", func() {
		title := "Docker"
		opts := domain.BlogSearchFilterOptions{
			GlobalLogic: domain.GlobalLogicAND,
			Title:       &title,
			AuthorIDs:   []string{author2.Hex()},
			Page:        1, Limit: 10,
		}
		blogs, _, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Len(blogs, 2) // "Intro to Docker" and "Docker with Go"
	})

	s.Run("Global OR Logic", func() {
		title := "APIs"
		opts := domain.BlogSearchFilterOptions{
			GlobalLogic: domain.GlobalLogicOR,
			Title:       &title,
			AuthorIDs:   []string{author2.Hex()}, // Author2 has 2 blogs
			Page:        1, Limit: 10,
		}
		blogs, _, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Len(blogs, 3) // Blog with title "REST APIs" + author2's two blogs
		s.ElementsMatch([]string{blogsToCreate[2].ID, blogsToCreate[3].ID, blogsToCreate[4].ID}, getBlogIDs(blogs))
	})

	s.Run("Sort by Popularity", func() {
		opts := domain.BlogSearchFilterOptions{SortBy: "popularity", Page: 1, Limit: 10}
		blogs, _, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		// Expected order by views desc: 300, 200, 150, 100, 50
		expectedOrder := []string{blogsToCreate[3].ID, blogsToCreate[1].ID, blogsToCreate[2].ID, blogsToCreate[0].ID, blogsToCreate[4].ID}
		s.Equal(expectedOrder, getBlogIDs(blogs))
	})

	s.Run("Pagination", func() {
		// Sort by date ascending to get a predictable order
		opts := domain.BlogSearchFilterOptions{SortBy: "date", SortOrder: domain.SortOrderASC, Page: 2, Limit: 2}
		blogs, total, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Equal(int64(5), total)
		s.Len(blogs, 2)
		// Page 1 would be [0, 1]. Page 2 should be [2, 3].
		s.Equal(blogsToCreate[2].ID, blogs[0].ID)
		s.Equal(blogsToCreate[3].ID, blogs[1].ID)
	})
}
