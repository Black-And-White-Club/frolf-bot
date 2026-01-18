package guildhandlerintegrationtests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestMain(m *testing.M) {
	// Set the APP_ENV to "test"
	err := os.Setenv("APP_ENV", "test")
	if err != nil {
		panic("Failed to set APP_ENV: " + err.Error())
	}

	exitCode := m.Run()

	// Shutdown container pool
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	testutils.ShutdownContainerPool(ctx)

	os.Exit(exitCode)
}
