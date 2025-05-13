// integration_tests/modules/leaderboard/main_test.go
package leaderboardintegrationtests

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/uptrace/bun"
)

var (
	sharedDB      *bun.DB
	sharedCtx     context.Context
	sharedTestEnv *testutils.TestEnvironment
)

func TestMain(m *testing.M) {
	var err error

	log.Println("Setting up test environment...")
	sharedTestEnv, err = testutils.NewTestEnvironment(nil)
	if err != nil {
		log.Fatalf("Failed to set up test environment: %v", err)
	}
	log.Println("Test environment setup complete.")

	sharedCtx = sharedTestEnv.Ctx
	sharedDB = sharedTestEnv.DB

	exitCode := m.Run()

	log.Println("Tearing down test environment...")
	sharedTestEnv.Cleanup()
	log.Println("Test environment teardown complete.")

	os.Exit(exitCode)
}
