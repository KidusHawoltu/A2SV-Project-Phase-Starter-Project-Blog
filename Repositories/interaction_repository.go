package repositories

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InteractionModel is the struct that represents how an interaction is stored in MongoDB.
type InteractionModel struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"user_id"`
	BlogID    primitive.ObjectID `bson:"blog_id"`
	Action    string             `bson:"action"` // Storing ActionType as a string
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// InteractionRepository implements the domain.IInteractionRepository interface.
type InteractionRepository struct {
	collection *mongo.Collection
}

// NewInteractionRepository is the constructor for the interaction repository.
func NewInteractionRepository(col *mongo.Collection) *InteractionRepository {
	return &InteractionRepository{
		collection: col,
	}
}

// --- Mapper Functions ---

func toInteractionDomain(model *InteractionModel) *domain.BlogInteraction {
	return &domain.BlogInteraction{
		ID:        model.ID.Hex(),
		UserID:    model.UserID.Hex(),
		BlogID:    model.BlogID.Hex(),
		Action:    domain.ActionType(model.Action),
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func fromInteractionDomain(interaction *domain.BlogInteraction) (*InteractionModel, error) {
	userID, err := primitive.ObjectIDFromHex(interaction.UserID)
	if err != nil {
		return nil, usecases.ErrInternal // Invalid ID is an internal error
	}
	blogID, err := primitive.ObjectIDFromHex(interaction.BlogID)
	if err != nil {
		return nil, usecases.ErrInternal
	}
	return &InteractionModel{
		UserID:    userID,
		BlogID:    blogID,
		Action:    string(interaction.Action),
		CreatedAt: interaction.CreatedAt,
		UpdatedAt: interaction.UpdatedAt,
	}, nil
}

func (r *InteractionRepository) CreateInteractionIndexes(ctx context.Context) error {
	// A unique, compound index on user_id and blog_id.
	// This serves two purposes:
	// 1. Ensures a user can only have one interaction document per blog (e.g., can't "like" a blog twice).
	// 2. Makes lookups by user and blog ID (in the Get method) extremely fast.
	uniqueInteractionIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1}, // 1 for ascending
			{Key: "blog_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	// Create the index. This command is idempotent. Using CreateMany for consistency, but CreateOne is fine too.
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{uniqueInteractionIndex})
	return err
}

// --- Interface Implementations ---

func (r *InteractionRepository) Create(ctx context.Context, interaction *domain.BlogInteraction) error {
	model, err := fromInteractionDomain(interaction)
	if err != nil {
		return err
	}
	model.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	model.CreatedAt = now
	model.UpdatedAt = now

	_, err = r.collection.InsertOne(ctx, model)
	if err != nil {
		// This handles the unique index constraint violation.
		if mongo.IsDuplicateKeyError(err) {
			return usecases.ErrConflict
		}
		return err
	}
	interaction.ID = model.ID.Hex()
	return nil
}

func (r *InteractionRepository) Get(ctx context.Context, userID, blogID string) (*domain.BlogInteraction, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, usecases.ErrNotFound // Invalid ID can't be found
	}
	blogObjID, err := primitive.ObjectIDFromHex(blogID)
	if err != nil {
		return nil, usecases.ErrNotFound
	}

	filter := bson.M{"user_id": userObjID, "blog_id": blogObjID}
	var model InteractionModel
	err = r.collection.FindOne(ctx, filter).Decode(&model)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, usecases.ErrNotFound
		}
		return nil, err
	}
	return toInteractionDomain(&model), nil
}

func (r *InteractionRepository) GetByID(ctx context.Context, id string) (*domain.BlogInteraction, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, usecases.ErrNotFound
	}

	var model InteractionModel
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&model)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, usecases.ErrNotFound
		}
		return nil, err
	}

	return toInteractionDomain(&model), nil
}

func (r *InteractionRepository) Update(ctx context.Context, interaction *domain.BlogInteraction) error {
	objID, err := primitive.ObjectIDFromHex(interaction.ID)
	if err != nil {
		return usecases.ErrNotFound
	}

	update := bson.M{
		"$set": bson.M{
			"action":     string(interaction.Action),
			"updated_at": time.Now().UTC(),
		},
	}

	res, err := r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return usecases.ErrNotFound
	}
	return nil
}

func (r *InteractionRepository) Delete(ctx context.Context, interactionID string) error {
	objID, err := primitive.ObjectIDFromHex(interactionID)
	if err != nil {
		return usecases.ErrNotFound
	}

	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return usecases.ErrNotFound
	}
	return nil
}
