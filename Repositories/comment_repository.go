package repositories

import (
	"context"
	"errors"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CommentModel is the struct that represents how a comment is stored in MongoDB.
type CommentModel struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty"`
	BlogID     primitive.ObjectID  `bson:"blog_id"`
	AuthorID   *primitive.ObjectID `bson:"author_id,omitempty"` // Pointer to allow for null/unset (anonymization)
	ParentID   *primitive.ObjectID `bson:"parent_id,omitempty"` // Pointer for top-level vs. reply distinction
	Content    string              `bson:"content"`
	ReplyCount int64               `bson:"reply_count"`
	CreatedAt  time.Time           `bson:"created_at"`
	UpdatedAt  time.Time           `bson:"updated_at"`
}

type commentRepository struct {
	collection *mongo.Collection
}

func NewCommentRepository(col *mongo.Collection) domain.ICommentRepository {
	return &commentRepository{
		collection: col,
	}
}

func toCommentDomain(model *CommentModel) *domain.Comment {
	comment := &domain.Comment{
		ID:         model.ID.Hex(),
		BlogID:     model.BlogID.Hex(),
		Content:    model.Content,
		ReplyCount: model.ReplyCount,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
	if model.AuthorID != nil {
		authorID := model.AuthorID.Hex()
		comment.AuthorID = &authorID
	}
	if model.ParentID != nil {
		parentID := model.ParentID.Hex()
		comment.ParentID = &parentID
	}
	return comment
}

func fromCommentDomain(comment *domain.Comment) (*CommentModel, error) {
	blogID, err := primitive.ObjectIDFromHex(comment.BlogID)
	if err != nil {
		return nil, usecases.ErrInternal
	}

	model := &CommentModel{
		Content:    comment.Content,
		BlogID:     blogID,
		ReplyCount: comment.ReplyCount,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  comment.UpdatedAt,
	}

	if comment.AuthorID != nil {
		authorID, err := primitive.ObjectIDFromHex(*comment.AuthorID)
		if err != nil {
			return nil, usecases.ErrInternal
		}
		model.AuthorID = &authorID
	}
	if comment.ParentID != nil {
		parentID, err := primitive.ObjectIDFromHex(*comment.ParentID)
		if err != nil {
			return nil, usecases.ErrInternal
		}
		model.ParentID = &parentID
	}
	return model, nil
}

func (r *commentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	model, err := fromCommentDomain(comment)
	if err != nil {
		return err
	}
	model.ID = primitive.NewObjectID()
	now := time.Now().UTC()
	model.CreatedAt = now
	model.UpdatedAt = now

	_, err = r.collection.InsertOne(ctx, model)
	if err != nil {
		return err
	}
	comment.ID = model.ID.Hex()
	return nil
}

func (r *commentRepository) GetByID(ctx context.Context, commentID string) (*domain.Comment, error) {
	objID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return nil, usecases.ErrInternal
	}

	var model CommentModel
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&model)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, usecases.ErrNotFound
		}
		return nil, err
	}
	return toCommentDomain(&model), nil
}

func (r *commentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	objID, err := primitive.ObjectIDFromHex(comment.ID)
	if err != nil {
		return usecases.ErrInternal
	}
	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"content":    comment.Content,
			"updated_at": time.Now().UTC(),
		},
	}
	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return usecases.ErrNotFound
	}
	return nil
}

func (r *commentRepository) Anonymize(ctx context.Context, commentID string) error {
	objID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return usecases.ErrInternal
	}
	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"content":    "[deleted]",
			"updated_at": time.Now().UTC(),
		},
		"$unset": bson.M{
			"author_id": "", // $unset removes the field from the document
		},
	}
	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return usecases.ErrNotFound
	}
	return nil
}

func (r *commentRepository) FetchByBlogID(ctx context.Context, blogID string, page, limit int64) ([]*domain.Comment, int64, error) {
	blogObjID, err := primitive.ObjectIDFromHex(blogID)
	if err != nil {
		return nil, 0, usecases.ErrInternal
	}
	filter := bson.M{"blog_id": blogObjID, "parent_id": nil}
	return r.fetchPaginated(ctx, filter, page, limit)
}

func (r *commentRepository) FetchReplies(ctx context.Context, parentID string, page, limit int64) ([]*domain.Comment, int64, error) {
	parentObjID, err := primitive.ObjectIDFromHex(parentID)
	if err != nil {
		return nil, 0, usecases.ErrInternal
	}
	filter := bson.M{"parent_id": parentObjID}
	return r.fetchPaginated(ctx, filter, page, limit)
}

// fetchPaginated is a helper to reduce code duplication between FetchByBlogID and FetchReplies.
func (r *commentRepository) fetchPaginated(ctx context.Context, filter bson.M, page, limit int64) ([]*domain.Comment, int64, error) {
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	findOptions := options.Find()
	findOptions.SetLimit(limit)
	findOptions.SetSkip((page - 1) * limit)
	findOptions.SetSort(bson.D{{Key: "created_at", Value: 1}}) // Sort oldest first for conversation flow

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var comments []*domain.Comment
	for cursor.Next(ctx) {
		var model CommentModel
		if err := cursor.Decode(&model); err != nil {
			return nil, 0, err
		}
		comments = append(comments, toCommentDomain(&model))
	}
	return comments, total, cursor.Err()
}

func (r *commentRepository) IncrementReplyCount(ctx context.Context, parentID string, value int) error {
	parentObjID, err := primitive.ObjectIDFromHex(parentID)
	if err != nil {
		return usecases.ErrInternal
	}
	filter := bson.M{"_id": parentObjID}
	update := bson.M{"$inc": bson.M{"reply_count": value}}
	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return usecases.ErrNotFound
	}
	return nil
}
