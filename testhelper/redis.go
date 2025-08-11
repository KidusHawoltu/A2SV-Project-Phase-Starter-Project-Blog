package testhelper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Use sync.Once to ensure the Redis container is started only once per test run.
var (
	once          sync.Once
	RedisClient   *redis.Client
	containerAddr string
)

// GetTestRedisClient is a thread-safe function that starts a Redis container on its
// first call and returns a client connected to it. Subsequent calls will return
// the same client without starting a new container.
// It also registers a cleanup function with the testing framework to terminate
// the container when all tests in the package are done.
func GetTestRedisClient(t *testing.T) *redis.Client {
	// The code inside this 'Do' block will only ever execute once across all
	// goroutines and test packages for the entire 'go test' run.
	once.Do(func() {
		log.Println("🛠️ [testhelper] Starting Redis container for integration tests...")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		req := testcontainers.ContainerRequest{
			Image:        "redis:6-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		}

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			// Use t.Fatalf() to stop the test run if the container can't start.
			t.Fatalf("FATAL: Could not start Redis container for tests: %v", err)
		}

		// Register a cleanup function. This will be called when the test
		// session for the package that called this function first concludes.
		t.Cleanup(func() {
			log.Println("🧹 [testhelper] Terminating Redis container...")
			if err := container.Terminate(context.Background()); err != nil {
				t.Fatalf("Failed to terminate Redis container: %v", err)
			}
		})

		host, err := container.Host(ctx)
		if err != nil {
			t.Fatalf("Could not get Redis container host: %v", err)
		}
		port, err := container.MappedPort(ctx, "6379")
		if err != nil {
			t.Fatalf("Could not get Redis container port: %v", err)
		}
		containerAddr = fmt.Sprintf("%s:%s", host, port.Port())

		// Create the client that will be shared.
		RedisClient = redis.NewClient(&redis.Options{Addr: containerAddr})
		if err := RedisClient.Ping(ctx).Err(); err != nil {
			t.Fatalf("Could not ping test Redis container: %v", err)
		}
		log.Println("✅ [testhelper] Redis container is up and running.")
	})

	// Return the memoized, shared client.
	return RedisClient
}
