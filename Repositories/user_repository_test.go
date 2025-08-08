package repositories_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	repositories "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserRepositorySuite defines the test suite.
type UserRepositorySuite struct {
	suite.Suite
	repository usecases.UserRepository
	collection *mongo.Collection
}

// SetupTest runs before each test. It's now responsible for initializing
// the repository with the shared testDB and ensuring the collection is clean.
func (s *UserRepositorySuite) SetupTest() {
	// The collection name for this specific suite's tests.
	collectionName := "users"

	// Initialize the repository instance using the global testDB from main_repository_test.go
	s.repository = repositories.NewMongoUserRepository(testDB, collectionName)

	// Keep a direct handle to the collection for easy verification and cleanup.
	s.collection = testDB.Collection(collectionName)
}

// TearDownTest runs after each test to ensure a clean state for the next test.
// Dropping the collection is a robust way to guarantee isolation.
func (s *UserRepositorySuite) TearDownTest() {
	err := s.collection.Drop(context.Background())
	s.Require().NoError(err, "Failed to drop test collection")
}

// TestUserRepositorySuite is the entry point for the test suite.
func TestUserRepositorySuite(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}
	suite.Run(t, new(UserRepositorySuite))
}

// --- The Actual Tests (largely unchanged) ---

func (s *UserRepositorySuite) TestCreate() {
	ctx := context.Background()
	s.Run("Create Local User", func() {
		password := "hashedpassword"
		user := &domain.User{
			Username: "testuser",
			Email:    "create@test.com",
			Password: &password,
			Role:     domain.RoleUser,
			Provider: domain.ProviderLocal,
		}

		err := s.repository.Create(ctx, user)
		s.Require().NoError(err)

		// Verify the user was actually created in the DB
		var createdUser repositories.UserMongo
		err = s.collection.FindOne(ctx, bson.M{"email": "create@test.com"}).Decode(&createdUser)
		s.Require().NoError(err)
		s.Equal("testuser", createdUser.Username)
		s.Equal(string(domain.ProviderLocal), createdUser.Provider)
		s.Require().NotNil(createdUser.Password)
		s.Equal(password, *createdUser.Password)
	})

	s.Run("Create Google User", func() {
		// Arrange
		user := &domain.User{
			Username:   "googleuser",
			Email:      "google@test.com",
			Password:   nil, // Google users have no password
			Role:       domain.RoleUser,
			Provider:   domain.ProviderGoogle,
			ProviderID: "google-id-12345",
		}

		// Act
		err := s.repository.Create(ctx, user)
		s.Require().NoError(err)
		s.NotEmpty(user.ID)

		// Assert: Verify directly from the DB
		var createdUser repositories.UserMongo
		objID, _ := primitive.ObjectIDFromHex(user.ID)
		err = s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&createdUser)
		s.Require().NoError(err)
		s.Equal("googleuser", createdUser.Username)
		s.Equal(string(domain.ProviderGoogle), createdUser.Provider)
		s.Equal("google-id-12345", createdUser.ProviderID)
		s.Nil(createdUser.Password, "Password field should be nil for Google user")
	})
}

func (s *UserRepositorySuite) TestGetByEmail() {
	// Arrange: Insert a user directly into the DB for testing
	password := "hashedpassword"
	user := &domain.User{
		Username: "getbyemail",
		Email:    "get@test.com",
		Password: &password,
		Role:     domain.RoleUser,
	}
	err := s.repository.Create(context.Background(), user)
	s.Require().NoError(err)

	s.Run("Success - User Found", func() {
		// Act
		foundUser, err := s.repository.GetByEmail(context.Background(), "get@test.com")

		// Assert
		s.Require().NoError(err)
		s.Require().NotNil(foundUser)
		s.Equal("getbyemail", foundUser.Username)
	})

	s.Run("Failure - User Not Found", func() {
		// Act
		foundUser, err := s.repository.GetByEmail(context.Background(), "notfound@test.com")

		// Assert
		s.Require().NoError(err) // GetByEmail should return (nil, nil) for not found
		s.Nil(foundUser)
	})
}

func (s *UserRepositorySuite) TestGetByID() {
	// Arrange
	user := &domain.User{Email: "getbyid@test.com", Username: "getbyid"}
	err := s.repository.Create(context.Background(), user)
	s.Require().NoError(err)

	// Retrieve the created user to get its real ID for the success case
	createdUser, err := s.repository.GetByEmail(context.Background(), user.Email)
	s.Require().NoError(err)

	s.Run("Success - User Found", func() {
		// Act
		foundUser, err := s.repository.GetByID(context.Background(), createdUser.ID)

		// Assert
		s.Require().NoError(err)
		s.NotNil(foundUser)
		s.Equal(createdUser.ID, foundUser.ID)
	})

	s.Run("Failure - User Not Found", func() {
		// Arrange: Generate a new, valid ObjectID that doesn't exist in the DB
		nonExistentID := primitive.NewObjectID().Hex()

		// Act
		foundUser, err := s.repository.GetByID(context.Background(), nonExistentID)

		// Assert
		s.Require().Error(err)
		s.Equal(domain.ErrUserNotFound, err)
		s.Nil(foundUser)
	})
}

