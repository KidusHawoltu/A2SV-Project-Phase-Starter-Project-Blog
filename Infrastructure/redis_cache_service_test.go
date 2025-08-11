package infrastructure_test

import (
	"context"
	"testing"
	"time"

	domain "A2SV_Starter_Project_Blog/Domain"
	. "A2SV_Starter_Project_Blog/Infrastructure"
	"A2SV_Starter_Project_Blog/testhelper"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/suite"
)

// RedisCacheServiceTestSuite defines the suite for testing the RedisCacheService.
type RedisCacheServiceTestSuite struct {
	suite.Suite
	redisClient  *redis.Client
	cacheService domain.ICacheService
}

func (s *RedisCacheServiceTestSuite) SetupSuite() {
	s.redisClient = testhelper.RedisClient
	s.cacheService = NewRedisCacheService(&RedisService{Client: s.redisClient})
}

// SetupTest runs before each test method, flushing the Redis DB for total isolation.
func (s *RedisCacheServiceTestSuite) SetupTest() {
	// Ensure the database is empty before each test.
	err := s.redisClient.FlushDB(context.Background()).Err()
	s.Require().NoError(err)
}

// TestRedisCacheServiceSuite is the entry point for the suite.
func TestRedisCacheServiceSuite(t *testing.T) {
	suite.Run(t, new(RedisCacheServiceTestSuite))
}

// --- The Actual Test Methods ---

func (s *RedisCacheServiceTestSuite) TestSetAndGet() {
	ctx := context.Background()
	key := "test:set-get"
	value := []byte("hello world")
	ttl := 5 * time.Minute

	// Act: Set a value in the cache.
	err := s.cacheService.Set(ctx, key, value, ttl)
	s.Require().NoError(err, "Set should not return an error")

	// Act: Get the value back from the cache.
	retrievedValue, err := s.cacheService.Get(ctx, key)
	s.Require().NoError(err, "Get should not return an error for an existing key")

	// Assert: The retrieved value should match the original.
	s.Equal(value, retrievedValue, "The retrieved value should be the same as the one set")

	// Assert: Check the TTL of the key in Redis directly to ensure it was set.
	// We check if the TTL is within a reasonable range of the expected value.
	retrievedTTL := s.redisClient.TTL(ctx, key).Val()
	s.InDelta(ttl, retrievedTTL, float64(2*time.Second), "TTL should be close to the set value")
}

func (s *RedisCacheServiceTestSuite) TestGet_NotFound() {
	ctx := context.Background()
	key := "test:non-existent-key"

	// Act: Try to get a key that does not exist.
	retrievedValue, err := s.cacheService.Get(ctx, key)

	// Assert: The error should be our specific domain.ErrNotFound.
	// This tests the error translation logic.
	s.ErrorIs(err, domain.ErrNotFound, "Error should be domain.ErrNotFound for a missing key")
	s.Nil(retrievedValue, "Value should be nil for a missing key")
}

func (s *RedisCacheServiceTestSuite) TestDelete() {
	ctx := context.Background()
	key := "test:to-be-deleted"
	value := []byte("some data")

	// Arrange: Set a value first so we can delete it.
	err := s.cacheService.Set(ctx, key, value, 1*time.Minute)
	s.Require().NoError(err)

	// Verify it's there.
	_, getErr := s.cacheService.Get(ctx, key)
	s.Require().NoError(getErr, "Key should exist before deletion")

	// Act: Delete the key.
	err = s.cacheService.Delete(ctx, key)
	s.Require().NoError(err, "Delete should not return an error")

	// Assert: Getting the key now should result in ErrNotFound.
	_, getErrAfterDelete := s.cacheService.Get(ctx, key)
	s.ErrorIs(getErrAfterDelete, domain.ErrNotFound, "Key should not exist after deletion")
}

