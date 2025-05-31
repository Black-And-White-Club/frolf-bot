package testutils

import (
	"sync"
	"testing"
)

var (
	globalEnv *TestEnvironment
	envMutex  sync.RWMutex
)

// GetOrCreateTestEnv returns a shared test environment, creating it if needed
func GetOrCreateTestEnv(t *testing.T) *TestEnvironment {
	envMutex.RLock()
	if globalEnv != nil {
		envMutex.RUnlock()
		return globalEnv
	}
	envMutex.RUnlock()

	envMutex.Lock()
	defer envMutex.Unlock()

	// Double-check pattern
	if globalEnv != nil {
		return globalEnv
	}

	env, err := NewTestEnvironment(t)
	if err != nil {
		t.Fatalf("Failed to create test environment: %v", err)
	}

	globalEnv = env

	// Setup cleanup for the entire test suite
	t.Cleanup(func() {
		if globalEnv != nil {
			globalEnv.Cleanup()
			globalEnv = nil
		}
	})

	return globalEnv
}
