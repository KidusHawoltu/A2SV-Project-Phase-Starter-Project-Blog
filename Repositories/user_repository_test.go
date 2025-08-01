package repositories_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	repositories "A2SV_Starter_Project_Blog/Repositories"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepositorySuite defines the test suite
type UserRepositorySuite struct {
	suite.Suite
	client     *mongo.Client
	db         *mongo.Database
	repository usecases.UserRepository
	collection *mongo.Collection
}

// SetupSuite runs once before all tests in the suite
func (s *UserRepositorySuite) SetupSuite() {
	// Connect to a test database
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	s.Require().NoError(err)
	s.client = client
	s.db = client.Database("g6_blog_test_db")
	s.collection = s.db.Collection("users")
	s.repository = repositories.NewMongoUserRepository(s.db, "users")
}

// TearDownSuite runs once after all tests in the suite
func (s *UserRepositorySuite) TearDownSuite() {
	s.db.Drop(context.Background())
	s.client.Disconnect(context.Background())
}

// SetupTest runs before each test
func (s *UserRepositorySuite) SetupTest() {
	// Clean up the collection before each test to ensure isolation
	s.collection.DeleteMany(context.Background(), bson.M{})
}

// TestUserRepositorySuite is the entry point for the test suite
func TestUserRepositorySuite(t *testing.T) {
	suite.Run(t, new(UserRepositorySuite))
}

// --- The Actual Tests ---

func (s *UserRepositorySuite) TestCreate() {
	user := &domain.User{
		Username: "testuser",
		Email:    "create@test.com",
		Password: "hashedpassword",
		Role:     domain.RoleUser,
	}

	err := s.repository.Create(context.Background(), user)
	s.Require().NoError(err)

	// Verify the user was actually created in the DB
	var createdUser bson.M
	err = s.collection.FindOne(context.Background(), bson.M{"email": "create@test.com"}).Decode(&createdUser)
	s.Require().NoError(err)
	s.Equal("testuser", createdUser["username"])
}

func (s *UserRepositorySuite) TestGetByEmail() {
	// Arrange: Insert a user directly into the DB for testing
	user := &domain.User{
		Username: "getbyemail",
		Email:    "get@test.com",
		Password: "hashedpassword",
		Role:     domain.RoleUser,
	}
	s.repository.Create(context.Background(), user)

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
		s.Require().NoError(err) // GetByEmail should return nil, nil for not found
		s.Nil(foundUser)
	})
}

func (s *UserRepositorySuite) TestGetByID() {
	// Arrange
	user := &domain.User{Email: "getbyid@test.com", Username: "getbyid"}
	err := s.repository.Create(context.Background(), user)
	s.Require().NoError(err)
	// Retrieve the created user to get its real ID
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
		// Act
		foundUser, err := s.repository.GetByID(context.Background(), "507f1f77bcf86cd799439011") // A valid but non-existent ObjectID

		// Assert
		s.Require().Error(err)
		s.Equal(domain.ErrUserNotFound, err)
		s.Nil(foundUser)
	})
}