func (s *RedisCacheServiceTestSuite) TestSet_WithZeroExpiration() {
	ctx := context.Background()
	key := "test:persistent-key"
	value := []byte("this should persist")

	// Act: Set a value with 0 expiration, which means it should not expire.
	err := s.cacheService.Set(ctx, key, value, 0)
	s.Require().NoError(err)

	// Assert: Check the TTL directly in Redis.
	// A TTL of -1 means the key has no expiration.
	ttl := s.redisClient.TTL(ctx, key).Val()
	s.Equal(time.Duration(-1), ttl, "A zero expiration should result in a persistent key (-1 TTL)")
}

func (s *RedisCacheServiceTestSuite) TestSet_ReplacesExistingValue() {
	ctx := context.Background()
	key := "test:overwrite"
	initialValue := []byte("initial")
	newValue := []byte("new value")

	// Arrange: Set an initial value.
	err := s.cacheService.Set(ctx, key, initialValue, 1*time.Minute)
	s.Require().NoError(err)

	// Act: Set a new value for the same key.
	err = s.cacheService.Set(ctx, key, newValue, 1*time.Minute)
	s.Require().NoError(err)

	// Assert: Get the value and ensure it's the new one.
	retrievedValue, err := s.cacheService.Get(ctx, key)
	s.Require().NoError(err)
	s.Equal(newValue, retrievedValue, "The value should be overwritten with the new value")
}

func (s *RedisCacheServiceTestSuite) TestAddToSetAndGetSetMembers() {
	ctx := context.Background()
	setKey := "test:my-set"
	members := []interface{}{"member1", "member2", "member3"}

	// Act: Add the members to the set.
	err := s.cacheService.AddToSet(ctx, setKey, members...)
	s.Require().NoError(err, "AddToSet should not return an error")

	// Act: Add a duplicate member.
	err = s.cacheService.AddToSet(ctx, setKey, "member2")
	s.Require().NoError(err, "Adding a duplicate member to a set should not error")

	// Act: Retrieve all members from the set.
	retrievedMembers, err := s.cacheService.GetSetMembers(ctx, setKey)
	s.Require().NoError(err, "GetSetMembers should not return an error for an existing set")

	// Assert: Check the members. Sets are unordered, so we use ElementsMatch.
	s.ElementsMatch([]string{"member1", "member2", "member3"}, retrievedMembers, "Retrieved members should match the unique set of added members")
	s.Len(retrievedMembers, 3, "Set should contain exactly 3 unique members")
}

func (s *RedisCacheServiceTestSuite) TestGetSetMembers_NotFound() {
	ctx := context.Background()
	setKey := "test:non-existent-set"

	// Act: Retrieve members from a set that doesn't exist.
	retrievedMembers, err := s.cacheService.GetSetMembers(ctx, setKey)
	s.Require().NoError(err, "GetSetMembers on a non-existent key should not return an error")

	// Assert: The result should be an empty slice.
	s.Empty(retrievedMembers, "Result should be an empty slice for a non-existent set")
}

func (s *RedisCacheServiceTestSuite) TestDeleteKeys() {
	ctx := context.Background()

	// Arrange: Create multiple keys to delete.
	keys := []string{"key:1", "key:2", "set:a"}
	err := s.redisClient.Set(ctx, "key:1", "value1", 0).Err()
	s.Require().NoError(err)
	err = s.redisClient.Set(ctx, "key:2", "value2", 0).Err()
	s.Require().NoError(err)
	err = s.redisClient.SAdd(ctx, "set:a", "member1").Err()
	s.Require().NoError(err)

	// Verify they all exist before deletion.
	count, err := s.redisClient.Exists(ctx, keys...).Result()
	s.Require().NoError(err)
	s.Equal(int64(3), count, "All keys should exist before deletion")

	// Act: Call the method to delete all keys at once.
	err = s.cacheService.DeleteKeys(ctx, keys)
	s.Require().NoError(err, "DeleteKeys should not return an error")

	// Assert: Check that none of the keys exist anymore.
	countAfter, err := s.redisClient.Exists(ctx, keys...).Result()
	s.Require().NoError(err)
	s.Equal(int64(0), countAfter, "None of the keys should exist after deletion")
}
