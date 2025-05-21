package scorehandler_integration_tests

import (
	"log"
	"os"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestMain(m *testing.M) {
	log.Println("TestMain started in package scorehandler_integration_tests")

	testEnvOnce.Do(func() {
		log.Println("TestMain: Initializing global test environment...")
		testEnv, testEnvErr = testutils.NewTestEnvironment(nil)
		if testEnvErr != nil {
			log.Printf("TestMain: Failed to setup test environment: %v", testEnvErr)
		} else {
			log.Println("TestMain: Global test environment initialized successfully.")
		}
	})

	if testEnvErr != nil {
		log.Fatalf("Exiting due to failed test environment initialization: %v", testEnvErr)
	}

	oldAppEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	defer func() {
		log.Println("TestMain defer: Running global test environment cleanup.")

		os.Setenv("APP_ENV", oldAppEnv)

		if testEnv != nil {
			testEnv.Cleanup()
		}
		log.Println("TestMain defer: Global test environment cleanup finished.")
		log.Println("TestMain defer: All cleanup complete.")
	}()

	log.Println("TestMain: Running tests with m.Run()...")
	exitCode := m.Run()
	log.Printf("TestMain: m.Run() finished with exit code: %d", exitCode)

	os.Exit(exitCode)
}
