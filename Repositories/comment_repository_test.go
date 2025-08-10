package repositories_test

import (
	"context"
	"testing"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Repositories"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CommentRepositoryTestSuite struct {
	suite.Suite
	repo        *CommentRepository
	collection  *mongo.Collection
	fixedUserID primitive.ObjectID
	fixedBlogID primitive.ObjectID
}

func (s *CommentRepositoryTestSuite) SetupTest() {
	collectionName := "blog_comments_test"
	s.repo = NewCommentRepository(testDB.Collection(collectionName))
	s.collection = testDB.Collection(collectionName)

	s.fixedUserID = primitive.NewObjectID()
	s.fixedBlogID = primitive.NewObjectID()
}

func (s *CommentRepositoryTestSuite) TearDownTest() {
	err := s.collection.Drop(context.Background())
	s.Require().NoError(err, "Failed to drop test collection")
}

func TestCommentRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}
	t.Parallel()
	suite.Run(t, new(CommentRepositoryTestSuite))
}

func (s *CommentRepositoryTestSuite) TestCreateCommentIndexes() {
	ctx := context.Background()

	// Act: Call the index creation method.
	err := s.repo.CreateCommentIndexes(ctx)
	s.Require().NoError(err, "CreateCommentIndexes should not return an error")

	// Assert: List all indexes and verify the ones we created.
	cursor, err := s.collection.Indexes().List(ctx)
	s.Require().NoError(err, "Failed to list collection indexes")

	var indexes []bson.M
	err = cursor.All(ctx, &indexes)
	s.Require().NoError(err, "Failed to decode indexes")

	// We expect the default '_id_' index plus our 2 custom ones.
	s.Len(indexes, 3, "Expected 3 indexes in total")

	// Create maps to easily check for the existence of our indexes by name.
	indexNames := make(map[string]bool)
	indexSpecs := make(map[string]bson.M)
	for _, idx := range indexes {
		name := idx["name"].(string)
		indexNames[name] = true
		indexSpecs[name] = idx
	}

	s.Run("Blog Comments Index", func() {
		indexName := "blog_id_1_parent_id_1_created_at_1"
		s.True(indexNames[indexName], "Index for fetching blog comments should exist")

		keyDoc := indexSpecs[indexName]["key"].(bson.M)
		s.Len(keyDoc, 3, "Blog comments index should be a compound index of 3 keys")
		s.Equal(int32(1), keyDoc["blog_id"], "Index should contain 'blog_id'")
		s.Equal(int32(1), keyDoc["parent_id"], "Index should contain 'parent_id'")
		s.Equal(int32(1), keyDoc["created_at"], "Index should contain 'created_at'")
	})

	s.Run("Replies Index", func() {
		indexName := "parent_id_1_created_at_1"
		s.True(indexNames[indexName], "Index for fetching replies should exist")

		keyDoc := indexSpecs[indexName]["key"].(bson.M)
		s.Len(keyDoc, 2, "Replies index should be a compound index of 2 keys")
		s.Equal(int32(1), keyDoc["parent_id"], "Index should contain 'parent_id'")
		s.Equal(int32(1), keyDoc["created_at"], "Index should contain 'created_at'")
	})
}

func (s *CommentRepositoryTestSuite) TestCreateAndGetByID() {
	ctx := context.Background()
	authorID := s.fixedUserID.Hex()
	blogID := s.fixedBlogID.Hex()

	s.Run("Top Level Comment", func() {
		// Arrange
		comment, err := domain.NewComment(blogID, authorID, "This is a top-level comment.", nil)
		s.Require().NoError(err)

		// Act: Create
		err = s.repo.Create(ctx, comment)
		s.NoError(err)
		s.NotEmpty(comment.ID)

		// Act: GetByID
		found, err := s.repo.GetByID(ctx, comment.ID)
		s.NoError(err)
		s.NotNil(found)
		s.Equal(comment.ID, found.ID)
		s.Equal(blogID, found.BlogID)
		s.Require().NotNil(found.AuthorID)
		s.Equal(authorID, *found.AuthorID)
		s.Nil(found.ParentID, "ParentID should be nil for a top-level comment")
	})

	s.Run("Reply Comment", func() {
		parentID := primitive.NewObjectID().Hex()
		comment, err := domain.NewComment(blogID, authorID, "This is a reply.", &parentID)
		s.Require().NoError(err)

		err = s.repo.Create(ctx, comment)
		s.NoError(err)

		found, err := s.repo.GetByID(ctx, comment.ID)
		s.NoError(err)
		s.Require().NotNil(found.ParentID)
		s.Equal(parentID, *found.ParentID)
	})
}

