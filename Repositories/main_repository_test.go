package repositories_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// testDB will hold the connection to the database and will be available to all tests in this package.
var testDB *mongo.Database
var ttlWait = 60

// TestMain is a special function that Go's testing framework runs before any tests in the package.
// It's the perfect place for global setup and teardown.
func TestMain(m *testing.M) {
	// --- Setup ---

	// Load .env.test first for test-specific configurations.
	// Fallback to regular .env for shared configs like MONGO_URI.
	godotenv.Load(".env.test")
	godotenv.Load("../.env.test")
	godotenv.Load(".env")
	godotenv.Load("../.env")

	// Use our custom helper to get configuration with test-specific fallbacks.
	mongoURI := getTestEnv("MONGO_URI_TEST", "MONGO_URI", "mongodb://localhost:27017")

	// IMPORTANT SAFETY NOTICE:
	// Always use a separate database for testing to avoid wiping production or development data.
	dbName := getTestEnv("DB_NAME_TEST", "DB_NAME", "g6-blog-db-test")
	if dbName == "g6-blog-db" {
		log.Fatalf("FATAL: Cannot run tests on the main database '%s'. Set DB_NAME_TEST in your .env.test file.", dbName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB for testing: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB (ping failed): %v", err)
	}

	// Assign the database handle to our package-level variable.
	testDB = client.Database(dbName)
	log.Printf("Integration tests will use database: %s", dbName)

	// --- Speed up the TTL Monitor ---
	if err := setupTTLMonitor(client, 1); err != nil {
		log.Println("Failed to speed up TTL monitor")
		ttlWait = 65
	}

	// Defer the cleanup to reset the TTL monitor back to its default (60s)
	defer func() {
		log.Println("Resetting TTL monitor to default (60s)...")
		if err := setupTTLMonitor(client, 60); err != nil {
			log.Println("WARNING: Could not reset TTL monitor")
		}

		// --- Teardown ---
		log.Printf("Tearing down: dropping database %s", dbName)
		if err := testDB.Drop(context.Background()); err != nil {
			log.Fatalf("Failed to drop test database: %v", err)
		}

		if err := client.Disconnect(context.Background()); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	// --- Run Tests ---
	// m.Run() executes all the tests in the package. The exit code is captured.
	exitCode := m.Run()

	// Exit with the code from the test run.
	os.Exit(exitCode)
}

// getTestEnv implements the desired fallback logic for environment variables.
// It checks for a test-specific key, then a normal key, and finally returns a default value.
func getTestEnv(testKey, fallbackKey, defaultValue string) string {
	// 1. Check for the test-specific environment variable (e.g., DB_NAME_TEST)
	if value, exists := os.LookupEnv(testKey); exists {
		return value
	}
	// 2. If not found, check for the normal environment variable (e.g., DB_NAME)
	if value, exists := os.LookupEnv(fallbackKey); exists {
		return value
	}
	// 3. If neither is found, use the hardcoded default value.
	return defaultValue
}

func setupTTLMonitor(client *mongo.Client, seconds int) error {
	adminDB := client.Database("admin")
	setParamCmd := bson.D{{Key: "setParameter", Value: 1}, {Key: "ttlMonitorSleepSecs", Value: seconds}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := adminDB.RunCommand(ctx, setParamCmd).Err()
	if err != nil {
		log.Println("Note: User may not have admin privileges to set 'ttlMonitorSleepSecs'.")
		return err
	}
	log.Printf("Successfully set TTL monitor interval to %d second(s).", seconds)
	return nil
}
