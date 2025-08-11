package infrastructure_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "A2SV_Starter_Project_Blog/Infrastructure"
	"A2SV_Starter_Project_Blog/testhelper"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

// RateLimiterTestSuite defines the suite for testing the RateLimiter middleware.
type RateLimiterTestSuite struct {
	suite.Suite
	rateLimiter *RateLimiter
}

// SetupSuite starts a Redis container and creates a RateLimiter instance for the suite.
func (s *RateLimiterTestSuite) SetupSuite() {
	s.rateLimiter = NewRateLimiter(&RedisService{Client: testhelper.RedisClient})
}

// SetupTest runs before each test method, flushing the Redis DB for isolation.
func (s *RateLimiterTestSuite) SetupTest() {
	// Flush the entire Redis database to ensure tests are isolated from each other.
	err := testhelper.RedisClient.FlushDB(context.Background()).Err()
	s.Require().NoError(err)
}

// TestRateLimiterSuite is the entry point for the suite.
func TestRateLimiterSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RateLimiterTestSuite))
}

// --- The Actual Test Methods ---

func (s *RateLimiterTestSuite) TestLimiterMiddleware_AllowsRequestsBelowLimit() {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Limit of 5 requests per minute
	limiterMiddleware := s.rateLimiter.LimiterMiddleware(5, time.Minute, "userID")
	router.GET("/test", limiterMiddleware, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Act & Assert
	for i := range 5 {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code, fmt.Sprintf("Request #%d should be allowed", i+1))
	}
}

func (s *RateLimiterTestSuite) TestLimiterMiddleware_BlocksRequestsAboveLimit() {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// A very tight limit for testing: 2 requests per minute
	limiterMiddleware := s.rateLimiter.LimiterMiddleware(2, time.Minute, "userID")
	router.GET("/test", limiterMiddleware, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Act: Send two successful requests
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		s.Equal(http.StatusOK, w.Code)
	}

	// Act & Assert: The third request should be blocked
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	s.Equal(http.StatusTooManyRequests, w.Code, "The third request should be rate limited")
	s.Contains(w.Body.String(), "Too Many Requests")
	// Check for the presence of our user-friendly retry message
	s.Contains(w.Body.String(), "retryAfter")
	// Check for the standard HTTP header
	s.NotEmpty(w.Header().Get("Retry-After"))
}

func (s *RateLimiterTestSuite) TestLimiterMiddleware_DifferentiatesByUserID() {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Limit of 1 request per user
	limiterMiddleware := s.rateLimiter.LimiterMiddleware(1, time.Minute, "userID")
	router.GET("/test-user", limiterMiddleware, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// --- User 1 ---
	// Act: First request for user1 should succeed
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/test-user", nil)
	// Manually set the user ID on the context, simulating what AuthMiddleware would do
	gin.SetMode(gin.TestMode)
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = req1
	c1.Set("userID", "user-123")
	limiterMiddleware(c1)
	s.Equal(http.StatusOK, w1.Code, "First request for user-123 should be allowed")

	// Act: Second request for user1 should be blocked
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/test-user", nil)
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = req2
	c2.Set("userID", "user-123")
	limiterMiddleware(c2)
	s.Equal(http.StatusTooManyRequests, w2.Code, "Second request for user-123 should be blocked")

	// --- User 2 ---
	// Act: First request for user456 should succeed, because it's a different user
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/test-user", nil)
	c3, _ := gin.CreateTestContext(w3)
	c3.Request = req3
	c3.Set("userID", "user-456")
	limiterMiddleware(c3)
	s.Equal(http.StatusOK, w3.Code, "First request for user-456 should be allowed")
}

func (s *RateLimiterTestSuite) TestLimiterMiddleware_ResetsAfterPeriod() {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// A very short period for testing: 1 request per 2 seconds
	limiterMiddleware := s.rateLimiter.LimiterMiddleware(1, 2*time.Second, "userID")
	router.GET("/test", limiterMiddleware, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Act: First request should succeed
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w1, req1)
	s.Equal(http.StatusOK, w1.Code)

	// Act: Second request immediately after should be blocked
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w2, req2)
	s.Equal(http.StatusTooManyRequests, w2.Code)

	// Act: Wait for the period to expire
	time.Sleep(2 * time.Second)

	// Act & Assert: A new request should now be allowed
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w3, req3)
	s.Equal(http.StatusOK, w3.Code, "Request should be allowed after the time period has reset")
}
