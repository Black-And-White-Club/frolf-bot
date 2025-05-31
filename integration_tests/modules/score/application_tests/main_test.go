package scoreintegrationtests

import (
	"log"
	"os"
	"testing"
)

// TestMain initializes and cleans up the global test environment.
func TestMain(m *testing.M) {
	log.Println("TestMain started in package scoreintegrationtests")

	// Set test environment
	oldAppEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	// Defer cleanup
	defer func() {
		os.Setenv("APP_ENV", oldAppEnv)
		log.Println("TestMain: cleanup finished.")
	}()

	// Run the tests
	exitCode := m.Run()
	log.Printf("TestMain: finished with exit code: %d", exitCode)
	os.Exit(exitCode)
}
