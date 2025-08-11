package infrastructure

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCacheService is the Redis implementation of the ICacheService interface.
type RedisCacheService struct {
	client *redis.Client
}

// NewRedisCacheService creates a new RedisCacheService.
func NewRedisCacheService(redisService *RedisService) domain.ICacheService {
	return &RedisCacheService{
		client: redisService.Client,
	}
}

// Get retrieves an item from the Redis cache.
func (s *RedisCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		// Translate the Redis-specific "not found" error to our domain error.
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return val, nil
}

// Set adds an item to the Redis cache.
func (s *RedisCacheService) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return s.client.Set(ctx, key, value, expiration).Err()
}

// Delete removes an item from the Redis cache.
func (s *RedisCacheService) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

func (s *RedisCacheService) AddToSet(ctx context.Context, key string, members ...any) error {
	// SAdd is variadic, so it can take multiple members at once.
	return s.client.SAdd(ctx, key, members...).Err()
}

// GetSetMembers retrieves all members of a Redis set.
func (s *RedisCacheService) GetSetMembers(ctx context.Context, key string) ([]string, error) {
	return s.client.SMembers(ctx, key).Result()
}

// DeleteKeys uses a pipeline to delete multiple keys atomically.
func (s *RedisCacheService) DeleteKeys(ctx context.Context, keys []string) error {
	// Using a pipeline is much more efficient than sending multiple DEL commands.
	pipe := s.client.Pipeline()
	pipe.Del(ctx, keys...)
	_, err := pipe.Exec(ctx)
	return err
}
