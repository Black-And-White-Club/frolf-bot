package testutils

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	code := m.Run()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	globalPool.Shutdown(ctx)
	os.Exit(code)
}
