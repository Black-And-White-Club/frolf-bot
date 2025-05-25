// integration_tests/modules/round/application_tests/main_test.go
package roundintegrationtests

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

func TestMain(m *testing.M) {
	// Set the APP_ENV environment variable to "test" for the entire test run
	oldAppEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	exitCode := m.Run()

	// Restore the original APP_ENV value
	os.Setenv("APP_ENV", oldAppEnv)

	log.Println("Tearing down round test environment...")
	if testEnv != nil {
		testEnv.Cleanup()
	}
	log.Println("Round test environment teardown complete.")

	os.Exit(exitCode)
}
