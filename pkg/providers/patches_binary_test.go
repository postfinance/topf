// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeProviderScript(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "provider.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestBinaryPatchesProvider(t *testing.T) {
	t.Run("passesPatchContextOnStdin", func(t *testing.T) {
		// prints the number of arguments, each argument on its own line and then stdin
		provider := NewBinaryPatchesProvider(writeProviderScript(t, `printf '%s\n' "$#" "$@"; cat`))

		patchContext := `{"ClusterName": "my cluster", "Data": {"quote": "\"it's\""}}`

		output, err := provider.GetPatches(context.Background(), []byte(patchContext))
		if err != nil {
			t.Fatal(err)
		}

		want := "1\npatches\n" + patchContext
		if string(output) != want {
			t.Errorf("unexpected invocation: got %q, want %q", output, want)
		}
	})

	t.Run("nonZeroExitCode", func(t *testing.T) {
		provider := NewBinaryPatchesProvider(writeProviderScript(t, "exit 1"))

		if _, err := provider.GetPatches(context.Background(), []byte("{}")); err == nil {
			t.Fatal("expected error for failing provider, got nil")
		}
	})

	t.Run("missingBinary", func(t *testing.T) {
		provider := NewBinaryPatchesProvider(filepath.Join(t.TempDir(), "does-not-exist"))

		if _, err := provider.GetPatches(context.Background(), []byte("{}")); err == nil {
			t.Fatal("expected error for missing provider, got nil")
		}
	})

	t.Run("canceledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		provider := NewBinaryPatchesProvider(writeProviderScript(t, "exit 0"))

		if _, err := provider.GetPatches(ctx, []byte("{}")); !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	})
}
