package repositories_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"math"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
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

func (s *BlogRepositoryTestSuite) TestCreate() {
	ctx := context.Background()
	blog, err := domain.NewBlog("Test Title", "Test Content", s.fixedAuthorID.Hex(), []string{"go", "test"})
	s.Require().NoError(err, "NewBlog should not fail")

	err = s.repo.Create(ctx, blog)
	s.Require().NoError(err, "Create should succeed")
	s.NotEmpty(blog.ID, "Blog ID should be populated")

	// Verify directly from the DB
	var createdModel BlogModel
	objID, _ := primitive.ObjectIDFromHex(blog.ID)
	err = testDB.Collection(s.collection).FindOne(ctx, bson.M{"_id": objID}).Decode(&createdModel)
	s.Require().NoError(err)

	s.Equal("Test Title", createdModel.Title)
	s.Equal(s.fixedAuthorID, createdModel.AuthorID)
	s.Equal([]string{"go", "test"}, createdModel.Tags)
	s.Equal(float64(0), createdModel.EngagementScore, "EngagementScore should be initialized to 0")
	s.Equal(int64(0), createdModel.CommentsCount, "CommentsCount should be initialized to 0")
}

func (s *BlogRepositoryTestSuite) TestGetByID() {
	ctx := context.Background()
	originalBlog, _ := domain.NewBlog("Gettable Blog", "Content", s.fixedAuthorID.Hex(), []string{"get"})
	err := s.repo.Create(ctx, originalBlog)
	s.Require().NoError(err, "Setup for GetByID failed: could not create blog")

	foundBlog, err := s.repo.GetByID(ctx, originalBlog.ID)

	// Assert: Check the results of the GetByID call.
	s.NoError(err, "GetByID should succeed for an existing blog")
	s.NotNil(foundBlog)
	s.Equal(originalBlog.ID, foundBlog.ID)
	s.Equal("Gettable Blog", foundBlog.Title)
	s.Equal(s.fixedAuthorID.Hex(), foundBlog.AuthorID)
	s.Equal([]string{"get"}, foundBlog.Tags)
	s.Equal(int64(0), foundBlog.CommentsCount, "CommentsCount should be mapped correctly")
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

func calculatePopularity(score float64, createdAt time.Time) float64 {
	// Calculate the age of the post in hours.
	ageInHours := time.Since(createdAt).Hours()

	// Denominator: (Age in hours + 2) ^ Gravity
	denominator := math.Pow(ageInHours+2, Gravity)

	// Avoid division by zero, though unlikely with the +2 buffer.
	if denominator == 0 {
		return score
	}

	return score / denominator
}

func (s *BlogRepositoryTestSuite) TestSearchAndFilter() {
	ctx := context.Background()
	author1 := primitive.NewObjectID()
	author2 := primitive.NewObjectID()

	// Arrange: Create a diverse set of blogs to test against.
	blogsToCreate := []*domain.Blog{
		{Title: "Golang Basics", AuthorID: author1.Hex(), Tags: []string{"go", "beginner"}, CreatedAt: time.Now().Add(-120 * time.Hour), Views: 100, Likes: 10, Dislikes: 1, CommentsCount: 5},
		{Title: "Advanced Golang", AuthorID: author1.Hex(), Tags: []string{"go", "advanced"}, CreatedAt: time.Now().Add(-96 * time.Hour), Views: 200, Likes: 20, Dislikes: 2, CommentsCount: 10},
		{Title: "Intro to Docker", AuthorID: author2.Hex(), Tags: []string{"docker", "devops"}, CreatedAt: time.Now().Add(-72 * time.Hour), Views: 150, Likes: 15, Dislikes: 5, CommentsCount: 8},
		{Title: "Docker with Go", AuthorID: author2.Hex(), Tags: []string{"docker", "go"}, CreatedAt: time.Now().Add(-48 * time.Hour), Views: 300, Likes: 5, Dislikes: 10, CommentsCount: 2},
		{Title: "REST APIs", AuthorID: author1.Hex(), Tags: []string{"api", "backend"}, CreatedAt: time.Now().Add(-1 * time.Hour), Views: 50, Likes: 30, Dislikes: 0, CommentsCount: 15},
	}

	engagementScores := make(map[string]float64)
	for i, blog := range blogsToCreate {
		err := s.repo.Create(ctx, blog)
		s.Require().NoError(err)

		// Manually calculate and set the engagement score for the test.
		score := (float64(blog.Likes) * LikeWeight) + (float64(blog.Dislikes) * DislikeWeight) + (float64(blog.Views) * ViewWeight) + (float64(blog.CommentsCount) * CommentWeight)
		objID, _ := primitive.ObjectIDFromHex(blog.ID)
		_, err = testDB.Collection(s.collection).UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"engagementScore": score}})
		s.Require().NoError(err)
		blogsToCreate[i].ID = blog.ID // Ensure original slice has the ID
		engagementScores[blog.ID] = score
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
		endDate := time.Now()                                            // now to include the newest posts
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
		actualBlogs, _, err := s.repo.SearchAndFilter(ctx, opts)
		s.NoError(err)
		s.Require().Len(actualBlogs, 5)

		blogsToSort := make([]*domain.Blog, len(blogsToCreate))
		copy(blogsToSort, blogsToCreate)
		sort.Slice(blogsToSort, func(i, j int) bool {
			scoreI := engagementScores[blogsToSort[i].ID]
			scoreJ := engagementScores[blogsToSort[j].ID]

			popI := calculatePopularity(scoreI, blogsToSort[i].CreatedAt)
			popJ := calculatePopularity(scoreJ, blogsToSort[j].CreatedAt)

			return popI > popJ // Descending order
		})
		actualIDs := getBlogIDs(actualBlogs)
		expectedIDs := getBlogIDs(blogsToSort)

		s.Equal(expectedIDs, actualIDs, "The order of blogs returned by the database does not match the expected popularity sort order")
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

