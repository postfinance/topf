// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/postfinance/topf/internal/decryption"
)

const testBundle = "cluster:\n  secret: plaintext-secret\n"

// fakeSops puts a fake sops executable running script on PATH. An empty
// script means no sops binary is available at all.
func fakeSops(t *testing.T, script string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("fake sops script requires a POSIX shell")
	}

	bin := t.TempDir()

	if script != "" {
		//nolint:gosec // test helper needs an executable file
		if err := os.WriteFile(filepath.Join(bin, "sops"), []byte("#!/bin/sh\n"+script), 0o700); err != nil {
			t.Fatalf("writing fake sops: %v", err)
		}
	}

	t.Setenv("PATH", bin)
}

func TestFilesystemSecretsPut(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		want    string
		wantErr bool
	}{
		{
			name:   "encrypted with sops",
			script: "cat >/dev/null\necho 'sops: encrypted'\n",
			want:   "sops: encrypted\n",
		},
		{
			name: "plaintext when sops is not installed",
			want: testBundle,
		},
		{
			name:   "plaintext when no sops config exists",
			script: "echo 'config file not found, or has no creation rules, and no keys provided through command line options' >&2\nexit 1\n",
			want:   testBundle,
		},
		{
			name:   "plaintext when no creation rule matches",
			script: "echo 'error loading config: no matching creation rules found' >&2\nexit 1\n",
			want:   testBundle,
		},
		{
			name:    "error when encryption fails",
			script:  "echo 'could not encrypt data key with PGP key' >&2\nexit 1\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSops(t, tt.script)

			path := filepath.Join(t.TempDir(), "secrets.yaml")
			provider := NewFilesystemSecretsProvider(path, decryption.NewCache())

			err := provider.Put("test", []byte(testBundle))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Fatalf("expected no secrets file to be written, stat error: %v", statErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("Put() error: %v", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading secrets file: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("unexpected file content:\ngot:  %q\nwant: %q", got, tt.want)
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}

			if perm := info.Mode().Perm(); perm != 0o600 {
				t.Errorf("expected file mode 0600, got %o", perm)
			}

			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil {
				t.Fatalf("reading dir: %v", err)
			}

			if len(entries) != 1 {
				t.Errorf("expected only secrets.yaml in dir, found %d entries", len(entries))
			}
		})
	}
}
