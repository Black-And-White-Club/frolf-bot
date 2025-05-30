package roundhandler_integration_tests

import (
	"log"
	"os"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// TestMain initializes and cleans up the global test environment.
// Per-test setup (router, event bus, module) is now handled within test functions.
func TestMain(m *testing.M) {
	log.Println("TestMain started in package roundhandler_integration_tests")

	// Initialize the global test environment exactly once.
	testEnvOnce.Do(func() {
		log.Println("TestMain: Initializing global test environment...")

		// Create a test instance for the global environment
		// This is needed because NewTestEnvironment now requires a *testing.T
		globalTestInstance := &testing.T{}
		testEnv, testEnvErr = testutils.NewTestEnvironment(globalTestInstance)
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
	// This is important to enable test-specific behavior in various components
	oldAppEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	// Defer the global test environment cleanup.
	defer func() {
		log.Println("TestMain defer: Running global test environment cleanup.")

		// Restore the original APP_ENV value
		os.Setenv("APP_ENV", oldAppEnv)

		if testEnv != nil {
			// Perform a final container health check and cleanup
			log.Println("TestMain defer: Performing final container cleanup...")
			testEnv.Cleanup()
		}
		log.Println("TestMain defer: Global test environment cleanup finished.")
		log.Println("TestMain defer: All cleanup complete.")
	}()

	log.Println("TestMain: Running tests with m.Run()...")

	// Add periodic container recreation logging
	if testEnv != nil {
		log.Printf("TestMain: Container recreation configured every %d tests", 20)
	}

	// Run the tests. The exit code captures the test results.
	exitCode := m.Run()
	log.Printf("TestMain: m.Run() finished with exit code: %d", exitCode)

	// os.Exit with the test result code. Deferred functions will run before exiting.
	os.Exit(exitCode)
}
