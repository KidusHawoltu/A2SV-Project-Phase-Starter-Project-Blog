package repositories_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	repositories "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TokenRepositorySuite defines the test suite
type TokenRepositorySuite struct {
	suite.Suite
	client     *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
	repository usecases.TokenRepository
}

// SetupSuite runs once before all tests in the suite
func (s *TokenRepositorySuite) SetupSuite() {
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	s.Require().NoError(err, "Failed to connect to MongoDB")

	s.client = client
	s.db = client.Database("g6_blog_test_tokens") // Use a dedicated test database
	s.collection = s.db.Collection("tokens")
	s.repository = repositories.NewMongoTokenRepository(s.db, "tokens")
}

// TearDownSuite runs once after all tests are done
func (s *TokenRepositorySuite) TearDownSuite() {
	s.Require().NoError(s.db.Drop(context.Background()), "Failed to drop test database")
	s.client.Disconnect(context.Background())
}

// SetupTest runs before each individual test function
func (s *TokenRepositorySuite) SetupTest() {
	// Clean the collection to ensure test isolation
	s.Require().NoError(s.collection.Drop(context.Background()), "Failed to drop collection")
}

// TestTokenRepositorySuite is the entry point for running the test suite
func TestTokenRepositorySuite(t *testing.T) {
	suite.Run(t, new(TokenRepositorySuite))
}

// --- The Actual Tests ---

func (s *TokenRepositorySuite) TestStore() {
	ctx := context.Background()
	token := &domain.Token{
		ID:        uuid.NewString(),
		UserID:    "user-123",
		Type:      domain.TokenTypeRefresh,
		Value:     "a-unique-refresh-token",
		ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Truncate(time.Millisecond),
	}

	// Act
	err := s.repository.Store(ctx, token)
	s.Require().NoError(err)

	// Assert: Verify directly from the database
	var result bson.M
	err = s.collection.FindOne(ctx, bson.M{"value": "a-unique-refresh-token"}).Decode(&result)
	s.Require().NoError(err)
	s.Equal(token.UserID, result["user_id"])
	s.Equal(string(token.Type), result["type"])
}

func (s *TokenRepositorySuite) TestGetByValue() {
	ctx := context.Background()
	// Arrange: Pre-insert a token to find
	token := &domain.Token{
		ID:        uuid.NewString(),
		UserID:    "user-456",
		Type:      domain.TokenTypePasswordReset,
		Value:     "a-findable-token",
		ExpiresAt: time.Now().Add(15 * time.Minute).UTC().Truncate(time.Millisecond),
	}
	err := s.repository.Store(ctx, token)
	s.Require().NoError(err)

	s.Run("Success - Token Found", func() {
		// Act
		foundToken, err := s.repository.GetByValue(ctx, "a-findable-token")

		// Assert
		s.Require().NoError(err)
		s.Require().NotNil(foundToken)
		s.Equal(token.ID, foundToken.ID)
		s.Equal(token.UserID, foundToken.UserID)
		s.Equal(token.Value, foundToken.Value)
	})

	s.Run("Failure - Token Not Found", func() {
		// Act
		foundToken, err := s.repository.GetByValue(ctx, "a-non-existent-token")

		// Assert
		// The repository implementation correctly returns an error when no documents are found
		s.Require().Error(err)
		s.Equal(mongo.ErrNoDocuments, err)
		s.Nil(foundToken)
	})
}

func (s *TokenRepositorySuite) TestDelete() {
	ctx := context.Background()
	// Arrange: Store a token to be deleted
	token := &domain.Token{
		ID:    uuid.NewString(),
		Value: "token-to-delete",
	}
	err := s.repository.Store(ctx, token)
	s.Require().NoError(err)

	s.Run("Success - Delete Existing Token", func() {
		// Act
		err := s.repository.Delete(ctx, token.ID)
		s.Require().NoError(err)

		// Assert: Verify it's gone from the DB
		_, err = s.repository.GetByValue(ctx, "token-to-delete")
		s.Require().Error(err, "Token should not be found after deletion")
		s.Equal(mongo.ErrNoDocuments, err)
	})

	s.Run("Failure - Delete Non-Existent Token", func() {
		// Act
		err := s.repository.Delete(ctx, uuid.NewString())
		
		// Assert: Your implementation returns a custom error
		s.Require().Error(err)
		s.Equal("token not found", err.Error())
	})
}

func (s *TokenRepositorySuite) TestDeleteByUserID() {
	ctx := context.Background()
	// Arrange: Store multiple tokens for multiple users
	s.repository.Store(ctx, &domain.Token{ID: uuid.NewString(), UserID: "user-abc", Type: domain.TokenTypeRefresh, Value: "abc-refresh-1"})
	s.repository.Store(ctx, &domain.Token{ID: uuid.NewString(), UserID: "user-abc", Type: domain.TokenTypeRefresh, Value: "abc-refresh-2"})
	s.repository.Store(ctx, &domain.Token{ID: uuid.NewString(), UserID: "user-abc", Type: domain.TokenTypeActivation, Value: "abc-activation"})
	s.repository.Store(ctx, &domain.Token{ID: uuid.NewString(), UserID: "user-xyz", Type: domain.TokenTypeRefresh, Value: "xyz-refresh"})
	
	// Act
	err := s.repository.DeleteByUserID(ctx, "user-abc", domain.TokenTypeRefresh)
	s.Require().NoError(err)
	
	// Assert: Check the state of the database
	var count int64
	
	// Tokens for user-abc of type refresh should be gone
	count, _ = s.collection.CountDocuments(ctx, bson.M{"user_id": "user-abc", "type": domain.TokenTypeRefresh})
	s.Equal(int64(0), count, "Refresh tokens for user-abc should have been deleted")
	
	// Activation token for user-abc should still exist
	count, _ = s.collection.CountDocuments(ctx, bson.M{"user_id": "user-abc", "type": domain.TokenTypeActivation})
	s.Equal(int64(1), count, "Activation token for user-abc should not have been deleted")

	// Token for user-xyz should still exist
	count, _ = s.collection.CountDocuments(ctx, bson.M{"user_id": "user-xyz"})
	s.Equal(int64(1), count, "Token for user-xyz should not have been deleted")
}