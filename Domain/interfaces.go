package domain

import (
	"context"
)

type IBlogUsecase interface {
	Create(ctx context.Context, title, content string, authorID string, tags []string) (*Blog, error)

	Fetch(ctx context.Context, page, limit int64) ([]*Blog, int64, error)

	GetByID(ctx context.Context, id string) (*Blog, error)

	Update(ctx context.Context, blogID, userID, userRole string, updates map[string]any) (*Blog, error)

	Delete(ctx context.Context, blogID, userID, userRole string) error
}

type IBlogRepository interface {
	Create(ctx context.Context, blog *Blog) error

	Fetch(ctx context.Context, page, limit int64) ([]*Blog, int64, error)

	GetByID(ctx context.Context, id string) (*Blog, error)

	Update(ctx context.Context, blog *Blog) error

	Delete(ctx context.Context, id string) error
}
