package handler_tests

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// Global test environment shared across all tests
var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

// TestMain initializes and cleans up the test environment for handler integration tests.
func TestMain(m *testing.M) {
	log.Println("TestMain started in package handler_tests")

	// Initialize test environment once
	testEnvOnce.Do(func() {
		testEnv, testEnvErr = testutils.NewTestEnvironment(&testing.T{})
		if testEnvErr != nil {
			log.Fatalf("Failed to setup test environment: %v", testEnvErr)
		}
		log.Printf("TestMain: testEnv initialized successfully: %v", testEnv != nil)
	})

	// Setup cleanup
	defer func() {
		log.Println("TestMain defer cleanup started")
		if testEnv != nil {
			testEnv.Cleanup()
		}
		log.Println("TestMain defer cleanup finished")
	}()

	// Run tests
	log.Println("TestMain running tests with m.Run()")
	exitCode := m.Run()
	log.Printf("TestMain m.Run() finished with exit code: %d", exitCode)

	os.Exit(exitCode)
}