func (s *CommentRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()
	authorID := s.fixedUserID.Hex()
	blogID := s.fixedBlogID.Hex()
	comment, _ := domain.NewComment(blogID, authorID, "Original content", nil)
	err := s.repo.Create(ctx, comment)
	s.Require().NoError(err)

	// Act
	comment.Content = "Updated content"
	err = s.repo.Update(ctx, comment)
	s.NoError(err)

	// Assert
	found, err := s.repo.GetByID(ctx, comment.ID)
	s.NoError(err)
	s.Equal("Updated content", found.Content)
	s.WithinDuration(time.Now(), found.UpdatedAt, 5*time.Second) // Check if UpdatedAt was touched
}

func (s *CommentRepositoryTestSuite) TestAnonymize() {
	ctx := context.Background()
	authorID := s.fixedUserID.Hex()
	blogID := s.fixedBlogID.Hex()
	comment, _ := domain.NewComment(blogID, authorID, "Content to be deleted", nil)
	err := s.repo.Create(ctx, comment)
	s.Require().NoError(err)

	// Act
	err = s.repo.Anonymize(ctx, comment.ID)
	s.NoError(err)

	// Assert: Fetch and check fields
	found, err := s.repo.GetByID(ctx, comment.ID)
	s.NoError(err)
	s.Equal("[deleted]", found.Content)
	s.Nil(found.AuthorID, "AuthorID should be nil after anonymization")
}

func (s *CommentRepositoryTestSuite) TestFetchByBlogID_And_FetchReplies() {
	ctx := context.Background()
	// Arrange: Create a nested comment structure
	// Top Level 1
	top1, _ := domain.NewComment(s.fixedBlogID.Hex(), s.fixedUserID.Hex(), "Top 1", nil)
	s.repo.Create(ctx, top1)

	// Reply to Top 1
	reply1a, _ := domain.NewComment(s.fixedBlogID.Hex(), s.fixedUserID.Hex(), "Reply 1a", &top1.ID)
	s.repo.Create(ctx, reply1a)

	// Top Level 2
	top2, _ := domain.NewComment(s.fixedBlogID.Hex(), s.fixedUserID.Hex(), "Top 2", nil)
	s.repo.Create(ctx, top2)

	// Another blog's comment (should not be fetched)
	otherBlogID := primitive.NewObjectID()
	otherBlogComment, _ := domain.NewComment(otherBlogID.Hex(), s.fixedUserID.Hex(), "Other blog", nil)
	s.repo.Create(ctx, otherBlogComment)

	s.Run("Fetch Top Level Comments", func() {
		comments, total, err := s.repo.FetchByBlogID(ctx, s.fixedBlogID.Hex(), 1, 10)
		s.NoError(err)
		s.Equal(int64(2), total, "Should only find 2 top-level comments for this blog")
		s.Len(comments, 2)
		// Verify that we only got top-level comments
		s.Nil(comments[0].ParentID)
		s.Nil(comments[1].ParentID)
	})

	s.Run("Fetch Replies", func() {
		replies, total, err := s.repo.FetchReplies(ctx, top1.ID, 1, 10)
		s.NoError(err)
		s.Equal(int64(1), total, "Should find exactly 1 reply for top1")
		s.Len(replies, 1)
		s.Require().NotNil(replies[0].ParentID)
		s.Equal(top1.ID, *replies[0].ParentID)
	})
}

func (s *CommentRepositoryTestSuite) TestIncrementReplyCount() {
	ctx := context.Background()
	// Arrange: Create a parent comment
	parent, _ := domain.NewComment(s.fixedBlogID.Hex(), s.fixedUserID.Hex(), "Parent comment", nil)
	err := s.repo.Create(ctx, parent)
	s.Require().NoError(err)

	// Act: Increment twice
	err = s.repo.IncrementReplyCount(ctx, parent.ID, 1)
	s.NoError(err)
	err = s.repo.IncrementReplyCount(ctx, parent.ID, 1)
	s.NoError(err)

	// Assert
	found, err := s.repo.GetByID(ctx, parent.ID)
	s.NoError(err)
	s.Equal(int64(2), found.ReplyCount, "ReplyCount should be 2")

	// Act: Decrement
	err = s.repo.IncrementReplyCount(ctx, parent.ID, -1)
	s.NoError(err)

	// Assert
	found, err = s.repo.GetByID(ctx, parent.ID)
	s.NoError(err)
	s.Equal(int64(1), found.ReplyCount, "ReplyCount should be 1")
}
