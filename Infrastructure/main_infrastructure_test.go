package infrastructure_test

import (
	"log"
	"os"
	"testing"

	"A2SV_Starter_Project_Blog/testhelper"
)

// This TestMain will be the controller for all tests in the infrastructure_test package.
func TestMain(m *testing.M) {
	// 1. Use a dummy *testing.T to get the shared Redis client.
	// This will trigger the sync.Once in the helper and start the container.
	// The cleanup will be registered to this top-level test context.
	testhelper.GetTestRedisClient(&testing.T{})

	log.Println("✅ Redis container is ready for all infrastructure tests.")

	// 2. Run all the tests in the package.
	exitCode := m.Run()

	// 3. After m.Run() returns, the cleanup function registered by the helper
	// will be automatically called by the test framework.

	// 4. Exit with the result code.
	os.Exit(exitCode)
}
