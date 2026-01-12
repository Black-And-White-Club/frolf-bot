package roundhandler_integration_tests

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestMain(m *testing.M) {
	log.Println("TestMain started in package roundhandler_integration_tests")

	// 1. Initialize the global Containers (Postgres/NATS)
	globalT := &testing.T{}
	var err error
	testEnv, err = testutils.NewTestEnvironment(globalT)
	if err != nil {
		log.Fatalf("TestMain: Failed to initialize test environment: %v", err)
	}

	// 2. Initialize Shared Application Infrastructure (Router/Module/EventBus)
	// We call this once here to ensure it's ready before any test runs
	ctx := context.Background()
	deps, err := initSharedInfrastructure(ctx, testEnv)
	if err != nil {
		log.Fatalf("TestMain: Failed to initialize application infra: %v", err)
	}
	sharedDeps = deps

	log.Println("TestMain: Global environment and application infra ready.")

	// 3. Run Tests
	exitCode := m.Run()

	// 4. Global Cleanup
	log.Println("TestMain: Running global cleanup...")
	if sharedDeps.RoundModule != nil {
		sharedDeps.RoundModule.Close()
	}
	if sharedDeps.Router != nil {
		sharedDeps.Router.Close()
	}
	if sharedDeps.EventBus != nil {
		sharedDeps.EventBus.Close()
	}
	testEnv.Cleanup()

	os.Exit(exitCode)
}
