package infrastructure_test

import (
	"context"
	"testing"
	"time"

	. "A2SV_Starter_Project_Blog/Infrastructure"
	"A2SV_Starter_Project_Blog/testhelper"
	"github.com/stretchr/testify/suite"
)

type RedisServiceTestSuite struct {
	suite.Suite
	redisAddr string
}

func (s *RedisServiceTestSuite) SetupSuite() {
	s.redisAddr = testhelper.RedisClient.Options().Addr
}

// TestRedisServiceSuite is the entry point.
func TestRedisServiceSuite(t *testing.T) {
	suite.Run(t, new(RedisServiceTestSuite))
}

// --- The Actual Tests ---

func (s *RedisServiceTestSuite) TestNewRedisService_Success() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Act
	redisService, err := NewRedisService(ctx, s.redisAddr, "", 0)

	// Assert
	s.Require().NoError(err, "NewRedisService should not return an error on successful connection")
	s.Require().NotNil(redisService, "RedisService should not be nil")
	s.Require().NotNil(redisService.Client, "RedisService.Client should not be nil")

	// Verify the connection is actually live
	pingErr := redisService.Client.Ping(ctx).Err()
	s.NoError(pingErr, "Should be able to ping the redis server via the service client")

	// Cleanup
	err = redisService.Close()
	s.NoError(err, "Closing the redis service should not produce an error")
}

func (s *RedisServiceTestSuite) TestNewRedisService_Failure_InvalidAddress() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	redisService, err := NewRedisService(ctx, "localhost:12345", "", 0)

	// Assert
	s.Require().Error(err, "NewRedisService should return an error for an invalid address")
	s.Nil(redisService, "RedisService should be nil on failure")
}

func (s *RedisServiceTestSuite) TestClose() {
	// Arrange
	ctx := context.Background()
	redisService, err := NewRedisService(ctx, s.redisAddr, "", 0)
	s.Require().NoError(err)

	// Act
	err = redisService.Close()
	s.NoError(err)

	// Assert: Pinging with a closed client should result in an error.
	pingErr := redisService.Client.Ping(ctx).Err()
	s.Error(pingErr, "Pinging a closed client should return an error")
}
