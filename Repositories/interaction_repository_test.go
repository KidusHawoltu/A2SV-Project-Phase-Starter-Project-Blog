package repositories_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// InteractionRepositoryTestSuite defines the suite for the interaction repository integration tests.
type InteractionRepositoryTestSuite struct {
	suite.Suite
	repo        *InteractionRepository
	collection  *mongo.Collection
	fixedUserID primitive.ObjectID // A reusable, valid user ID for tests
	fixedBlogID primitive.ObjectID // A reusable, valid blog ID for tests
}

// SetupTest runs before each test in the suite.
func (s *InteractionRepositoryTestSuite) SetupTest() {
	collectionName := "interactions"
	// Initialize the repository instance using the global testDB.
	s.repo = NewInteractionRepository(testDB.Collection(collectionName))
	// Keep a direct handle to the collection for easy verification and cleanup.
	s.collection = testDB.Collection(collectionName)

	// Create reusable IDs for tests
	s.fixedUserID = primitive.NewObjectID()
	s.fixedBlogID = primitive.NewObjectID()
}

// TearDownTest runs after each test to ensure a clean state.
func (s *InteractionRepositoryTestSuite) TearDownTest() {
	err := s.collection.Drop(context.Background())
	s.Require().NoError(err, "Failed to drop test collection")
}

// TestInteractionRepositorySuite is the entry point for the test suite.
func TestInteractionRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}
	suite.Run(t, new(InteractionRepositoryTestSuite))
}

// --- Tests for each method of the repository ---

func (s *InteractionRepositoryTestSuite) TestCreateInteractionIndexes() {
	ctx := context.Background()

	// Act: Call the method we want to test.
	err := s.repo.CreateInteractionIndexes(ctx)
	s.Require().NoError(err, "CreateInteractionIndexes should not return an error")

	// Assert: List the indexes and verify the one we created is there.
	cursor, err := s.collection.Indexes().List(ctx)
	s.Require().NoError(err, "Failed to list collection indexes")

	var indexes []bson.M
	err = cursor.All(ctx, &indexes)
	s.Require().NoError(err, "Failed to decode indexes")

	// We expect the default '_id_' index and our custom one.
	s.Len(indexes, 2, "Expected 2 indexes in total")

	var foundOurIndex bool
	for _, idx := range indexes {
		// Skip the default index
		if idx["name"] == "_id_" {
			continue
		}

		s.Equal("user_id_1_blog_id_1", idx["name"], "Index name is incorrect")
		s.True(idx["unique"].(bool), "Index should be unique")

		keyDoc := idx["key"].(bson.M)
		s.Len(keyDoc, 2, "Compound index should have two keys")
		s.Equal(int32(1), keyDoc["user_id"], "Index should contain 'user_id'")
		s.Equal(int32(1), keyDoc["blog_id"], "Index should contain 'blog_id'")

		foundOurIndex = true
	}

	s.True(foundOurIndex, "The custom compound index was not found")
}

func (s *InteractionRepositoryTestSuite) TestCreateAndGet() {
	ctx := context.Background()
	// Arrange
	interaction := &domain.BlogInteraction{
		UserID: s.fixedUserID.Hex(),
		BlogID: s.fixedBlogID.Hex(),
		Action: domain.ActionTypeLike,
	}

	// Act: Create the interaction
	err := s.repo.Create(ctx, interaction)

	// Assert: Check creation
	s.NoError(err, "Create should succeed")
	s.NotEmpty(interaction.ID, "Interaction ID should be populated by the Create method")

	// Act: Get the interaction by its user/blog pair
	foundInteraction, err := s.repo.Get(ctx, s.fixedUserID.Hex(), s.fixedBlogID.Hex())

	// Assert: Check retrieval
	s.NoError(err, "Get should succeed")
	s.NotNil(foundInteraction)
	s.Equal(interaction.ID, foundInteraction.ID)
	s.Equal(s.fixedUserID.Hex(), foundInteraction.UserID)
	s.Equal(domain.ActionTypeLike, foundInteraction.Action)
}

func (s *InteractionRepositoryTestSuite) TestGet_NotFound() {
	ctx := context.Background()
	// Act: Try to get an interaction that doesn't exist
	foundInteraction, err := s.repo.Get(ctx, primitive.NewObjectID().Hex(), primitive.NewObjectID().Hex())

	// Assert
	s.Error(err, "Get should return an error for a non-existent interaction")
	s.ErrorIs(err, usecases.ErrNotFound, "The error should be ErrNotFound")
	s.Nil(foundInteraction, "The returned interaction should be nil")
}

func (s *InteractionRepositoryTestSuite) TestCreate_Duplicate() {
	ctx := context.Background()
	// Arrange: Explicitly create the index needed for this test.
	err := s.repo.CreateInteractionIndexes(ctx)
	s.Require().NoError(err, "Failed to create index for duplicate test")

	// Arrange: Create an initial interaction
	interaction1 := &domain.BlogInteraction{
		UserID: s.fixedUserID.Hex(),
		BlogID: s.fixedBlogID.Hex(),
		Action: domain.ActionTypeLike,
	}
	err = s.repo.Create(ctx, interaction1)
	s.Require().NoError(err)

	// Act: Attempt to create a second interaction with the exact same UserID and BlogID
	interaction2 := &domain.BlogInteraction{
		UserID: s.fixedUserID.Hex(),
		BlogID: s.fixedBlogID.Hex(),
		Action: domain.ActionTypeDislike, // Different action, but same user/blog pair
	}
	err = s.repo.Create(ctx, interaction2)

	// Assert: The operation should fail due to the unique index
	s.Error(err, "Creating a duplicate interaction should fail")
	s.ErrorIs(err, usecases.ErrConflict, "The error should be ErrConflict due to unique index violation")
}

func (s *InteractionRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()
	// Arrange: Create an initial "like" interaction
	interaction := &domain.BlogInteraction{
		UserID: s.fixedUserID.Hex(),
		BlogID: s.fixedBlogID.Hex(),
		Action: domain.ActionTypeLike,
	}
	err := s.repo.Create(ctx, interaction)
	s.Require().NoError(err)

	// Act: Modify the domain object and call update to change it to a "dislike"
	interaction.Action = domain.ActionTypeDislike
	err = s.repo.Update(ctx, interaction)
	s.NoError(err)

	// Assert: Verify the change by fetching directly from the database
	objID, err := primitive.ObjectIDFromHex(interaction.ID)
	s.Require().NoError(err, "Interaction ID from Create should be a valid ObjectID hex")

	var updatedModel InteractionModel
	filter := bson.M{"_id": objID}
	err = s.collection.FindOne(ctx, filter).Decode(&updatedModel)
	s.NoError(err)
	s.Equal(string(domain.ActionTypeDislike), updatedModel.Action, "The action should be updated to dislike")
}

func (s *InteractionRepositoryTestSuite) TestDelete() {
	ctx := context.Background()
	// Arrange: Create an interaction to delete
	interaction := &domain.BlogInteraction{
		UserID: s.fixedUserID.Hex(),
		BlogID: s.fixedBlogID.Hex(),
		Action: domain.ActionTypeLike,
	}
	err := s.repo.Create(ctx, interaction)
	s.Require().NoError(err)

	// Act: Delete the interaction by its ID
	err = s.repo.Delete(ctx, interaction.ID)
	s.NoError(err)

	// Assert: Verify it's gone by trying to Get it again
	foundInteraction, err := s.repo.Get(ctx, s.fixedUserID.Hex(), s.fixedBlogID.Hex())
	s.ErrorIs(err, usecases.ErrNotFound)
	s.Nil(foundInteraction)
}