func (s *BlogRepositoryTestSuite) TestIncrementLikes() {
	ctx := context.Background()
	// Arrange: Create a blog with a known number of likes.
	blog, _ := domain.NewBlog("Title", "Content", s.fixedAuthorID.Hex(), nil)
	blog.Likes = 10
	err := s.repo.Create(ctx, blog)
	s.Require().NoError(err)

	s.Run("Increment", func() {
		// Act: Increment the likes count.
		err := s.repo.IncrementLikes(ctx, blog.ID, 1)
		s.NoError(err)

		// Assert: Fetch directly from the DB to verify the change.
		var updatedBlog BlogModel
		objID, err := primitive.ObjectIDFromHex(blog.ID)
		s.Require().NoError(err, "The blog ID from the test setup should be a valid ObjectID hex")
		err = testDB.Collection(s.collection).FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedBlog)
		s.NoError(err)
		s.Equal(int64(11), updatedBlog.Likes, "Likes count should be incremented by 1")
		s.Equal(LikeWeight, updatedBlog.EngagementScore, "Engagement score should increase by the like weight")
	})

	s.Run("Decrement", func() {
		// Act: Decrement the likes count.
		err := s.repo.IncrementLikes(ctx, blog.ID, -1)
		s.NoError(err)

		// Assert: The count should now be back to 10.
		var updatedBlog BlogModel
		objID, err := primitive.ObjectIDFromHex(blog.ID)
		s.Require().NoError(err, "The blog ID from the test setup should be a valid ObjectID hex")
		err = testDB.Collection(s.collection).FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedBlog)
		s.NoError(err)
		s.Equal(int64(10), updatedBlog.Likes, "Likes count should be decremented back to 10")
		s.Equal(float64(0), updatedBlog.EngagementScore, "Engagement score should be back to zero")
	})
}

func (s *BlogRepositoryTestSuite) TestIncrementDislikes() {
	ctx := context.Background()
	// Arrange: Create a blog with a known number of dislikes.
	blog, _ := domain.NewBlog("Title", "Content", s.fixedAuthorID.Hex(), nil)
	blog.Dislikes = 5
	err := s.repo.Create(ctx, blog)
	s.Require().NoError(err)

	// Act: Increment the dislikes count.
	err = s.repo.IncrementDislikes(ctx, blog.ID, 1)
	s.NoError(err)

	// Assert: Fetch directly from the DB to verify the change.
	var updatedBlog BlogModel
	objID, err := primitive.ObjectIDFromHex(blog.ID)
	s.Require().NoError(err, "The blog ID from the test setup should be a valid ObjectID hex")
	err = testDB.Collection(s.collection).FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedBlog)
	s.NoError(err)
	s.Equal(int64(6), updatedBlog.Dislikes, "Dislikes count should be incremented")
	s.Equal(DislikeWeight, updatedBlog.EngagementScore, "Engagement score should decrease by the dislike weight")
}

