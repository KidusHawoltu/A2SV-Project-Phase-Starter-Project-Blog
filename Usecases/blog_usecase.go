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
	blogRepo        domain.IBlogRepository
	userRepo        UserRepository
	interactionRepo domain.IInteractionRepository
	contextTimeout  time.Duration
}

// NewBlogUsecase is the constructor for a blogUsecase.
// It uses dependency injection to receive its dependencies.
func NewBlogUsecase(blogRepository domain.IBlogRepository, userRepository UserRepository, interactionRepository domain.IInteractionRepository, timeout time.Duration) domain.IBlogUsecase {
	return &blogUsecase{
		blogRepo:        blogRepository,
		userRepo:        userRepository,
		interactionRepo: interactionRepository,
		contextTimeout:  timeout,
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

func (bu *blogUsecase) SearchAndFilter(ctx context.Context, options domain.BlogSearchFilterOptions) ([]*domain.Blog, int64, error) {
	// 1. Set up a context with a timeout for the entire operation.
	ctx, cancel := context.WithTimeout(ctx, bu.contextTimeout)
	defer cancel()

	if options.AuthorName != nil && *options.AuthorName != "" {
		// Find all user IDs that match the provided name.
		userIDs, err := bu.userRepo.FindUserIDsByName(ctx, *options.AuthorName)
		if err != nil {
			// If there's a problem querying users, it's an internal error.
			return nil, 0, ErrInternal
		}

		// If no users are found with that name and the logic is AND, no blogs can possibly match
		//  We can short-circuit here and return an empty result
		if len(userIDs) == 0 && options.GlobalLogic == domain.GlobalLogicAND {
			return []*domain.Blog{}, 0, nil
		}

		// Add the found IDs to the options struct. The repository will use this.
		options.AuthorIDs = userIDs
	}

	if options.Limit <= 0 {
		options.Limit = 10
	}
	if options.Limit > 100 {
		options.Limit = 100 // Enforce a max limit
	}
	if options.Page <= 0 {
		options.Page = 1
	}

	return bu.blogRepo.SearchAndFilter(ctx, options)
}

// GetByID retrieves a single blog post.
func (bu *blogUsecase) GetByID(ctx context.Context, id string) (*domain.Blog, error) {
	ctx, cancel := context.WithTimeout(ctx, bu.contextTimeout)
	defer cancel()

	blog, err := bu.blogRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Increment the view of the blog by 1 in background
	go func() {
		_ = bu.blogRepo.IncrementViews(context.Background(), id)
	}()

	return blog, nil
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

func (bu *blogUsecase) InteractWithBlog(ctx context.Context, blogID, userID string, newAction domain.ActionType) error {
	ctx, cancel := context.WithTimeout(ctx, bu.contextTimeout)
	defer cancel()

	// Step 1: Check if an interaction already exists for this user and blog.
	interaction, err := bu.interactionRepo.Get(ctx, userID, blogID)
	// We specifically check for ErrNotFound. Any other error is a real problem.
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err // Return on unexpected database errors
	}

	// --- Scenario 1: No previous interaction exists. ---
	if interaction == nil {
		newInteraction := &domain.BlogInteraction{
			UserID: userID,
			BlogID: blogID,
			Action: newAction,
		}
		if err := bu.interactionRepo.Create(ctx, newInteraction); err != nil {
			return err
		}

		if newAction == domain.ActionTypeLike {
			return bu.blogRepo.IncrementLikes(ctx, blogID, 1)
		}
		return bu.blogRepo.IncrementDislikes(ctx, blogID, 1)
	}

	// --- Scenario 2: The user is repeating the same action (e.g., clicking "like" on an already-liked post). ---
	// This is an "undo" operation.
	if interaction.Action == newAction {
		// Delete the interaction record to remove their "vote".
		if err := bu.interactionRepo.Delete(ctx, interaction.ID); err != nil {
			return err
		}

		// Atomically decrement the correct counter.
		if newAction == domain.ActionTypeLike {
			return bu.blogRepo.IncrementLikes(ctx, blogID, -1)
		}
		return bu.blogRepo.IncrementDislikes(ctx, blogID, -1)
	}

	// --- Scenario 3: The user is switching their action (e.g., from dislike to like). ---
	// First, update the action in the interaction record.
	interaction.Action = newAction
	if err := bu.interactionRepo.Update(ctx, interaction); err != nil {
		return err
	}

	// Prepare the increments for the single atomic update.
	var likesIncrement, dislikesIncrement int
	if newAction == domain.ActionTypeLike {
		// Switching TO a like: likes go up, dislikes go down.
		likesIncrement = 1
		dislikesIncrement = -1
	} else { // Switching TO a dislike
		likesIncrement = -1
		dislikesIncrement = 1
	}

	// Call the single, atomic repository method to update both counts.
	// This prevents data inconsistency if one of the two updates were to fail.
	return bu.blogRepo.UpdateInteractionCounts(ctx, blogID, likesIncrement, dislikesIncrement)
}
