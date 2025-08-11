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

// CachingInteractionRepository is a decorator for IInteractionRepository that adds a caching layer.
type CachingInteractionRepository struct {
	next       domain.IInteractionRepository
	cache      domain.ICacheService
	defaultTTL time.Duration
}

// NewCachingInteractionRepository creates a new caching decorator for the interaction repository.
func NewCachingInteractionRepository(next domain.IInteractionRepository, cache domain.ICacheService) domain.IInteractionRepository {
	return &CachingInteractionRepository{
		next:       next,
		cache:      cache,
		defaultTTL: 15 * time.Minute, // Interaction states are read often, a moderate TTL is good.
	}
}

// Get is the primary cached read method.
func (r *CachingInteractionRepository) Get(ctx context.Context, userID, blogID string) (*domain.BlogInteraction, error) {
	// 1. Define the cache key.
	cacheKey := fmt.Sprintf("interaction:user:%s:blog:%s", userID, blogID)

	// 2. Try to fetch from the cache.
	cachedInteraction, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		// Cache HIT!
		var interaction domain.BlogInteraction
		if json.Unmarshal(cachedInteraction, &interaction) == nil {
			return &interaction, nil
		}
	}
	if !errors.Is(err, domain.ErrNotFound) {
		log.Printf("[CACHE] Error getting interaction from cache: %v", err)
	}

	// 3. Cache MISS. Fetch from the primary repository (MongoDB).
	interaction, err := r.next.Get(ctx, userID, blogID)
	if err != nil {
		return nil, err
	}

	// 4. Store the result in the cache for the next request.
	interactionBytes, jsonErr := json.Marshal(interaction)
	if jsonErr != nil {
		log.Printf("[CACHE] Error marshaling interaction for cache: %v", jsonErr)
		return interaction, nil // Don't fail the request, just the caching step.
	}

	if cacheErr := r.cache.Set(ctx, cacheKey, interactionBytes, r.defaultTTL); cacheErr != nil {
		log.Printf("[CACHE] Error setting interaction cache for key %s: %v", cacheKey, cacheErr)
	}

	return interaction, nil
}

// Update must invalidate the cache to prevent serving a stale state.
func (r *CachingInteractionRepository) Update(ctx context.Context, interaction *domain.BlogInteraction) error {
	// 1. Update the primary data source first.
	if err := r.next.Update(ctx, interaction); err != nil {
		return err
	}

	// 2. If successful, invalidate the cache for this user/blog pair.
	cacheKey := fmt.Sprintf("interaction:user:%s:blog:%s", interaction.UserID, interaction.BlogID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("[CACHE] Error deleting interaction cache for key %s: %v", cacheKey, err)
	}
	return nil
}

// Delete must invalidate the cache.
func (r *CachingInteractionRepository) Delete(ctx context.Context, interactionID string) error {
	// To invalidate the cache, we need the userID and blogID from the interaction.
	// We must fetch the interaction from the DB first.
	interactionToDelete, err := r.next.GetByID(ctx, interactionID) // Requires GetByID
	if err != nil {
		if errors.Is(err, usecases.ErrNotFound) {
			return nil // Already gone from DB, nothing to do.
		}
		return err
	}

	// Now, perform the actual deletion from the database.
	if err := r.next.Delete(ctx, interactionID); err != nil {
		return err
	}

	// If DB deletion was successful, invalidate the cache using the IDs we fetched.
	cacheKey := fmt.Sprintf("interaction:user:%s:blog:%s", interactionToDelete.UserID, interactionToDelete.BlogID)
	if err := r.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("[CACHE] Error deleting interaction cache for key %s: %v", cacheKey, err)
	}

	return nil
}

// --- Pass-Through Methods ---

func (r *CachingInteractionRepository) Create(ctx context.Context, interaction *domain.BlogInteraction) error {
	return r.next.Create(ctx, interaction)
}

func (r *CachingInteractionRepository) GetByID(ctx context.Context, id string) (*domain.BlogInteraction, error) {
	return r.next.GetByID(ctx, id)
}
