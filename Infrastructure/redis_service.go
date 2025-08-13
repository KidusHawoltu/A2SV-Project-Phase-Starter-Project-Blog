package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisService holds the client connection to Redis.
type RedisService struct {
	Client *redis.Client
}

// NewRedisService creates a new RedisService, establishes a connection,
// and pings the server to ensure connectivity.
func NewRedisService(ctx context.Context, url, addr, password string, db int) (*RedisService, error) {
	// Create the connection options.
	opts, err := redis.ParseURL(url)
	if err != nil {
		if addr == "" {
			return nil, fmt.Errorf("redis connection failed: either URL or address must be provided")
		}
		opts = &redis.Options{
			Addr:     addr,     // e.g., "localhost:6379"
			Password: password, // empty string if no password
			DB:       db,       // 0 for the default database
		}
	}

	// Create a new Redis client.
	client := redis.NewClient(opts)

	// Use a short-lived context for the initial ping to fail fast.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Ping the Redis server to verify the connection.
	// This is a crucial health check.
	if err := client.Ping(pingCtx).Err(); err != nil {
		// If the ping fails, we close the client and return an error.
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis and ping server: %w", err)
	}

	// If the connection is successful, return the service.
	return &RedisService{
		Client: client,
	}, nil
}

// Close gracefully closes the Redis connection.
// This should be called on application shutdown.
func (s *RedisService) Close() error {
	if s.Client != nil {
		return s.Client.Close()
	}
	return nil
}
