// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/postfinance/topf/internal/decryption"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
	"go.yaml.in/yaml/v4"
)

func TestNodeEmptyClass(t *testing.T) {
	var n Node

	err := yaml.Unmarshal([]byte(`
host: n1
role: worker
classes: [gpu, ""]
`), &n)
	if err == nil || !strings.Contains(err.Error(), "empty class") {
		t.Fatalf("expected empty class error, got %v", err)
	}
}

func writePatch(t *testing.T, dir, rel, hostname string) {
	t.Helper()

	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	content := "machine:\n  network:\n    hostname: " + hostname + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadClassesMergeOrder(t *testing.T) {
	dir := t.TempDir()
	writePatch(t, dir, "all/01.yaml", "from-all")
	writePatch(t, dir, "worker/01.yaml", "from-worker")
	writePatch(t, dir, "class/hw/01.yaml", "from-hw")
	writePatch(t, dir, "class/gpu/01.yaml", "from-gpu")
	writePatch(t, dir, "node/other/01.yaml", "from-other-node")

	hostname := func(t *testing.T, node *Node) string {
		t.Helper()

		ctx := &PatchContext{Node: node, PatchesDir: dir, DecryptCache: decryption.NewCache()}

		patches, _, err := ctx.Load()
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}

		out, err := configpatcher.Apply(configpatcher.WithBytes([]byte(minimalMachineConfig)), patches)
		if err != nil {
			t.Fatalf("Apply() error: %v", err)
		}

		cfg, err := out.Config()
		if err != nil {
			t.Fatal(err)
		}

		return cfg.NetworkHostnameConfig().Hostname()
	}

	t.Run("classes apply in listed order after role", func(t *testing.T) {
		if got := hostname(t, &Node{Host: "n1", Role: RoleWorker, Classes: []string{"gpu", "hw"}}); got != "from-hw" {
			t.Errorf("hostname = %q, want from-hw", got)
		}
	})

	t.Run("node patches override classes", func(t *testing.T) {
		writePatch(t, dir, "node/n2/01.yaml", "from-node")

		if got := hostname(t, &Node{Host: "n2", Role: RoleWorker, Classes: []string{"gpu"}}); got != "from-node" {
			t.Errorf("hostname = %q, want from-node", got)
		}
	})

	t.Run("missing class folder is an error", func(t *testing.T) {
		ctx := &PatchContext{
			Node:         &Node{Host: "n1", Role: RoleWorker, Classes: []string{"gpuu"}},
			PatchesDir:   dir,
			DecryptCache: decryption.NewCache(),
		}

		if _, _, err := ctx.Load(); err == nil || !strings.Contains(err.Error(), `class "gpuu"`) {
			t.Fatalf("expected missing class error, got %v", err)
		}
	})
}
