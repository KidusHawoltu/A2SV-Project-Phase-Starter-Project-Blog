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

// CachingTokenRepository is a decorator for TokenRepository that adds a caching layer.
type CachingTokenRepository struct {
	next  usecases.TokenRepository
	cache domain.ICacheService
}

// NewCachingTokenRepository creates a new caching decorator for the token repository.
func NewCachingTokenRepository(next usecases.TokenRepository, cache domain.ICacheService) usecases.TokenRepository {
	return &CachingTokenRepository{
		next:  next,
		cache: cache,
	}
}

// --- Caching and Invalidation Methods ---

// Store implements a "write-through" cache. It stores in the DB, then immediately caches the result.
func (r *CachingTokenRepository) Store(ctx context.Context, token *domain.Token) error {
	// 1. Store the token in the primary data source (MongoDB).
	if err := r.next.Store(ctx, token); err != nil {
		return err
	}

	// 2. If successful, "warm up" the cache immediately.
	cacheKey := fmt.Sprintf("token:value:%s", token.Value)

	// Calculate the remaining lifetime of the token to use as the cache TTL.
	// This ensures the cache and DB expiry are synchronized.
	ttl := time.Until(token.ExpiresAt)
	if ttl <= 0 {
		return nil // Don't cache an already-expired token.
	}

	tokenBytes, jsonErr := json.Marshal(token)
	if jsonErr != nil {
		log.Printf("[CACHE] Error marshaling token for cache: %v", jsonErr)
		return nil // The primary store succeeded, so we don't return an error.
	}

	if cacheErr := r.cache.Set(ctx, cacheKey, tokenBytes, ttl); cacheErr != nil {
		log.Printf("[CACHE] Error setting token cache for key %s: %v", cacheKey, cacheErr)
	}

	return nil
}

// GetByValue is the primary cached read method.
func (r *CachingTokenRepository) GetByValue(ctx context.Context, tokenValue string) (*domain.Token, error) {
	cacheKey := fmt.Sprintf("token:value:%s", tokenValue)

	// 1. Try to fetch from the cache first.
	cachedToken, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var token domain.Token
		if json.Unmarshal(cachedToken, &token) == nil {
			return &token, nil // Cache HIT!
		}
	}
	if !errors.Is(err, domain.ErrNotFound) {
		log.Printf("[CACHE] Error getting token from cache: %v", err)
	}

	// 2. Cache MISS. Fetch from the primary repository.
	token, err := r.next.GetByValue(ctx, tokenValue)
	if err != nil {
		return nil, err
	}

	// 3. Store the result in the cache for the next request.
	ttl := time.Until(token.ExpiresAt)
	if ttl <= 0 {
		return token, nil // Return the found token, but don't cache if it's already expired.
	}
	tokenBytes, _ := json.Marshal(token)
	r.cache.Set(ctx, cacheKey, tokenBytes, ttl) // Errors are logged inside the service.

	return token, nil
}

// Delete must invalidate the cache.
func (r *CachingTokenRepository) Delete(ctx context.Context, tokenID string) error {
	// To invalidate the cache, we need the token's value, which is not in the input.
	// We must fetch the token first before deleting it.
	// This is a trade-off for data consistency.
	tokenToDelete, err := r.next.GetByID(ctx, tokenID) // Assuming GetByID exists on the repo interface
	if err != nil {
		if errors.Is(err, usecases.ErrNotFound) {
			return nil // If it's not in the DB, it can't be in the cache. No-op.
		}
		return err
	}

	// Now, perform the actual deletion.
	if err := r.next.Delete(ctx, tokenID); err != nil {
		return err
	}

	// If DB deletion was successful, invalidate the cache.
	cacheKey := fmt.Sprintf("token:value:%s", tokenToDelete.Value)
	if err := r.cache.Delete(ctx, cacheKey); err != nil && !errors.Is(err, domain.ErrNotFound) {
		log.Printf("[CACHE] Error deleting token cache for key %s: %v", cacheKey, err)
	}

	return nil
}

// --- Pass-Through Methods ---

func (r *CachingTokenRepository) DeleteByUserID(ctx context.Context, userID string, tokenType domain.TokenType) error {
	return r.next.DeleteByUserID(ctx, userID, tokenType)
}

func (r *CachingTokenRepository) GetByID(ctx context.Context, tokenID string) (*domain.Token, error) {
	return r.next.GetByID(ctx, tokenID)
}
