package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	usecases "A2SV_Starter_Project_Blog/Usecases"
)

// CachingUserRepository is a decorator for a UserRepository that adds a caching layer.
type CachingUserRepository struct {
	next       usecases.UserRepository // Using the usecase interface for consistency
	cache      domain.ICacheService
	defaultTTL time.Duration
}

// NewCachingUserRepository creates a new caching decorator for the user repository.
func NewCachingUserRepository(next usecases.UserRepository, cache domain.ICacheService) usecases.UserRepository {
	return &CachingUserRepository{
		next:       next,
		cache:      cache,
		defaultTTL: 1 * time.Hour, // User profiles don't change often, can have a longer TTL
	}
}

// GetByID is the primary method we will cache.
func (r *CachingUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	// 1. Define the cache key.
	cacheKey := fmt.Sprintf("user:id:%s", id)

	// 2. Try to fetch from the cache.
	cachedUser, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var user domain.User
		if json.Unmarshal(cachedUser, &user) == nil {
			return &user, nil
		}
	}
	if !errors.Is(err, domain.ErrNotFound) {
		log.Printf("[CACHE] Error getting user from cache: %v", err)
	}
	user, err := r.next.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 4. Store the result in the cache for the next request.
	userBytes, jsonErr := json.Marshal(user)
	if jsonErr != nil {
		return user, nil // Don't fail the request, just the caching step.
	}

	if cacheErr := r.cache.Set(ctx, cacheKey, userBytes, r.defaultTTL); cacheErr != nil {
		log.Printf("[CACHE] Error setting user cache for key %s: %v", cacheKey, cacheErr)
	}

	return user, nil
}

// Update must invalidate the cache to prevent stale data.
func (r *CachingUserRepository) Update(ctx context.Context, user *domain.User) error {
	// 1. Update the primary data source first.
	if err := r.next.Update(ctx, user); err != nil {
		return err
	}

	// 2. If successful, invalidate the cache for this user.
	cacheKey := fmt.Sprintf("user:id:%s", user.ID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("[CACHE] Error deleting user cache for key %s: %v", cacheKey, err)
	}

	return nil
}

// --- Pass-Through Methods ---
// For all other methods, we simply pass the call directly to the wrapped repository.
// The caching layer is not involved in these operations.

func (r *CachingUserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.next.Create(ctx, user)
}

func (r *CachingUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.next.GetByEmail(ctx, email)
}

func (r *CachingUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.next.GetByUsername(ctx, username)
}

func (r *CachingUserRepository) FindUserIDsByName(ctx context.Context, authorName string) ([]string, error) {
	return r.next.FindUserIDsByName(ctx, authorName)
}

func (r *CachingUserRepository) FindByProviderID(ctx context.Context, provider domain.AuthProvider, providerID string) (*domain.User, error) {
	return r.next.FindByProviderID(ctx, provider, providerID)
}

func (r *CachingUserRepository) SearchAndFilter(ctx context.Context, opts domain.UserSearchFilterOptions) ([]*domain.User, int64, error) {
	return r.next.SearchAndFilter(ctx, opts)
}
