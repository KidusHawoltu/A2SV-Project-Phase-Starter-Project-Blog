package infrastructure_test

import (
	"context"
	"testing"

	. "A2SV_Starter_Project_Blog/Infrastructure"
	"A2SV_Starter_Project_Blog/testhelper"
	"github.com/stretchr/testify/suite"
)

// RedisServiceTestSuite defines the test suite.
type RedisServiceTestSuite struct {
	suite.Suite
	redisAddr string // e.g., "localhost:6379"
	redisURI  string // e.g., "redis://localhost:6379"
}

// SetupSuite runs once before the suite starts.
func (s *RedisServiceTestSuite) SetupSuite() {
	s.redisAddr = testhelper.RedisClient.Options().Addr
	s.redisURI = "redis://" + s.redisAddr
}

// TestRedisServiceSuite is the entry point.
func TestRedisServiceSuite(t *testing.T) {
	suite.Run(t, new(RedisServiceTestSuite))
}

// --- The Actual Tests ---

func (s *RedisServiceTestSuite) TestNewRedisService() {
	ctx := context.Background()

	s.Run("Success - Connecting with Addr, Pass, DB", func() {
		// This test verifies the fallback logic when the URL is empty.
		// Act
		redisService, err := NewRedisService(ctx, "", s.redisAddr, "", 0)

		// Assert
		s.Require().NoError(err, "Should connect successfully using address parts")
		s.Require().NotNil(redisService)
		s.NoError(redisService.Client.Ping(ctx).Err(), "Client should be live")

		// Cleanup
		s.NoError(redisService.Close())
	})

	s.Run("Success - Connecting with full URL", func() {
		// This test verifies the primary logic of parsing a full URI.
		// Act
		redisService, err := NewRedisService(ctx, s.redisURI, "", "", 0)

		// Assert
		s.Require().NoError(err, "Should connect successfully using a full URI")
		s.Require().NotNil(redisService)
		s.NoError(redisService.Client.Ping(ctx).Err(), "Client should be live")

		// Cleanup
		s.NoError(redisService.Close())
	})

	s.Run("Failure - Invalid Address", func() {
		// Act
		redisService, err := NewRedisService(ctx, "", "localhost:12345", "", 0)

		// Assert
		s.Require().Error(err, "Should return an error for an invalid address")
		s.Nil(redisService)
	})

	s.Run("Failure - Invalid URL", func() {
		// Act
		redisService, err := NewRedisService(ctx, "redis://not a valid url!", "", "", 0)

		// Assert
		s.Require().Error(err, "Should return an error for a malformed URL")
		s.Nil(redisService)
	})
}

func (s *RedisServiceTestSuite) TestClose() {
	// Arrange
	ctx := context.Background()
	// Use the fallback connection method for this test
	redisService, err := NewRedisService(ctx, "", s.redisAddr, "", 0)
	s.Require().NoError(err)

	// Act
	err = redisService.Close()
	s.NoError(err)

	// Assert: Pinging with a closed client should result in an error.
	pingErr := redisService.Client.Ping(ctx).Err()
	s.Error(pingErr, "Pinging a closed client should return an error")
}
