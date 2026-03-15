package bettinghandlerintegrationtests

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

// TestMain initializes and cleans up the global test environment.
// Per-test setup (module, router) is handled within test functions.
func TestMain(m *testing.M) {
	log.Println("TestMain started in package bettinghandlerintegrationtests")

	testEnvOnce.Do(func() {
		log.Println("TestMain: Initializing global test environment...")
		globalTestInstance := &testing.T{}
		testEnv, testEnvErr = testutils.NewTestEnvironment(globalTestInstance)
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
			log.Println("TestMain defer: Performing final container cleanup...")
			testEnv.Cleanup()
		}
		log.Println("TestMain defer: Global test environment cleanup finished.")
	}()

	log.Println("TestMain: Running tests with m.Run()...")

	exitCode := m.Run()
	log.Printf("TestMain: m.Run() finished with exit code: %d", exitCode)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	testutils.ShutdownContainerPool(ctx)

	os.Exit(exitCode)
}