func (s *UserRepositorySuite) TestFindUserIDsByName() {
	// Arrange: Insert a variety of users to test different search scenarios.
	usersToCreate := []*domain.User{
		{Username: "John Doe", Email: "john.doe@test.com"},
		{Username: "Jane Doe", Email: "jane.doe@test.com"},
		{Username: "johnny", Email: "johnny@test.com"},
		{Username: "john doe", Email: "johndoe.lc@test.com"}, // Lowercase version for case-insensitivity test
	}

	userIDs := make(map[string]string)
	for _, user := range usersToCreate {
		err := s.repository.Create(context.Background(), user)
		s.Require().NoError(err)

		// Fetch the created user to get their actual ID
		createdUser, err := s.repository.GetByEmail(context.Background(), user.Email)
		s.Require().NoError(err)
		userIDs[user.Username] = createdUser.ID
	}

	s.Run("Success - Single Exact Match", func() {
		// Act: Search for a unique username
		ids, err := s.repository.FindUserIDsByName(context.Background(), "Jane Doe")

		// Assert
		s.Require().NoError(err)
		s.Require().Len(ids, 1, "Should find exactly one user")
		s.Equal(userIDs["Jane Doe"], ids[0])
	})

	s.Run("Success - Multiple Case-Insensitive Matches", func() {
		// Act: Search for a name that matches multiple users due to case-insensitivity
		ids, err := s.repository.FindUserIDsByName(context.Background(), "john doe")

		// Assert
		s.Require().NoError(err)
		s.Require().Len(ids, 2, "Should find two users: 'John Doe' and 'john doe'")
		// Use ElementsMatch because the order of results from the DB is not guaranteed
		s.ElementsMatch([]string{userIDs["John Doe"], userIDs["john doe"]}, ids)
	})

	s.Run("Success - Partial Match", func() {
		// Act: Search for a partial name that should match multiple users
		ids, err := s.repository.FindUserIDsByName(context.Background(), "John")

		// Assert
		s.Require().NoError(err)
		s.Require().Len(ids, 3, "Should find 'John Doe', 'johnny', and 'john doe'")
		expectedIDs := []string{userIDs["John Doe"], userIDs["johnny"], userIDs["john doe"]}
		s.ElementsMatch(expectedIDs, ids)
	})

	s.Run("Success - No Matches Found", func() {
		// Act: Search for a name that doesn't exist
		ids, err := s.repository.FindUserIDsByName(context.Background(), "NonExistentUser")

		// Assert
		s.Require().NoError(err)
		s.Empty(ids, "Should return an empty slice for no matches")
	})

	s.Run("Success - Empty Search String Matches All", func() {
		// Act: An empty regex should match all documents
		ids, err := s.repository.FindUserIDsByName(context.Background(), "")

		// Assert
		s.Require().NoError(err)
		s.Len(ids, 4, "An empty search should return all users")
	})
}

func (s *UserRepositorySuite) TestFindByProviderID() {
	ctx := context.Background()
	// Arrange: Create a Google user to be found
	user := &domain.User{
		Username:   "provideruser",
		Email:      "provider@test.com",
		Provider:   domain.ProviderGoogle,
		ProviderID: "google-id-xyz",
	}
	err := s.repository.Create(ctx, user)
	s.Require().NoError(err)

	s.Run("Success - User Found", func() {
		// Act
		foundUser, err := s.repository.FindByProviderID(ctx, domain.ProviderGoogle, "google-id-xyz")

		// Assert
		s.NoError(err)
		s.NotNil(foundUser)
		s.Equal("provideruser", foundUser.Username)
		s.Equal(user.ID, foundUser.ID)
	})

	s.Run("Failure - User Not Found", func() {
		// Act
		foundUser, err := s.repository.FindByProviderID(ctx, domain.ProviderGoogle, "non-existent-id")

		// Assert
		s.NoError(err) // Should return (nil, nil) for not found
		s.Nil(foundUser)
	})

	s.Run("Failure - Wrong Provider", func() {
		// Act
		// Looking for a "local" provider with a Google ID should not find the user.
		foundUser, err := s.repository.FindByProviderID(ctx, domain.ProviderLocal, "google-id-xyz")

		// Assert
		s.NoError(err)
		s.Nil(foundUser)
	})
}
