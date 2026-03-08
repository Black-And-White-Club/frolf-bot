package main

import "testing"

func TestRuntimeServiceVersion(t *testing.T) {
	originalVersion := Version
	t.Cleanup(func() {
		Version = originalVersion
	})

	t.Run("prefers explicit environment override", func(t *testing.T) {
		t.Setenv("SERVICE_VERSION", "v9.9.9")
		Version = "build-version"

		if got := runtimeServiceVersion("configured-version"); got != "v9.9.9" {
			t.Fatalf("expected env override, got %q", got)
		}
	})

	t.Run("falls back to configured version", func(t *testing.T) {
		t.Setenv("SERVICE_VERSION", "")
		Version = "build-version"

		if got := runtimeServiceVersion("configured-version"); got != "configured-version" {
			t.Fatalf("expected configured version, got %q", got)
		}
	})

	t.Run("falls back to build version", func(t *testing.T) {
		t.Setenv("SERVICE_VERSION", "")
		Version = "build-version"

		if got := runtimeServiceVersion(""); got != "build-version" {
			t.Fatalf("expected build version, got %q", got)
		}
	})

	t.Run("defaults to dev when everything is empty", func(t *testing.T) {
		t.Setenv("SERVICE_VERSION", "")
		Version = ""

		if got := runtimeServiceVersion(""); got != "dev" {
			t.Fatalf("expected dev fallback, got %q", got)
		}
	})
}
