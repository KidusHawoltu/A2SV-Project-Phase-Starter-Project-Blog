package usecases

import (
	"context"
	"errors"
	"log"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
)

type commentUsecase struct {
	blogRepo    domain.IBlogRepository
	commentRepo domain.ICommentRepository
	timeout     time.Duration
}

func NewCommentUsecase(
	blogRepo domain.IBlogRepository,
	commentRepo domain.ICommentRepository,
	timeout time.Duration,
) domain.ICommentUsecase {
	return &commentUsecase{
		blogRepo:    blogRepo,
		commentRepo: commentRepo,
		timeout:     timeout,
	}
}

func (cu *commentUsecase) CreateComment(ctx context.Context, userID, blogID, content string, parentID *string) (*domain.Comment, error) {
	ctx, cancel := context.WithTimeout(ctx, cu.timeout)
	defer cancel()

	// 1. Usecase-level validation: Check if referenced entities exist.
	if _, err := cu.blogRepo.GetByID(ctx, blogID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound // Or a more specific "blog not found" error
		}
		return nil, err
	}

	// If it's a reply, check if the parent comment exists.
	if parentID != nil && *parentID != "" {
		if _, err := cu.commentRepo.GetByID(ctx, *parentID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound // Or "parent comment not found"
			}
			return nil, err
		}
	}

	// 2. Create the domain entity using the factory. This enforces domain invariants.
	comment, err := domain.NewComment(blogID, userID, content, parentID)
	if err != nil {
		return nil, err // Pass up domain.ErrValidation
	}

	// 3. Persist the new comment.
	if err := cu.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	// 4. After successfully creating the comment, update the counters.
	go func() {
		// Increment the total comment count on the blog post.
		if err := cu.blogRepo.IncrementCommentCount(context.Background(), blogID, 1); err != nil {
			log.Printf("non-critical error: failed to increment comment count for blog %s: %v", blogID, err)
		}
	}()

	if parentID != nil {
		go func() {
			// If it's a reply, also increment the reply count on the parent comment.
			if err := cu.commentRepo.IncrementReplyCount(context.Background(), *parentID, 1); err != nil {
				log.Printf("non-critical error: failed to increment reply count for parent comment %s: %v", *parentID, err)
			}
		}()
	}
	// Note: We don't wait for the WaitGroup here (`wg.Wait()`) because these are non-critical
	// background updates. We want to return the created comment to the user immediately.

	return comment, nil
}

func (cu *commentUsecase) UpdateComment(ctx context.Context, userID, commentID, content string) (*domain.Comment, error) {
	ctx, cancel := context.WithTimeout(ctx, cu.timeout)
	defer cancel()

	// 1. Fetch the existing comment.
	comment, err := cu.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	// 2. Authorization: Enforce business rule that only the author can edit.
	if comment.AuthorID == nil || *comment.AuthorID != userID {
		return nil, domain.ErrPermissionDenied
	}

	// 3. Update the content and timestamp.
	comment.Content = content
	comment.UpdatedAt = time.Now().UTC()

	// 4. Persist the changes.
	if err := cu.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (cu *commentUsecase) DeleteComment(ctx context.Context, userID, commentID string) error {
	ctx, cancel := context.WithTimeout(ctx, cu.timeout)
	defer cancel()

	// 1. Fetch the comment to check for ownership and get its metadata.
	comment, err := cu.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	// 2. Authorization: Only the author can delete.
	isOwner := comment.AuthorID != nil && *comment.AuthorID == userID

	if !isOwner {
		return domain.ErrPermissionDenied
	}

	// 3. Anonymize the comment.
	if err := cu.commentRepo.Anonymize(ctx, commentID); err != nil {
		return err
	}

	// 4. After anonymizing, decrement the relevant counters.
	go func() {
		if err := cu.blogRepo.IncrementCommentCount(context.Background(), comment.BlogID, -1); err != nil {
			log.Printf("non-critical error: failed to decrement comment count for blog %s: %v", comment.BlogID, err)
		}
	}()

	return nil
}

func (cu *commentUsecase) GetCommentsForBlog(ctx context.Context, blogID string, page, limit int64) ([]*domain.Comment, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, cu.timeout)
	defer cancel()

	return cu.commentRepo.FetchByBlogID(ctx, blogID, page, limit)
}

func (cu *commentUsecase) GetRepliesForComment(ctx context.Context, parentID string, page, limit int64) ([]*domain.Comment, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, cu.timeout)
	defer cancel()

	return cu.commentRepo.FetchReplies(ctx, parentID, page, limit)
}
