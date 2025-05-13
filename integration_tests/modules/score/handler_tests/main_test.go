// File: integration_tests/scorehandler_integration_tests/main_test.go
package scorehandler_integration_tests

import (
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

// TestMain initializes and cleans up the test environment for score handler integration tests.
func TestMain(m *testing.M) {
	log.Println("TestMain started in package scorehandler_integration_tests")

	// Initialize test environment once using the shared NewTestEnvironment function
	testEnvOnce.Do(func() {
		testEnv, testEnvErr = testutils.NewTestEnvironment(nil)
		if testEnvErr != nil {
			log.Fatalf("Failed to setup test environment in scorehandler_integration_tests: %v", testEnvErr)
		}
		log.Printf("TestMain in scorehandler_integration_tests: testEnv initialized successfully: %v", testEnv != nil)
	})

	// If initialization failed, ensure we exit
	if testEnvErr != nil {
		os.Exit(1)
	}

	defer func() {
		log.Println("TestMain defer cleanup started in scorehandler_integration_tests")
		// Add a small diagnostic delay here before the main cleanup
		log.Println("Adding diagnostic delay in TestMain cleanup...")
		time.Sleep(1 * time.Second) // Adjust duration as needed for testing
		log.Println("Diagnostic delay finished in TestMain cleanup.")

		if testEnv != nil {
			testEnv.Cleanup()
		}
		log.Println("TestMain defer cleanup finished in scorehandler_integration_tests")
	}()

	log.Println("TestMain running tests with m.Run() in scorehandler_integration_tests")
	exitCode := m.Run()
	log.Printf("TestMain m.Run() finished with exit code: %d", exitCode)

	os.Exit(exitCode)
}

func GetTestEnv() *testutils.TestEnvironment {
	if testEnv == nil || testEnvErr != nil {
		log.Fatalf("Attempted to get test environment before successful initialization. Error: %v", testEnvErr)
	}
	return testEnv
}
