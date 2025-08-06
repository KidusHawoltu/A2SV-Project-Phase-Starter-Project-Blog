package domain

import (
	"context"
)

type IBlogUsecase interface {
	Create(ctx context.Context, title, content string, authorID string, tags []string) (*Blog, error)

	SearchAndFilter(ctx context.Context, options BlogSearchFilterOptions) ([]*Blog, int64, error)

	GetByID(ctx context.Context, id string) (*Blog, error)

	Update(ctx context.Context, blogID, userID string, userRole Role, updates map[string]any) (*Blog, error)

	Delete(ctx context.Context, blogID, userID string, userRole Role) error

	InteractWithBlog(ctx context.Context, blogID, userID string, action ActionType) error
}

type IBlogRepository interface {
	Create(ctx context.Context, blog *Blog) error

	SearchAndFilter(ctx context.Context, options BlogSearchFilterOptions) ([]*Blog, int64, error)

	GetByID(ctx context.Context, id string) (*Blog, error)

	Update(ctx context.Context, blog *Blog) error

	Delete(ctx context.Context, id string) error

	IncrementLikes(ctx context.Context, blogID string, value int) error
	IncrementDislikes(ctx context.Context, blogID string, value int) error
	IncrementViews(ctx context.Context, blogID string) error
	IncrementCommentCount(ctx context.Context, blogId string, value int) error
	UpdateInteractionCounts(ctx context.Context, blogID string, likesInc, dislikesInc int) error
}

type IInteractionRepository interface {
	Get(ctx context.Context, userID, blogID string) (*BlogInteraction, error)

	Create(ctx context.Context, interaction *BlogInteraction) error

	Update(ctx context.Context, interaction *BlogInteraction) error

	Delete(ctx context.Context, interactionID string) error
}

type IAIService interface {
	GenerateCompletion(ctx context.Context, prompt string) (string, error)
}

type IAIUsecase interface {
	GenerateBlogIdeas(ctx context.Context, keywords []string) ([]string, error)
	RefineBlogPost(ctx context.Context, content string) (string, error)
}

type ICommentRepository interface {
	Create(ctx context.Context, comment *Comment) error
	GetByID(ctx context.Context, commentID string) (*Comment, error)
	Update(ctx context.Context, comment *Comment) error

	Anonymize(ctx context.Context, commentID string) error

	FetchByBlogID(ctx context.Context, blogID string, page, limit int64) ([]*Comment, int64, error)

	FetchReplies(ctx context.Context, parentID string, page, limit int64) ([]*Comment, int64, error)

	IncrementReplyCount(ctx context.Context, parentID string, value int) error
}

type ICommentUsecase interface {
	CreateComment(ctx context.Context, userID, blogID, content string, parentID *string) (*Comment, error)

	UpdateComment(ctx context.Context, userID, commentID, content string) (*Comment, error)

	DeleteComment(ctx context.Context, userID, commentID string) error

	GetCommentsForBlog(ctx context.Context, blogID string, page, limit int64) ([]*Comment, int64, error)

	GetRepliesForComment(ctx context.Context, parentID string, page, limit int64) ([]*Comment, int64, error)
}
