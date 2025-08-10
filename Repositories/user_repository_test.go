package repositories_test

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	repositories "A2SV_Starter_Project_Blog/Repositories"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserRepositorySuite defines the test suite.
type UserRepositorySuite struct {
	suite.Suite
	repository *repositories.MongoUserRepository
	collection *mongo.Collection
}

// SetupTest runs before each test. It's now responsible for initializing
// the repository with the shared testDB and ensuring the collection is clean.
func (s *UserRepositorySuite) SetupTest() {
	collectionName := "users"
	s.repository = repositories.NewMongoUserRepository(testDB, collectionName)
	s.collection = testDB.Collection(collectionName)
}

// TearDownTest runs after each test to ensure a clean state for the next test.
func (s *UserRepositorySuite) TearDownTest() {
	err := s.collection.Drop(context.Background())
	s.Require().NoError(err, "Failed to drop test collection")
}

// TestUserRepositorySuite is the entry point for the test suite.
func TestUserRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}
	t.Parallel()
	suite.Run(t, new(UserRepositorySuite))
}

func (s *UserRepositorySuite) TestCreateUserIndexes() {
	ctx := context.Background()

	// Act: Call the index creation method.
	err := s.repository.CreateUserIndexes(ctx)
	s.Require().NoError(err, "CreateUserIndexes should not return an error")

	// Assert: List all indexes and verify them.
	cursor, err := s.collection.Indexes().List(ctx)
	s.Require().NoError(err, "Failed to list collection indexes")

	var indexes []bson.M
	err = cursor.All(ctx, &indexes)
	s.Require().NoError(err, "Failed to decode indexes")

	// We expect the default '_id_' index plus our 4 custom ones.
	s.Len(indexes, 5, "Expected 5 indexes in total")

	indexNames := make(map[string]bool)
	indexSpecs := make(map[string]bson.M)
	for _, idx := range indexes {
		name := idx["name"].(string)
		indexNames[name] = true
		indexSpecs[name] = idx
	}

	s.Run("Email Unique Index", func() {
		indexName := "email_1"
		s.True(indexNames[indexName], "Email unique index should exist")
		s.True(indexSpecs[indexName]["unique"].(bool), "Email index should be unique")
		// Also verify the case-insensitive collation
		collation, ok := indexSpecs[indexName]["collation"].(bson.M)
		s.True(ok, "Collation should exist for email index")
		s.Equal("en", collation["locale"])
		s.Equal(int32(2), collation["strength"])
	})

	s.Run("Username Unique Index", func() {
		indexName := "username_1"
		s.True(indexNames[indexName], "Username unique index should exist")
		s.True(indexSpecs[indexName]["unique"].(bool), "Username index should be unique")
	})

	s.Run("Provider Unique Index", func() {
		indexName := "provider_1_providerId_1"
		s.True(indexNames[indexName], "Provider unique index should exist")
		s.True(indexSpecs[indexName]["unique"].(bool), "Provider index should be unique")
		s.NotNil(indexSpecs[indexName]["partialFilterExpression"], "Provider index should have a partial filter")
	})

	s.Run("Admin Filter Index", func() {
		indexName := "role_1_isActive_1_createdAt_-1"
		s.True(indexNames[indexName], "Admin filter index should exist")
	})
}

// --- The Actual Tests ---

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
		nonExistentID := primitive.NewObjectID().Hex()
		foundUser, err := s.repository.GetByID(context.Background(), nonExistentID)

		s.Require().Error(err)
		s.Equal(domain.ErrUserNotFound, err)
		s.Nil(foundUser)
	})
}

