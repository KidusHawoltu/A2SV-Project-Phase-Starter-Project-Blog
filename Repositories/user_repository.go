package repositories

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserMongo is the data model for the database.
type UserMongo struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	Username       string             `bson:"username"`
	Email          string             `bson:"email"`
	IsActive       bool               `bson:"isActive"`
	Password       *string            `bson:"password"`
	Role           domain.Role        `bson:"role"`
	Bio            string             `bson:"bio,omitempty"`
	ProfilePicture string             `bson:"profilePicture,omitempty"`
	Provider       string             `bson:"provider"`
	ProviderID     string             `bson:"providerId,omitempty"`
	CreatedAt      time.Time          `bson:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt"`
}

// Mappers
func toUserDomain(u UserMongo) *domain.User {
	return &domain.User{
		ID:             u.ID.Hex(),
		Username:       u.Username,
		Email:          u.Email,
		Password:       u.Password,
		Role:           u.Role,
		Bio:            u.Bio,
		IsActive:       u.IsActive,
		ProfilePicture: u.ProfilePicture,
		Provider:       domain.AuthProvider(u.Provider),
		ProviderID:     u.ProviderID,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func fromUserDomain(u domain.User) UserMongo {
	var objectID primitive.ObjectID
	if id, err := primitive.ObjectIDFromHex(u.ID); err == nil {
		objectID = id
	}
	return UserMongo{
		ID:             objectID,
		Username:       u.Username,
		Email:          u.Email,
		Password:       u.Password,
		Role:           u.Role,
		Bio:            u.Bio,
		IsActive:       u.IsActive,
		ProfilePicture: u.ProfilePicture,
		Provider:       string(u.Provider),
		ProviderID:     u.ProviderID,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository creates a new user repository instance.
func NewMongoUserRepository(db *mongo.Database, collectionName string) *MongoUserRepository {
	return &MongoUserRepository{
		collection: db.Collection(collectionName),
	}
}

func (r *MongoUserRepository) CreateUserIndexes(ctx context.Context) error {
	// Unique index for email to ensure no duplicates and for fast login lookups.
	// The `collation` option makes the index case-insensitive.
	emailIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetCollation(&options.Collation{
			Locale:   "en",
			Strength: 2, // 1 for base chars, 2 for accents, 3 for case/variants
		}),
	}

	// Unique index for username.
	usernameIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	// Unique compound index for OAuth providers.
	providerIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "provider", Value: 1},
			{Key: "providerId", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{
			"providerId": bson.M{"$type": "string"}, // Only apply unique constraint if providerId exists
		}),
	}

	// General purpose index for admin filtering and sorting.
	adminFilterIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "role", Value: 1},
			{Key: "isActive", Value: 1},
			{Key: "createdAt", Value: -1},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		emailIndex,
		usernameIndex,
		providerIndex,
		adminFilterIndex,
	})
	return err
}

func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) error {
	mongoModel := fromUserDomain(*user)
	now := time.Now()
	mongoModel.CreatedAt = now
	mongoModel.UpdatedAt = now
	mongoModel.ID = primitive.NewObjectID()

	_, err := r.collection.InsertOne(ctx, mongoModel)
	if err != nil {
		return err
	}

	// Update the domain object with the generated ID
	user.ID = mongoModel.ID.Hex()
	return nil
}

func (r *MongoUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var mongoModel UserMongo
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&mongoModel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not an application error, just means no user found
		}
		return nil, err
	}
	return toUserDomain(mongoModel), nil
}

// GetByUsername fetches a single user by their username.
func (r *MongoUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var mongoModel UserMongo
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&mongoModel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	fmt.Println("r", mongoModel.IsActive)
	return toUserDomain(mongoModel), nil
}

func (r *MongoUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrUserNotFound // Invalid ID format
	}

	var mongoModel UserMongo
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&mongoModel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return toUserDomain(mongoModel), nil
}

