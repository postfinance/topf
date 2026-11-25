// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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

	if script == "" {
		// only an empty dir on PATH, so no real sops is found either
		t.Setenv("PATH", bin)
		return
	}

	//nolint:gosec // test helper needs an executable file
	if err := os.WriteFile(filepath.Join(bin, "sops"), []byte("#!/bin/sh\n"+script), 0o700); err != nil {
		t.Fatalf("writing fake sops: %v", err)
	}

	// keep system dirs so the script can use cat, but put bin first
	t.Setenv("PATH", bin+string(os.PathListSeparator)+"/usr/bin"+string(os.PathListSeparator)+"/bin")
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return string(got)
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
			script: "cat >/dev/null\necho 'config file not found, or has no creation rules, and no keys provided through command line options' >&2\nexit 1\n",
			want:   testBundle,
		},
		{
			name:   "plaintext when no creation rule matches",
			script: "cat >/dev/null\necho 'error loading config: no matching creation rules found' >&2\nexit 1\n",
			want:   testBundle,
		},
		{
			name:    "error when encryption fails",
			script:  "cat >/dev/null\necho 'could not encrypt data key with PGP key' >&2\nexit 1\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSops(t, tt.script)

			path := filepath.Join(t.TempDir(), "secrets.yaml")
			provider := &filesystemSecrets{path: path}

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

			if got := readFile(t, path); got != tt.want {
				t.Errorf("unexpected file content:\ngot:  %q\nwant: %q", got, tt.want)
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}

			if perm := info.Mode().Perm(); perm != 0o600 {
				t.Errorf("expected file mode 0600, got %o", perm)
			}
		})
	}
}

func TestFilesystemSecretsPutSymlink(t *testing.T) {
	fakeSops(t, "cat >/dev/null\necho 'sops: encrypted'\n")

	dir := t.TempDir()
	target := filepath.Join(dir, "real", "secrets.yaml")
	link := filepath.Join(dir, "secrets.yaml")

	if err := os.Mkdir(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}

	t.Run("link to missing target creates target", func(t *testing.T) {
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}

		provider := &filesystemSecrets{path: link}
		if err := provider.Put("test", []byte(testBundle)); err != nil {
			t.Fatalf("Put() error: %v", err)
		}

		if got := readFile(t, target); got != "sops: encrypted\n" {
			t.Errorf("target content = %q", got)
		}
	})

	t.Run("link is preserved", func(t *testing.T) {
		info, err := os.Lstat(link)
		if err != nil {
			t.Fatal(err)
		}

		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected %s to remain a symlink, mode %v", link, info.Mode())
		}
	})
}
