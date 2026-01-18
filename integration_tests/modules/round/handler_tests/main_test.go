package roundhandler_integration_tests

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestMain(m *testing.M) {
	log.Println("TestMain started in package roundhandler_integration_tests")

	// 1. Setup global environment
	testEnvOnce.Do(func() {
		// Avoid passing &testing.T{} if possible.
		// If NewTestEnvironment requires it, ensure it doesn't call t.Fatal.
		env, err := testutils.NewTestEnvironment(&testing.T{})
		if err != nil {
			log.Fatalf("TestMain: Failed to setup test environment: %v", err)
		}
		testEnv = env
	})

	// 2. Set Env Vars
	oldAppEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	// 3. Run the tests and capture exit code
	exitCode := m.Run()

	// 4. Manual Cleanup (Don't rely on defer before os.Exit)
	log.Println("TestMain: Running global test environment cleanup.")
	os.Setenv("APP_ENV", oldAppEnv)

	if testEnv != nil {
		testEnv.Cleanup()
	}

	// Only shutdown the pool if testEnv.Cleanup() doesn't already do it
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	testutils.ShutdownContainerPool(ctx)

	log.Printf("TestMain: Finished with exit code: %d", exitCode)

	// 5. Finally Exit
	os.Exit(exitCode)
}