func (r *MongoUserRepository) Update(ctx context.Context, user *domain.User) error {
	objectID, err := primitive.ObjectIDFromHex(user.ID)
	if err != nil {
		return domain.ErrUserNotFound
	}
	user.UpdatedAt = time.Now()
	mongoModel := fromUserDomain(*user)

	update := bson.M{"$set": mongoModel}

	res, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return usecases.ErrNotFound
	}
	return nil
}

func (r *MongoUserRepository) FindUserIDsByName(ctx context.Context, authorName string) ([]string, error) {
	filter := bson.M{"username": bson.M{"$regex": authorName, "$options": "i"}}
	projection := options.Find().SetProjection(bson.M{"_id": 1})

	cursor, err := r.collection.Find(ctx, filter, projection)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var result struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		ids = append(ids, result.ID.Hex())
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

// FindByProviderID finds a user by their external provider ID (e.g., from Google).
func (r *MongoUserRepository) FindByProviderID(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error) {
	filter := bson.M{
		"provider":   string(provider),
		"providerId": providerID,
	}

	var mongoModel UserMongo
	err := r.collection.FindOne(ctx, filter).Decode(&mongoModel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Return (nil, nil) when no user is found
		}
		return nil, err
	}
	return toUserDomain(mongoModel), nil
}

// SearchAndFilter retrieves a paginated and filtered list of users based on the provided options.
func (r *MongoUserRepository) SearchAndFilter(ctx context.Context, opts domain.UserSearchFilterOptions) ([]*domain.User, int64, error) {
	filter := buildUserFilter(opts)

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find()
	findOptions.SetLimit(opts.Limit)
	findOptions.SetSkip((opts.Page - 1) * opts.Limit)

	sortValue := -1 // Default to DESC
	if opts.SortOrder == domain.SortOrderASC {
		sortValue = 1
	}

	var sortDoc bson.D
	switch opts.SortBy {
	case "username":
		sortDoc = bson.D{{Key: "username", Value: sortValue}}
	case "email":
		sortDoc = bson.D{{Key: "email", Value: sortValue}}
	default: // "createdAt" or any other value defaults to sorting by creation date.
		sortDoc = bson.D{{Key: "createdAt", Value: sortValue}}
	}
	findOptions.SetSort(sortDoc)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []*domain.User
	for cursor.Next(ctx) {
		var model UserMongo
		if err := cursor.Decode(&model); err != nil {
			return nil, 0, err
		}
		users = append(users, toUserDomain(model))
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// buildUserFilter is a helper function that constructs the MongoDB filter document
// from the search options. It is used by the SearchAndFilter method.
func buildUserFilter(opts domain.UserSearchFilterOptions) bson.M {
	var conditions []bson.M

	if opts.Username != nil && *opts.Username != "" {
		conditions = append(conditions, bson.M{"username": bson.M{"$regex": primitive.Regex{Pattern: *opts.Username, Options: "i"}}})
	}
	if opts.Email != nil && *opts.Email != "" {
		conditions = append(conditions, bson.M{"email": bson.M{"$regex": primitive.Regex{Pattern: *opts.Email, Options: "i"}}})
	}
	if opts.Role != nil {
		conditions = append(conditions, bson.M{"role": *opts.Role})
	}
	if opts.IsActive != nil {
		conditions = append(conditions, bson.M{"isActive": *opts.IsActive})
	}
	if opts.Provider != nil {
		conditions = append(conditions, bson.M{"provider": *opts.Provider})
	}

	dateFilter := bson.M{}
	if opts.StartDate != nil {
		dateFilter["$gte"] = *opts.StartDate
	}
	if opts.EndDate != nil {
		dateFilter["$lte"] = *opts.EndDate
	}
	if len(dateFilter) > 0 {
		conditions = append(conditions, bson.M{"createdAt": dateFilter})
	}

	if len(conditions) == 0 {
		return bson.M{}
	}

	operator := "$and" // Default to AND logic
	if opts.GlobalLogic == domain.GlobalLogicOR {
		operator = "$or"
	}
	return bson.M{operator: conditions}
}
