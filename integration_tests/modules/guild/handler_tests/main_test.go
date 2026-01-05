package guildhandlerintegrationtests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Set the APP_ENV to "test"
	err := os.Setenv("APP_ENV", "test")
	if err != nil {
		panic("Failed to set APP_ENV: " + err.Error())
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}
