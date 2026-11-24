// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/postfinance/topf/internal/secrets"
)

const testConfig = "clusterName: test\nclusterEndpoint: https://1.2.3.4:6443\n"

func writeFile(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("relative paths resolve against the config directory", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "topf.yaml")
		writeFile(t, path, testConfig+"patchesDir: patches\n", 0o600)

		cfg, err := loadConfig(path, secrets.NewResolver(nil))
		if err != nil {
			t.Fatal(err)
		}

		if want := filepath.Join(dir, "patches"); cfg.PatchesDir != want {
			t.Errorf("PatchesDir = %s, want %s", cfg.PatchesDir, want)
		}
	})

	t.Run("nodes provider nodes are merged and sorted", func(t *testing.T) {
		dir := t.TempDir()
		provider := filepath.Join(dir, "provider.sh")
		writeFile(t, provider, `#!/bin/sh
cat <<'EOF'
- {host: c0, role: control-plane}
- {host: w1, role: worker}
EOF
`, 0o700)

		path := filepath.Join(dir, "topf.yaml")
		writeFile(t, path, testConfig+"nodesProvider: "+provider+"\nnodes:\n  - {host: w2, role: worker}\n", 0o600)

		cfg, err := loadConfig(path, secrets.NewResolver(nil))
		if err != nil {
			t.Fatal(err)
		}

		var got []string
		for _, n := range cfg.Nodes {
			got = append(got, n.Host)
		}

		if want := []string{"c0", "w1", "w2"}; !slices.Equal(got, want) {
			t.Errorf("nodes = %v, want %v", got, want)
		}
	})

	t.Run("missing config file is an error", func(t *testing.T) {
		if _, err := loadConfig(filepath.Join(t.TempDir(), "topf.yaml"), secrets.NewResolver(nil)); err == nil {
			t.Fatal("expected error")
		}
	})
}
