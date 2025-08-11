package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
)

type CachingBlogRepository struct {
	next       domain.IBlogRepository
	cache      domain.ICacheService
	defaultTTL time.Duration
}

// NewCachingBlogRepository creates a new caching decorator for the blog repository.
func NewCachingBlogRepository(next domain.IBlogRepository, cache domain.ICacheService) domain.IBlogRepository {
	return &CachingBlogRepository{
		next:       next,
		cache:      cache,
		defaultTTL: 5 * time.Minute, // Cache a blog post for 5 minutes
	}
}

// GetByID is the primary method we will cache.
func (r *CachingBlogRepository) GetByID(ctx context.Context, id string) (*domain.Blog, error) {
	// 1. Define the cache key.
	cacheKey := fmt.Sprintf("blog:id:%s", id)

	// 2. Try to fetch from the cache.
	cachedBlog, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		// Cache HIT!
		var blog domain.Blog
		if json.Unmarshal(cachedBlog, &blog) == nil {
			return &blog, nil
		}
	}
	if !errors.Is(err, domain.ErrNotFound) {
		log.Printf("[CACHE] Error getting blog from cache: %v", err)
	}

	// 3. Cache MISS. Fetch from the primary repository (MongoDB).
	blog, err := r.next.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 4. Store the result in the cache for the next request.
	blogBytes, jsonErr := json.Marshal(blog)
	if jsonErr != nil {
		log.Printf("[CACHE] Error marshaling blog for cache: %v", jsonErr)
		return blog, nil // Don't fail the request, just the caching step.
	}

	if cacheErr := r.cache.Set(ctx, cacheKey, blogBytes, r.defaultTTL); cacheErr != nil {
		log.Printf("[CACHE] Error setting blog cache for key %s: %v", cacheKey, cacheErr)
	}

	return blog, nil
}

// Update must invalidate the cache to prevent serving stale content.
func (r *CachingBlogRepository) Update(ctx context.Context, blog *domain.Blog) error {
	// 1. Update the primary data source first.
	if err := r.next.Update(ctx, blog); err != nil {
		return err
	}

	// 2. If successful, invalidate the cache.
	cacheKey := fmt.Sprintf("blog:id:%s", blog.ID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("[CACHE] Error deleting blog cache for key %s: %v", cacheKey, err)
	}
	return nil
}

// Delete must also invalidate the cache.
func (r *CachingBlogRepository) Delete(ctx context.Context, id string) error {
	// 1. Delete from the primary data source first.
	if err := r.next.Delete(ctx, id); err != nil {
		return err
	}

	// 2. If successful, invalidate the cache.
	cacheKey := fmt.Sprintf("blog:id:%s", id)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("[CACHE] Error deleting blog cache for key %s: %v", cacheKey, err)
	}
	return nil
}

// --- Pass-Through Methods ---
// For all other methods, we simply pass the call directly to the wrapped repository.

func (r *CachingBlogRepository) Create(ctx context.Context, blog *domain.Blog) error {
	return r.next.Create(ctx, blog)
}

func (r *CachingBlogRepository) SearchAndFilter(ctx context.Context, opts domain.BlogSearchFilterOptions) ([]*domain.Blog, int64, error) {
	return r.next.SearchAndFilter(ctx, opts)
}

func (r *CachingBlogRepository) IncrementLikes(ctx context.Context, blogID string, value int) error {
	// We rely on TTL for this to update in the cache.
	return r.next.IncrementLikes(ctx, blogID, value)
}

func (r *CachingBlogRepository) IncrementDislikes(ctx context.Context, blogID string, value int) error {
	// We rely on TTL for this to update in the cache.
	return r.next.IncrementDislikes(ctx, blogID, value)
}

func (r *CachingBlogRepository) IncrementViews(ctx context.Context, blogID string) error {
	// We rely on TTL for this to update in the cache.
	return r.next.IncrementViews(ctx, blogID)
}

func (r *CachingBlogRepository) IncrementCommentCount(ctx context.Context, blogID string, value int) error {
	// We rely on TTL for this to update in the cache.
	return r.next.IncrementCommentCount(ctx, blogID, value)
}

func (r *CachingBlogRepository) UpdateInteractionCounts(ctx context.Context, blogID string, likesInc, dislikesInc int) error {
	// We rely on TTL for this to update in the cache.
	return r.next.UpdateInteractionCounts(ctx, blogID, likesInc, dislikesInc)
}