func (s *UserRepositorySuite) TestFindUserIDsByName() {
	usersToCreate := []*domain.User{
		{Username: "John Doe", Email: "john.doe@test.com"},
		{Username: "Jane Doe", Email: "jane.doe@test.com"},
		{Username: "johnny", Email: "johnny@test.com"},
		{Username: "john doe", Email: "johndoe.lc@test.com"},
	}

	userIDs := make(map[string]string)
	for _, user := range usersToCreate {
		err := s.repository.Create(context.Background(), user)
		s.Require().NoError(err)
		createdUser, err := s.repository.GetByEmail(context.Background(), user.Email)
		s.Require().NoError(err)
		userIDs[user.Username] = createdUser.ID
	}

	s.Run("Success - Single Exact Match", func() {
		ids, err := s.repository.FindUserIDsByName(context.Background(), "Jane Doe")
		s.Require().NoError(err)
		s.Require().Len(ids, 1, "Should find exactly one user")
		s.Equal(userIDs["Jane Doe"], ids[0])
	})

	s.Run("Success - Multiple Case-Insensitive Matches", func() {
		ids, err := s.repository.FindUserIDsByName(context.Background(), "john doe")
		s.Require().NoError(err)
		s.Require().Len(ids, 2, "Should find two users: 'John Doe' and 'john doe'")
		s.ElementsMatch([]string{userIDs["John Doe"], userIDs["john doe"]}, ids)
	})

	s.Run("Success - Partial Match", func() {
		ids, err := s.repository.FindUserIDsByName(context.Background(), "John")
		s.Require().NoError(err)
		s.Require().Len(ids, 3, "Should find 'John Doe', 'johnny', and 'john doe'")
		expectedIDs := []string{userIDs["John Doe"], userIDs["johnny"], userIDs["john doe"]}
		s.ElementsMatch(expectedIDs, ids)
	})

	s.Run("Success - No Matches Found", func() {
		ids, err := s.repository.FindUserIDsByName(context.Background(), "NonExistentUser")
		s.Require().NoError(err)
		s.Empty(ids, "Should return an empty slice for no matches")
	})

	s.Run("Success - Empty Search String Matches All", func() {
		ids, err := s.repository.FindUserIDsByName(context.Background(), "")
		s.Require().NoError(err)
		s.Len(ids, 4, "An empty search should return all users")
	})
}

func (s *UserRepositorySuite) TestFindByProviderID() {
	ctx := context.Background()
	user := &domain.User{
		Username:   "provideruser",
		Email:      "provider@test.com",
		Provider:   domain.ProviderGoogle,
		ProviderID: "google-id-xyz",
	}
	err := s.repository.Create(ctx, user)
	s.Require().NoError(err)

	s.Run("Success - User Found", func() {
		foundUser, err := s.repository.FindByProviderID(ctx, domain.ProviderGoogle, "google-id-xyz")
		s.NoError(err)
		s.NotNil(foundUser)
		s.Equal("provideruser", foundUser.Username)
		s.Equal(user.ID, foundUser.ID)
	})

	s.Run("Failure - User Not Found", func() {
		foundUser, err := s.repository.FindByProviderID(ctx, domain.ProviderGoogle, "non-existent-id")
		s.NoError(err)
		s.Nil(foundUser)
	})

	s.Run("Failure - Wrong Provider", func() {
		foundUser, err := s.repository.FindByProviderID(ctx, domain.ProviderLocal, "google-id-xyz")
		s.NoError(err)
		s.Nil(foundUser)
	})
}

