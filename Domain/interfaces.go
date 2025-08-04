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
	UpdateInteractionCounts(ctx context.Context, blogID string, likesInc, dislikesInc int) error
}

type IInteractionRepository interface {
	Get(ctx context.Context, userID, blogID string) (*BlogInteraction, error)

	Create(ctx context.Context, interaction *BlogInteraction) error

	Update(ctx context.Context, interaction *BlogInteraction) error

	Delete(ctx context.Context, interactionID string) error
}
