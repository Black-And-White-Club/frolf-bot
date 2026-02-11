package guildhandlerintegrationtests

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestMain(m *testing.M) {
	log.Println("TestMain started in package guildhandlerintegrationtests")

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
		os.Setenv("APP_ENV", oldAppEnv)
		if testEnv != nil {
			testEnv.Cleanup()
		}
	}()

	exitCode := m.Run()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	testutils.ShutdownContainerPool(ctx)

	os.Exit(exitCode)
}
