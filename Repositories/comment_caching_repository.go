package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
)

// Define a struct to hold the cached result for paginated queries.
type paginatedCommentResult struct {
	Comments []*domain.Comment `json:"comments"`
	Total    int64             `json:"total"`
}

// CachingCommentRepository is a decorator for ICommentRepository that adds caching.
type CachingCommentRepository struct {
	next       domain.ICommentRepository
	cache      domain.ICacheService
	defaultTTL time.Duration
}

// NewCachingCommentRepository creates a new caching decorator for the comment repository.
func NewCachingCommentRepository(next domain.ICommentRepository, cache domain.ICacheService) domain.ICommentRepository {
	return &CachingCommentRepository{
		next:       next,
		cache:      cache,
		defaultTTL: 2 * time.Minute, // A shorter TTL is good for dynamic data like comments.
	}
}

// FetchByBlogID caches the first page of top-level comments for a blog.
func (r *CachingCommentRepository) FetchByBlogID(ctx context.Context, blogID string, page, limit int64) ([]*domain.Comment, int64, error) {
	cacheKey := fmt.Sprintf("comments:blog:%s:page:%d:limit:%d", blogID, page, limit)
	trackerKey := fmt.Sprintf("tracker:comments:blog:%s", blogID)

	// Try to get from cache
	cachedData, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var result paginatedCommentResult
		if json.Unmarshal(cachedData, &result) == nil {
			return result.Comments, result.Total, nil
		}
	}

	// Cache MISS, fetch from the primary repository.
	comments, total, err := r.next.FetchByBlogID(ctx, blogID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Store the result AND update the tracker set.
	resultToCache := paginatedCommentResult{Comments: comments, Total: total}
	dataToCache, jsonErr := json.Marshal(resultToCache)
	if jsonErr == nil {
		// Set the actual data
		if err := r.cache.Set(ctx, cacheKey, dataToCache, r.defaultTTL); err != nil {
			log.Printf("[CACHE] Error setting blog comments cache for key %s: %v", cacheKey, err)
		}
		// Add the key to our tracker set
		if err := r.cache.AddToSet(ctx, trackerKey, cacheKey); err != nil {
			log.Printf("[CACHE] Error adding key to tracker set %s: %v", trackerKey, err)
		}
	}

	return comments, total, nil
}

func (r *CachingCommentRepository) FetchReplies(ctx context.Context, parentID string, page, limit int64) ([]*domain.Comment, int64, error) {
	cacheKey := fmt.Sprintf("comments:replies:%s:page:%d:limit:%d", parentID, page, limit)
	trackerKey := fmt.Sprintf("tracker:comments:replies:%s", parentID)

	cachedData, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var result paginatedCommentResult
		if json.Unmarshal(cachedData, &result) == nil {
			return result.Comments, result.Total, nil
		}
	}

	// Cache MISS
	comments, total, err := r.next.FetchReplies(ctx, parentID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Store the result AND update the tracker set.
	resultToCache := paginatedCommentResult{Comments: comments, Total: total}
	dataToCache, jsonErr := json.Marshal(resultToCache)
	if jsonErr == nil {
		r.cache.Set(ctx, cacheKey, dataToCache, r.defaultTTL)
		r.cache.AddToSet(ctx, trackerKey, cacheKey)
	}

	return comments, total, nil
}

// Create must invalidate ALL cached lists for the relevant blog/parent.
func (r *CachingCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	if err := r.next.Create(ctx, comment); err != nil {
		return err
	}

	// Determine which tracker set to use.
	var trackerKey string
	if comment.ParentID != nil {
		trackerKey = fmt.Sprintf("tracker:comments:replies:%s", *comment.ParentID)
	} else {
		trackerKey = fmt.Sprintf("tracker:comments:blog:%s", comment.BlogID)
	}

	// Invalidate using the tracker set.
	return r.invalidateCommentCache(ctx, trackerKey)
}

// --- New Invalidation Helper ---
func (r *CachingCommentRepository) invalidateCommentCache(ctx context.Context, trackerKey string) error {
	// 1. Get all the keys we need to delete from the tracker set.
	keysToDelete, err := r.cache.GetSetMembers(ctx, trackerKey)
	if err != nil {
		log.Printf("[CACHE] Could not get members of tracker set %s: %v", trackerKey, err)
		return nil // Don't fail the operation, just log.
	}

	// If there's nothing to delete, we're done.
	if len(keysToDelete) == 0 {
		return nil
	}

	// 2. Add the tracker key itself to the list of keys to be deleted.
	keysToDelete = append(keysToDelete, trackerKey)

	// 3. Delete all keys in one go.
	if err := r.cache.DeleteKeys(ctx, keysToDelete); err != nil {
		log.Printf("[CACHE] Error invalidating keys for tracker %s: %v", trackerKey, err)
	}

	return nil
}

// --- Pass-Through Methods ---

func (r *CachingCommentRepository) GetByID(ctx context.Context, commentID string) (*domain.Comment, error) {
	return r.next.GetByID(ctx, commentID)
}

func (r *CachingCommentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	// We rely on TTL for this to update in the cache.
	return r.next.Update(ctx, comment)
}

func (r *CachingCommentRepository) Anonymize(ctx context.Context, commentID string) error {
	// We rely on TTL for this to update in the cache.
	return r.next.Anonymize(ctx, commentID)
}

func (r *CachingCommentRepository) IncrementReplyCount(ctx context.Context, parentID string, value int) error {
	// We rely on TTL for this to update in the cache.
	return r.next.IncrementReplyCount(ctx, parentID, value)
}
