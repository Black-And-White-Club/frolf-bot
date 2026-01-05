package guildintegrationtests

import (
	"log"
	"os"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// TestMain initializes and cleans up the global test environment.
// Per-test setup (service) is handled within test functions.
func TestMain(m *testing.M) {
	log.Println("TestMain started in package guildintegrationtests")

	// Initialize the global test environment exactly once.
	testEnvOnce.Do(func() {
		log.Println("TestMain: Initializing global test environment...")

		// Create a test instance for the global environment
		// This is needed because NewTestEnvironment now requires a *testing.T
		globalTestInstance := &testing.T{}
		testEnv, testEnvErr = testutils.NewTestEnvironment(globalTestInstance)
		if testEnvErr != nil {
			log.Printf("TestMain: Failed to setup test environment: %v", testEnvErr)
		} else {
			log.Println("TestMain: Global test environment initialized successfully.")
		}
	})

	// If test environment initialization failed, log the error and exit
	if testEnvErr != nil {
		log.Fatalf("Exiting due to failed test environment initialization: %v", testEnvErr)
	}

	// Set the APP_ENV environment variable to "test" for the entire test run
	oldAppEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	// Defer the global test environment cleanup
	defer func() {
		log.Println("TestMain defer: Running global test environment cleanup.")
		// Restore the original APP_ENV value
		os.Setenv("APP_ENV", oldAppEnv)
		if testEnv != nil {
			log.Println("TestMain defer: Performing final container cleanup...")
			testEnv.Cleanup()
		}
		log.Println("TestMain defer: Global test environment cleanup finished.")
	}()

	log.Println("TestMain: Running tests with m.Run()...")
	// Add periodic container recreation logging
	if testEnv != nil {
		log.Printf("TestMain: Container recreation configured every %d tests", 20)
	}

	// Run the tests
	exitCode := m.Run()
	log.Printf("TestMain: m.Run() finished with exit code: %d", exitCode)

	// Exit with the test result code
	os.Exit(exitCode)
}
