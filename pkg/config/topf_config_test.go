// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"slices"
	"strings"
	"testing"
)

const testConfig = "clusterName: test\nclusterEndpoint: https://1.2.3.4:6443\n"

func hosts(nodes []Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Host)
	}

	return out
}

func TestParse(t *testing.T) {
	t.Run("paths", func(t *testing.T) {
		tests := []struct {
			name            string
			yaml            string
			wantPatchesDir  string
			wantSecretsPath string
		}{
			{"defaults", "", "/base", "/base/secrets.yaml"},
			{"relative", "patchesDir: _cfg\nsecretsPath: mine.yaml\n", "/base/_cfg", "/base/mine.yaml"},
			{"absolute", "patchesDir: /abs/patches\nsecretsPath: /etc/mine.yaml\n", "/abs/patches", "/etc/mine.yaml"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg, err := Parse([]byte(testConfig+tt.yaml), "/base")
				if err != nil {
					t.Fatal(err)
				}

				if cfg.PatchesDir != tt.wantPatchesDir {
					t.Errorf("PatchesDir = %s, want %s", cfg.PatchesDir, tt.wantPatchesDir)
				}

				if cfg.SecretsPath != tt.wantSecretsPath {
					t.Errorf("SecretsPath = %s, want %s", cfg.SecretsPath, tt.wantSecretsPath)
				}
			})
		}
	})

	t.Run("relative base dir is an error", func(t *testing.T) {
		if _, err := Parse([]byte(testConfig), "base"); err == nil {
			t.Fatal("expected error for relative base dir")
		}
	})

	t.Run("deprecated configDir is an error", func(t *testing.T) {
		_, err := Parse([]byte(testConfig+"configDir: _config\n"), "/base")
		if err == nil || !strings.Contains(err.Error(), "patchesDir") {
			t.Fatalf("expected configDir deprecation error, got %v", err)
		}
	})

	t.Run("invalid yaml is an error", func(t *testing.T) {
		if _, err := Parse([]byte("clusterName: ["), "/base"); err == nil {
			t.Fatal("expected decode error")
		}
	})

	t.Run("nodes are sorted", func(t *testing.T) {
		cfg, err := Parse([]byte(testConfig+`nodes:
  - {host: w2, role: worker}
  - {host: c1, role: control-plane}
  - {host: w1, role: worker}
`), "/base")
		if err != nil {
			t.Fatal(err)
		}

		if got, want := hosts(cfg.Nodes), []string{"c1", "w1", "w2"}; !slices.Equal(got, want) {
			t.Errorf("nodes = %v, want %v", got, want)
		}
	})
}

func TestAddNodes(t *testing.T) {
	cfg, err := Parse([]byte(testConfig+"nodes:\n  - {host: w2, role: worker}\n  - {host: c1, role: control-plane}\n"), "/base")
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := ParseNodes([]byte("- {host: w1, role: worker}\n- {host: c0, role: control-plane}\n"))
	if err != nil {
		t.Fatal(err)
	}

	cfg.AddNodes(nodes)

	if got, want := hosts(cfg.Nodes), []string{"c0", "c1", "w1", "w2"}; !slices.Equal(got, want) {
		t.Errorf("nodes = %v, want %v", got, want)
	}
}

func TestParseNodes(t *testing.T) {
	if _, err := ParseNodes([]byte("- {host: n1}\n")); err == nil {
		t.Fatal("expected error for node without role")
	}
}
