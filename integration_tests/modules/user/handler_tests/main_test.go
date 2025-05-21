package userhandler_integration_tests

import (
	"log"
	"os"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// TestMain initializes and cleans up the global test environment for user handler integration tests.
// Per-test setup (router, event bus, module) is handled within individual test functions using the global testEnv.
func TestMain(m *testing.M) {
	log.Println("TestMain started in package userhandler_integration_tests")

	// Initialize the global test environment exactly once.
	testEnvOnce.Do(func() {
		log.Println("TestMain: Initializing global test environment...")
		// Pass nil config to use default test config
		// Note: testutils.NewTestEnvironment typically takes a *testing.T or similar
		// for logging within the setup itself. Using nil here might suppress some logs
		// from testutils if it relies on t.Logf, but is consistent with the leaderboard example.
		// Consider passing &testing.T{} if you need that logging.
		testEnv, testEnvErr = testutils.NewTestEnvironment(nil)
		if testEnvErr != nil {
			log.Printf("TestMain: Failed to setup test environment: %v", testEnvErr)
			// Do not call os.Exit here yet, let the deferred cleanup run first if possible.
		} else {
			log.Println("TestMain: Global test environment initialized successfully.")
		}
	})

	// If test environment initialization failed, log the error and exit immediately
	// after any deferred cleanup that might still be relevant (like closing logs).
	if testEnvErr != nil {
		// We can't proceed if the environment isn't set up.
		log.Fatalf("Exiting due to failed test environment initialization: %v", testEnvErr)
		// os.Exit(1) is handled by log.Fatalf
	}

	// Set the APP_ENV environment variable to "test" for the entire test run
	// This is important to enable test-specific behavior in various components (e.g., NoOpMetrics)
	oldAppEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	// Defer the global test environment cleanup. This will run after m.Run() finishes.
	defer func() {
		log.Println("TestMain defer: Running global test environment cleanup.")

		// Restore the original APP_ENV value
		os.Setenv("APP_ENV", oldAppEnv)

		if testEnv != nil {
			testEnv.Cleanup()
		}
		log.Println("TestMain defer: Global test environment cleanup finished.")
		log.Println("TestMain defer: All cleanup complete.")
	}()

	log.Println("TestMain: Running tests with m.Run()...")
	// Run the tests. The exit code captures the test results.
	exitCode := m.Run()
	log.Printf("TestMain: m.Run() finished with exit code: %d", exitCode)

	// os.Exit with the test result code. Deferred functions will run before exiting.
	os.Exit(exitCode)
}
