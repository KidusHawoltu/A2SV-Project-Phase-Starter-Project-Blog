package repositories

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
)

type tokenMongo struct {
	ID 					primitive.ObjectID 	`bson:"_id"`
	UserID			string							`bson:"user_id"`
	Type				string							`bson:"type"`
	Value				string							`bson:"value"`
	ExpiresAt		time.Time						`bson:"expires_at"`
}

func toTokenDomain(tm *tokenMongo) *domain.Token{
	return &domain.Token{
		ID: tm.ID.Hex(),
		UserID: tm.UserID,
		Type: domain.TokenType(tm.Type),
		Value: tm.Value,
		ExpiresAt: tm.ExpiresAt,
	}
}

func fromTokenDomain(t *domain.Token) (*tokenMongo,error) {
	// let mongo generate the ID if it's new
	var objectID primitive.ObjectID
	if t.ID != "" {
		var err error
		objectID , err = primitive.ObjectIDFromHex(t.ID)
		if err != nil {
			return nil, domain.ErrInvalidID
		}
	}

	return &tokenMongo{
		ID: objectID,
		UserID: t.UserID,
		Type: string(t.Type),
		Value: t.Value,
		ExpiresAt: t.ExpiresAt,
	}, nil
}

// ===== Repository Implementation =====

type mongoTokenRepository struct {
	collection *mongo.Collection
}

func NewMongoTokenRepository(db *mongo.Database, collectionName string) usecases.TokenRepository {
	return &mongoTokenRepository {
		collection: db.Collection(collectionName),
	}
}

func (r *mongoTokenRepository) Store(ctx context.Context, token *domain.Token) error {
	mongoModel, err := fromTokenDomain(token)
	if err != nil {
		return err
	}
	_, err = r.collection.InsertOne(ctx, mongoModel)
	return err
}

func (r *mongoTokenRepository) GetByValue(ctx context.Context, tokenValue string) (*domain.Token, error) {
	var mongoToken tokenMongo

	err := r.collection.FindOne(ctx, bson.M{"value": tokenValue}).Decode(&mongoToken)
	if err != nil {
		return nil, err
	}

	domainToken := toTokenDomain(&mongoToken)
	return domainToken, nil
}

func (r *mongoTokenRepository) Delete(ctx context.Context, tokenID string) error {
	id, err := primitive.ObjectIDFromHex(tokenID)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0{
		return errors.New("token not found")
	}

	return nil
}

func (r *mongoTokenRepository) DeleteByUserID(ctx context.Context, userID string, tokenType domain.TokenType) error {
	
	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID, "type": string(tokenType)})
	return err
}