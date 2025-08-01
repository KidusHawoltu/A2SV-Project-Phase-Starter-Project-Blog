package usecases

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("resource conflict or already exists")
	ErrInternal = errors.New("internal server error")
)

// blogUsecase implements the domain.BlogUsecase interface.
// It orchestrates the business logic, using the repository for persistence.
type blogUsecase struct {
	blogRepo       domain.IBlogRepository
	userRepo       UserRepository
	contextTimeout time.Duration
}

// NewBlogUsecase is the constructor for a blogUsecase.
// It uses dependency injection to receive its dependencies.
func NewBlogUsecase(blogRepository domain.IBlogRepository, userRepository UserRepository, timeout time.Duration) domain.IBlogUsecase {
	return &blogUsecase{
		blogRepo:       blogRepository,
		userRepo:       userRepository,
		contextTimeout: timeout,
	}
}

// Create handles the business logic for creating a new blog post.
func (bu *blogUsecase) Create(ctx context.Context, title, content, authorID string, tags []string) (*domain.Blog, error) {
	// 1. Attempt to create the domain entity using the validating factory.
	// This enforces the domain's own invariants first.
	newBlog, err := domain.NewBlog(title, content, authorID, tags)
	if err != nil {
		// The error will be domain.ErrValidation, which we pass up.
		return nil, err
	}

	// 2. The usecase could perform additional, application-specific validation here.
	// (e.g., check if authorID exists in a user repository).
	author, err := bu.userRepo.GetByID(ctx, authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, domain.ErrUserNotFound
	}

	// 3. Set up a context with a timeout for the repository call.
	ctx, cancel := context.WithTimeout(ctx, bu.contextTimeout)
	defer cancel()

	// 4. Call the repository to persist the new blog.
	// The repository is responsible for generating and setting the final ID on the object.
	err = bu.blogRepo.Create(ctx, newBlog)
	if err != nil {
		// The repository might return ErrConflict or ErrInternal.
		return nil, err
	}

	return newBlog, nil
}

// Fetch retrieves a paginated list of blog posts.
func (bu *blogUsecase) Fetch(ctx context.Context, page, limit int64) ([]*domain.Blog, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, bu.contextTimeout)
	defer cancel()

	// Enforce application-wide pagination limits.
	if limit <= 0 || limit > 100 {
		limit = 10 // Default limit
	}
	if page <= 0 {
		page = 1
	}

	return bu.blogRepo.Fetch(ctx, page, limit)
}

// GetByID retrieves a single blog post.
func (bu *blogUsecase) GetByID(ctx context.Context, id string) (*domain.Blog, error) {
	ctx, cancel := context.WithTimeout(ctx, bu.contextTimeout)
	defer cancel()

	// The repository will handle translating a DB "not found" to our usecases.ErrNotFound.
	return bu.blogRepo.GetByID(ctx, id)
}

// Update handles the logic for updating a post, including authorization.
func (bu *blogUsecase) Update(ctx context.Context, blogID, userID string, userRole domain.Role, updates map[string]interface{}) (*domain.Blog, error) {
	ctx, cancel := context.WithTimeout(ctx, bu.contextTimeout)
	defer cancel()

	// 1. Fetch the existing blog post.
	blogToUpdate, err := bu.blogRepo.GetByID(ctx, blogID)
	if err != nil {
		return nil, err // Could be usecases.ErrNotFound
	}

	// 2. Authorization Check: Per PRD 3.2.3, only the author can update their post.
	if blogToUpdate.AuthorID != userID {
		return nil, domain.ErrPermissionDenied
	}

	// 3. Apply updates from the map. This is a secure way to handle partial updates.
	if title, ok := updates["title"].(string); ok {
		// Also enforce invariants on update. A title cannot be updated to be empty.
		if strings.TrimSpace(title) == "" {
			return nil, domain.ErrValidation
		}
		blogToUpdate.Title = title
	}
	if content, ok := updates["content"].(string); ok {
		if strings.TrimSpace(content) == "" {
			return nil, domain.ErrValidation
		}
		blogToUpdate.Content = content
	}
	if tags, ok := updates["tags"].([]string); ok {
		blogToUpdate.Tags = tags
	}

	// 4. Update the timestamp and persist the changes.
	blogToUpdate.UpdatedAt = time.Now().UTC()
	err = bu.blogRepo.Update(ctx, blogToUpdate)
	if err != nil {
		return nil, err
	}

	return blogToUpdate, nil
}

// Delete handles the logic for deleting a post, including complex authorization.
func (bu *blogUsecase) Delete(ctx context.Context, blogID, userID string, userRole domain.Role) error {
	ctx, cancel := context.WithTimeout(ctx, bu.contextTimeout)
	defer cancel()

	// 1. Fetch the blog to check for ownership.
	blogToDelete, err := bu.blogRepo.GetByID(ctx, blogID)
	if err != nil {
		return err // Could be usecases.ErrNotFound
	}

	// 2. Authorization Logic: An Admin can delete any post, a User can only delete their own.
	isOwner := blogToDelete.AuthorID == userID
	isAdmin := userRole == domain.RoleAdmin

	if !isAdmin && !isOwner {
		return domain.ErrPermissionDenied
	}

	// 3. If authorization passes, call the repository to delete the post.
	return bu.blogRepo.Delete(ctx, blogID)
}
