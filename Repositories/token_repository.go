package repositories

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
)

type tokenMongo struct {
	ID        primitive.ObjectID `bson:"_id"`
	UserID    string             `bson:"user_id"`
	Type      string             `bson:"type"`
	Value     string             `bson:"value"`
	ExpiresAt time.Time          `bson:"expires_at"`
}

func toTokenDomain(tm *tokenMongo) *domain.Token {
	return &domain.Token{
		ID:        tm.ID.Hex(),
		UserID:    tm.UserID,
		Type:      domain.TokenType(tm.Type),
		Value:     tm.Value,
		ExpiresAt: tm.ExpiresAt,
	}
}

func fromTokenDomain(t *domain.Token) (*tokenMongo, error) {
	// let mongo generate the ID if it's new
	var objectID primitive.ObjectID
	if t.ID != "" {
		var err error
		objectID, err = primitive.ObjectIDFromHex(t.ID)
		if err != nil {
			return nil, domain.ErrInvalidID
		}
	}
	t.ID = objectID.Hex()

	return &tokenMongo{
		ID:        objectID,
		UserID:    t.UserID,
		Type:      string(t.Type),
		Value:     t.Value,
		ExpiresAt: t.ExpiresAt,
	}, nil
}

// ===== Repository Implementation =====

type MongoTokenRepository struct {
	collection *mongo.Collection
}

func NewMongoTokenRepository(db *mongo.Database, collectionName string) *MongoTokenRepository {
	return &MongoTokenRepository{
		collection: db.Collection(collectionName),
	}
}

func (r *MongoTokenRepository) CreateTokenIndexes(ctx context.Context) error {
	// Index for GetByValue: unique index on 'value'
	// Ensures fast lookups and that token values are unique.
	valueIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "value", Value: 1}}, // 1 for ascending
		Options: options.Index().SetUnique(true),
	}

	// Index for DeleteByUserID: compound index on 'user_id' and 'type'
	// Speeds up deletion of all tokens for a specific user and type.
	userTypeIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "type", Value: 1}},
		Options: nil,
	}

	// TTL Index for auto-deletion of expired tokens.
	// MongoDB will automatically delete documents when their 'expires_at' time is reached.
	// SetExpireAfterSeconds(0) means it will use the timestamp from the 'expires_at' field.
	ttlIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0).
			SetPartialFilterExpression(bson.M{
				// Only include documents in this index where the 'type' field
				// is NOT equal to "access_token".
				"type": bson.M{"$in": []string{
					string(domain.TokenTypeRefresh),
					string(domain.TokenTypeActivation),
					string(domain.TokenTypePasswordReset),
				},
				},
			}),
	}

	// Create the indexes
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		valueIndex,
		userTypeIndex,
		ttlIndex,
	})

	return err
}

func (r *MongoTokenRepository) Store(ctx context.Context, token *domain.Token) error {
	mongoModel, err := fromTokenDomain(token)
	if err != nil {
		return err
	}
	_, err = r.collection.InsertOne(ctx, mongoModel)
	return err
}

func (r *MongoTokenRepository) GetByValue(ctx context.Context, tokenValue string) (*domain.Token, error) {
	var mongoToken tokenMongo

	err := r.collection.FindOne(ctx, bson.M{"value": tokenValue}).Decode(&mongoToken)
	if err != nil {
		return nil, err
	}

	domainToken := toTokenDomain(&mongoToken)
	return domainToken, nil
}

func (r *MongoTokenRepository) GetByID(ctx context.Context, tokenID string) (*domain.Token, error) {
	id, err := primitive.ObjectIDFromHex(tokenID)
	if err != nil {
		return nil, domain.ErrInvalidID
	}

	var mongoToken tokenMongo
	err = r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&mongoToken)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return toTokenDomain(&mongoToken), nil
}

func (r *MongoTokenRepository) Delete(ctx context.Context, tokenID string) error {
	id, err := primitive.ObjectIDFromHex(tokenID)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("token not found")
	}

	return nil
}

func (r *MongoTokenRepository) DeleteByUserID(ctx context.Context, userID string, tokenType domain.TokenType) error {

	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID, "type": string(tokenType)})
	return err
}