func (s *UserRepositorySuite) TestSearchAndFilter() {
	ctx := context.Background()
	timeNow := time.Now()
	timeYesterday := timeNow.AddDate(0, 0, -1)
	timeTwoDaysAgo := timeNow.AddDate(0, 0, -2)

	usersToCreate := []repositories.UserMongo{
		{ID: primitive.NewObjectID(), Username: "AliceAdmin", Email: "alice@test.com", Role: domain.RoleAdmin, IsActive: true, Provider: string(domain.ProviderLocal), CreatedAt: timeNow},
		{ID: primitive.NewObjectID(), Username: "BobUser", Email: "bob@test.com", Role: domain.RoleUser, IsActive: true, Provider: string(domain.ProviderLocal), CreatedAt: timeYesterday},
		{ID: primitive.NewObjectID(), Username: "CharlieGoogle", Email: "charlie@test.com", Role: domain.RoleUser, IsActive: true, Provider: string(domain.ProviderGoogle), CreatedAt: timeTwoDaysAgo},
		{ID: primitive.NewObjectID(), Username: "DianaInactive", Email: "diana@test.com", Role: domain.RoleUser, IsActive: false, Provider: string(domain.ProviderLocal), CreatedAt: timeTwoDaysAgo},
	}

	var docs []interface{}
	for _, u := range usersToCreate {
		docs = append(docs, u)
	}
	_, err := s.collection.InsertMany(ctx, docs)
	s.Require().NoError(err)

	s.Run("Filter by Role - Admin", func() {
		roleAdmin := domain.RoleAdmin
		options := domain.UserSearchFilterOptions{Role: &roleAdmin}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(1), total)
		s.Len(users, 1)
		s.Equal("AliceAdmin", users[0].Username)
	})

	s.Run("Filter by IsActive - false", func() {
		isActiveFalse := false
		options := domain.UserSearchFilterOptions{IsActive: &isActiveFalse}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(1), total)
		s.Len(users, 1)
		s.Equal("DianaInactive", users[0].Username)
	})

	s.Run("Search by Username - Partial Case-Insensitive", func() {
		usernameSearch := "ali"
		options := domain.UserSearchFilterOptions{Username: &usernameSearch}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(1), total)
		s.Len(users, 1)
		s.Equal("AliceAdmin", users[0].Username)
	})

	s.Run("Filter by Provider - Google", func() {
		providerGoogle := domain.ProviderGoogle
		options := domain.UserSearchFilterOptions{Provider: &providerGoogle}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(1), total)
		s.Len(users, 1)
		s.Equal("CharlieGoogle", users[0].Username)
	})

	s.Run("Filter by Date Range", func() {
		startDate := timeTwoDaysAgo.Add(-1 * time.Hour)
		endDate := timeTwoDaysAgo.Add(1 * time.Hour)
		options := domain.UserSearchFilterOptions{StartDate: &startDate, EndDate: &endDate}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(2), total)
		s.Len(users, 2)
		s.ElementsMatch([]string{"CharlieGoogle", "DianaInactive"}, []string{users[0].Username, users[1].Username})
	})

	s.Run("Combine Filters with AND logic - Admin and Active", func() {
		roleAdmin := domain.RoleAdmin
		isActiveTrue := true
		options := domain.UserSearchFilterOptions{
			Role:        &roleAdmin,
			IsActive:    &isActiveTrue,
			GlobalLogic: domain.GlobalLogicAND,
		}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(1), total, "Should only find Alice who is both admin and active")
		s.Len(users, 1)
		s.Equal("AliceAdmin", users[0].Username)
	})

	s.Run("Combine Filters with OR logic - Admin or Google Provider", func() {
		roleAdmin := domain.RoleAdmin
		providerGoogle := domain.ProviderGoogle
		options := domain.UserSearchFilterOptions{
			Role:        &roleAdmin,
			Provider:    &providerGoogle,
			GlobalLogic: domain.GlobalLogicOR,
		}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(2), total, "Should find Alice (admin) and Charlie (google)")
		s.Len(users, 2)
		s.ElementsMatch([]string{"AliceAdmin", "CharlieGoogle"}, []string{users[0].Username, users[1].Username})
	})

	s.Run("Pagination - Get first page of 2", func() {
		options := domain.UserSearchFilterOptions{
			Page:      1,
			Limit:     2,
			SortBy:    "createdAt",
			SortOrder: domain.SortOrderDESC,
		}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(4), total, "Total should be all users")
		s.Len(users, 2)
		s.Equal("AliceAdmin", users[0].Username)
		s.Equal("BobUser", users[1].Username)
	})

	s.Run("Pagination - Get second page of 2", func() {
		options := domain.UserSearchFilterOptions{
			Page:      2,
			Limit:     2,
			SortBy:    "createdAt",
			SortOrder: domain.SortOrderDESC,
		}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(4), total)
		s.Len(users, 2)
		s.ElementsMatch([]string{"CharlieGoogle", "DianaInactive"}, []string{users[0].Username, users[1].Username})
	})

	s.Run("No Filters - Returns all users", func() {
		options := domain.UserSearchFilterOptions{}
		users, total, err := s.repository.SearchAndFilter(ctx, options)
		s.NoError(err)
		s.Equal(int64(4), total)
		s.Len(users, 4)
	})
}
