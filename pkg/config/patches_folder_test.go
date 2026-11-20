// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/postfinance/topf/internal/decryption"
	"github.com/siderolabs/talos/pkg/machinery/config/configloader"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating dir: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func hostnamePatch(name string) string {
	return "machine:\n  network:\n    hostname: " + name + "\n"
}

// appliedHostname applies the patches to a minimal config and returns the
// resulting hostname, i.e. the hostname set by the last applied patch.
func appliedHostname(t *testing.T, patches []configpatcher.Patch) string {
	t.Helper()

	cfg, err := configloader.NewFromBytes([]byte(minimalMachineConfig))
	if err != nil {
		t.Fatalf("loading base config: %v", err)
	}

	out, err := configpatcher.Apply(configpatcher.WithConfig(cfg), patches)
	if err != nil {
		t.Fatalf("applying patches: %v", err)
	}

	patched, err := out.Config()
	if err != nil {
		t.Fatalf("getting patched config: %v", err)
	}

	return patched.RawV1Alpha1().MachineConfig.MachineNetwork.NetworkHostname //nolint:staticcheck // fine for tests
}

func newTestPatchContext() *PatchContext {
	return &PatchContext{
		Node:         &Node{Host: "node1"},
		DecryptCache: decryption.NewCache(),
	}
}

func TestLoadFolder(t *testing.T) {
	t.Run("missing folder yields no patches", func(t *testing.T) {
		patches, _, err := newTestPatchContext().loadFolder(filepath.Join(t.TempDir(), "does-not-exist"))
		if err != nil {
			t.Fatalf("loadFolder() error: %v", err)
		}

		if len(patches) != 0 {
			t.Fatalf("expected no patches, got %d", len(patches))
		}
	})

	t.Run("loads matching files in lexical order", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "10-a.yaml"), hostnamePatch("a"))
		writeFile(t, filepath.Join(dir, "20-b.yml"), hostnamePatch("b"))
		writeFile(t, filepath.Join(dir, "sub", "30-c.yaml"), hostnamePatch("c"))

		patches, _, err := newTestPatchContext().loadFolder(dir)
		if err != nil {
			t.Fatalf("loadFolder() error: %v", err)
		}

		if len(patches) != 3 {
			t.Fatalf("expected 3 patches, got %d", len(patches))
		}

		if got := appliedHostname(t, patches); got != "c" {
			t.Errorf("expected last patch to win with hostname c, got %q", got)
		}
	})

	t.Run("ignores backups, hidden files and other extensions", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "10-net.yaml"), hostnamePatch("real"))

		for _, name := range []string{
			"10-net.yaml~",
			"10-net.yaml.bak",
			"10-net.yaml.orig",
			"10-net.yaml.tpl.orig",
			".10-net.yaml.swp",
			".hidden.yaml",
			"README.md",
		} {
			writeFile(t, filepath.Join(dir, name), hostnamePatch("backup"))
		}

		patches, _, err := newTestPatchContext().loadFolder(dir)
		if err != nil {
			t.Fatalf("loadFolder() error: %v", err)
		}

		if len(patches) != 1 {
			t.Fatalf("expected 1 patch, got %d", len(patches))
		}

		if got := appliedHostname(t, patches); got != "real" {
			t.Errorf("expected hostname real, got %q", got)
		}
	})

	t.Run("renders templates", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "10-host.yaml.tpl"), hostnamePatch("{{ .Node.Host }}"))

		patches, _, err := newTestPatchContext().loadFolder(dir)
		if err != nil {
			t.Fatalf("loadFolder() error: %v", err)
		}

		if got := appliedHostname(t, patches); got != "node1" {
			t.Errorf("expected hostname node1, got %q", got)
		}
	})

	t.Run("follows symlinked files and directories", func(t *testing.T) {
		shared := t.TempDir()
		writeFile(t, filepath.Join(shared, "file", "10-a.yaml"), hostnamePatch("a"))
		writeFile(t, filepath.Join(shared, "dir", "20-b.yaml"), hostnamePatch("b"))

		dir := t.TempDir()

		if err := os.Symlink(filepath.Join(shared, "file", "10-a.yaml"), filepath.Join(dir, "10-a.yaml")); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		if err := os.Symlink(filepath.Join(shared, "dir"), filepath.Join(dir, "20-linked")); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		patches, _, err := newTestPatchContext().loadFolder(dir)
		if err != nil {
			t.Fatalf("loadFolder() error: %v", err)
		}

		if len(patches) != 2 {
			t.Fatalf("expected 2 patches, got %d", len(patches))
		}

		if got := appliedHostname(t, patches); got != "b" {
			t.Errorf("expected hostname b, got %q", got)
		}
	})

	t.Run("dangling symlink is an error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "10-a.yaml"), hostnamePatch("a"))

		if err := os.Symlink(filepath.Join(dir, "missing.yaml"), filepath.Join(dir, "20-b.yaml")); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		if _, _, err := newTestPatchContext().loadFolder(dir); err == nil {
			t.Fatal("expected error for dangling symlink, got nil")
		}
	})

	t.Run("dangling symlink as folder is an error", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "all")

		if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), dir); err != nil {
			t.Fatalf("symlink: %v", err)
		}

		if _, _, err := newTestPatchContext().loadFolder(dir); err == nil {
			t.Fatal("expected error for dangling folder symlink, got nil")
		}
	})
}
