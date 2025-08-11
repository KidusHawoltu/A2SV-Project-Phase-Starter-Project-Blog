package infrastructure

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// RateLimiter enforces rate limits using a sliding window log algorithm stored in Redis.
// It can handle different limits for authenticated users and anonymous IPs.
type RateLimiter struct {
	redisClient *redis.Client
}

// NewRateLimiter creates a new RateLimiter instance.
func NewRateLimiter(redisService *RedisService) *RateLimiter {
	if redisService == nil || redisService.Client == nil {
		log.Fatal("FATAL: RedisService is not initialized. Rate limiter cannot be created.")
	}
	return &RateLimiter{
		redisClient: redisService.Client,
	}
}

// LimiterMiddleware returns a Gin middleware handler with the specified rate limit.
func (rl *RateLimiter) LimiterMiddleware(limit int64, period time.Duration, userIDKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := rl.getKey(c, userIDKey)
		now := time.Now().UnixNano()
		key = fmt.Sprintf("rate-limit:%s", key)

		// Use a Redis pipeline for atomic and efficient operations.
		pipe := rl.redisClient.Pipeline()
		// 1. Record the current request timestamp.
		pipe.ZAdd(c, key, &redis.Z{Score: float64(now), Member: float64(now)})
		// 2. Trim timestamps older than the defined period.
		pipe.ZRemRangeByScore(c, key, "0", strconv.FormatInt(now-period.Nanoseconds(), 10))
		// 3. Get the count of requests within the window.
		countCmd := pipe.ZCard(c, key)
		// 4. Get the oldest timestamp in the set to calculate the retry time.
		oldestTimestampCmd := pipe.ZRangeWithScores(c, key, 0, 0)

		_, err := pipe.Exec(c)
		if err != nil {
			log.Printf("ERROR: Rate limiter Redis error: %v. Allowing request.", err)
			c.Next()
			return
		}

		count, err := countCmd.Result()
		if err != nil {
			log.Printf("ERROR: Rate limiter could not get count: %v. Allowing request.", err)
			c.Next()
			return
		}

		if count > limit {
			// Get the result of our command to find the oldest timestamp.
			oldestTimestamps, _ := oldestTimestampCmd.Result()

			var retryAfter time.Duration
			if len(oldestTimestamps) > 0 {
				// The oldest request timestamp in the current window.
				oldestReqNano := int64(oldestTimestamps[0].Score)
				// The time when this oldest request will fall out of the window.
				windowResetTime := time.Unix(0, oldestReqNano).Add(period)
				// The duration from now until that reset time.
				retryAfter = time.Until(windowResetTime)
			} else {
				// Fallback if we can't determine the exact time.
				retryAfter = period
			}

			// Add the 'Retry-After' header (in seconds), which is a standard.
			c.Header("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds())+1, 10))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":      "Too Many Requests",
				"detail":     fmt.Sprintf("You have exceeded the limit of %d requests per %v.", limit, period),
				"retryAfter": fmt.Sprintf("%.0f seconds", retryAfter.Seconds()+1),
			})
			return
		}

		c.Next()
	}
}

// getKey determines the identifier for the current request.
func (rl *RateLimiter) getKey(c *gin.Context, userIDKey string) string {
	if userID, exists := c.Get(userIDKey); exists {
		if idStr, ok := userID.(string); ok && idStr != "" {
			return "user:" + idStr
		}
	}
	return "ip:" + c.ClientIP()
}
