package repositories

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// userMongo is the data model for the database.
type userMongo struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	Username       string             `bson:"username"`
	Email          string             `bson:"email"`
	Password       string             `bson:"password"`
	Role           domain.Role        `bson:"role"`
	Bio            string             `bson:"bio,omitempty"`
	ProfilePicture string             `bson:"profilePicture,omitempty"`
	CreatedAt      time.Time          `bson:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt"`
}

// Mappers
func toUserDomain(u userMongo) *domain.User {
	return &domain.User{
		ID:             u.ID.Hex(),
		Username:       u.Username,
		Email:          u.Email,
		Password:       u.Password,
		Role:           u.Role,
		Bio:            u.Bio,
		ProfilePicture: u.ProfilePicture,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func fromUserDomain(u domain.User) userMongo {
	var objectID primitive.ObjectID
	if id, err := primitive.ObjectIDFromHex(u.ID); err == nil {
		objectID = id
	}
	return userMongo{
		ID:             objectID,
		Username:       u.Username,
		Email:          u.Email,
		Password:       u.Password,
		Role:           u.Role,
		Bio:            u.Bio,
		ProfilePicture: u.ProfilePicture,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

type mongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository creates a new user repository instance.
func NewMongoUserRepository(db *mongo.Database, collectionName string) usecases.UserRepository {
	return &mongoUserRepository{
		collection: db.Collection(collectionName),
	}
}

func (r *mongoUserRepository) Create(ctx context.Context, user *domain.User) error {
	mongoModel := fromUserDomain(*user)
	now := time.Now()
	mongoModel.CreatedAt = now
	mongoModel.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, mongoModel)
	return err
}

func (r *mongoUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var mongoModel userMongo
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
func (r *mongoUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var mongoModel userMongo
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&mongoModel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return toUserDomain(mongoModel), nil
}

func (r *mongoUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrUserNotFound // Invalid ID format
	}

	var mongoModel userMongo
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&mongoModel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return toUserDomain(mongoModel), nil
}