func (s *BlogRepositoryTestSuite) TestIncrementViews() {
	ctx := context.Background()
	// Arrange: Create a blog with zero views.
	blog, _ := domain.NewBlog("Title", "Content", s.fixedAuthorID.Hex(), nil)
	err := s.repo.Create(ctx, blog)
	s.Require().NoError(err)

	// Act: Call the increment method twice.
	err = s.repo.IncrementViews(ctx, blog.ID)
	s.NoError(err)
	err = s.repo.IncrementViews(ctx, blog.ID)
	s.NoError(err)

	// Assert: Verify the view count is 2.
	var updatedBlog BlogModel
	objID, err := primitive.ObjectIDFromHex(blog.ID)
	s.Require().NoError(err, "The blog ID from the test setup should be a valid ObjectID hex")
	err = testDB.Collection(s.collection).FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedBlog)
	s.NoError(err)
	s.Equal(int64(2), updatedBlog.Views, "Views count should be 2 after two increments")
	s.Equal(2*ViewWeight, updatedBlog.EngagementScore, "Engagement score should be twice the view weight")
}

func (s *BlogRepositoryTestSuite) TestIncrementCommentCount() {
	ctx := context.Background()
	// Arrange: Create a blog with an initial comment count.
	blog, _ := domain.NewBlog("Title", "Content", s.fixedAuthorID.Hex(), nil)
	blog.CommentsCount = 3
	err := s.repo.Create(ctx, blog)
	s.Require().NoError(err)

	// Update the initial engagement score to match the 3 comments
	initialScore := 3 * CommentWeight
	objID, _ := primitive.ObjectIDFromHex(blog.ID)
	_, err = testDB.Collection(s.collection).UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"engagementScore": initialScore}})
	s.Require().NoError(err)

	s.Run("Increment", func() {
		// Act: Increment the comment count by 1.
		err := s.repo.IncrementCommentCount(ctx, blog.ID, 1)
		s.NoError(err)

		// Assert: Fetch and verify both fields were updated.
		var updatedBlog BlogModel
		err = testDB.Collection(s.collection).FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedBlog)
		s.NoError(err)
		s.Equal(int64(4), updatedBlog.CommentsCount)
		s.Equal(initialScore+CommentWeight, updatedBlog.EngagementScore)
	})

	s.Run("Decrement", func() {
		// Act: Decrement the comment count by 1.
		err := s.repo.IncrementCommentCount(ctx, blog.ID, -1)
		s.NoError(err)

		// Assert: Verify the count is back to the initial state.
		var updatedBlog BlogModel
		err = testDB.Collection(s.collection).FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedBlog)
		s.NoError(err)
		s.Equal(int64(3), updatedBlog.CommentsCount)
		s.Equal(initialScore, updatedBlog.EngagementScore)
	})
}

func (s *BlogRepositoryTestSuite) TestUpdateInteractionCounts() {
	ctx := context.Background()
	// Arrange: Create a blog with initial likes and dislikes.
	blog, _ := domain.NewBlog("Title", "Content", s.fixedAuthorID.Hex(), nil)
	blog.Likes = 20
	blog.Dislikes = 10
	err := s.repo.Create(ctx, blog)
	s.Require().NoError(err)

	// Act: Simulate a user switching from a dislike to a like.
	// This means likes should go up by 1, and dislikes should go down by 1.
	err = s.repo.UpdateInteractionCounts(ctx, blog.ID, 1, -1)
	s.NoError(err)

	// Assert: Check that both fields were updated atomically in the single operation.
	var updatedBlog BlogModel
	objID, err := primitive.ObjectIDFromHex(blog.ID)
	s.Require().NoError(err, "The blog ID from the test setup should be a valid ObjectID hex")
	err = testDB.Collection(s.collection).FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedBlog)
	s.NoError(err)
	s.Equal(int64(21), updatedBlog.Likes, "Likes count should be 21")
	s.Equal(int64(9), updatedBlog.Dislikes, "Dislikes count should be 9")
	expectedScoreChange := (1 * LikeWeight) + (-1 * DislikeWeight)
	s.Equal(expectedScoreChange, updatedBlog.EngagementScore, "Engagement score should reflect the combined change")
}
