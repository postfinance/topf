// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeTestConfig(t *testing.T, dir, content string) string {
	t.Helper()

	path := filepath.Join(dir, "topf.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestLoadFromFile(t *testing.T) {
	t.Run("pathResolution", func(t *testing.T) {
		tests := []struct {
			name           string
			extraYAML      []string
			mkdir          string
			wantPatchesDir func(absDir string) string
			wantSecrets    func(absDir string) string
		}{
			{
				name:           "defaults",
				wantPatchesDir: func(d string) string { return d },
				wantSecrets:    func(d string) string { return filepath.Join(d, "secrets.yaml") },
			},
			{
				name:           "relativePatchesDir",
				extraYAML:      []string{"patchesDir: configs"},
				mkdir:          "configs",
				wantPatchesDir: func(d string) string { return filepath.Join(d, "configs") },
				wantSecrets:    func(d string) string { return filepath.Join(d, "secrets.yaml") },
			},
			{
				name:           "absolutePatchesDir",
				extraYAML:      []string{"patchesDir: /abs/patches"},
				wantPatchesDir: func(_ string) string { return "/abs/patches" },
				wantSecrets:    func(d string) string { return filepath.Join(d, "secrets.yaml") },
			},
			{
				name:           "relativeSecretsPath",
				extraYAML:      []string{"secretsPath: mine.yaml"},
				wantPatchesDir: func(d string) string { return d },
				wantSecrets:    func(d string) string { return filepath.Join(d, "mine.yaml") },
			},
			{
				name:           "absoluteSecretsPath",
				extraYAML:      []string{"secretsPath: /etc/mine.yaml"},
				wantPatchesDir: func(d string) string { return d },
				wantSecrets:    func(_ string) string { return "/etc/mine.yaml" },
			},
			{
				name:           "bothRelative",
				extraYAML:      []string{"patchesDir: _cfg", "secretsPath: mine.yaml"},
				mkdir:          "_cfg",
				wantPatchesDir: func(d string) string { return filepath.Join(d, "_cfg") },
				wantSecrets:    func(d string) string { return filepath.Join(d, "mine.yaml") },
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sub := t.TempDir()
				absDir, _ := filepath.Abs(sub)

				if tt.mkdir != "" {
					os.Mkdir(filepath.Join(sub, tt.mkdir), 0o755)
				}

				yaml := "clusterName: test\nclusterEndpoint: https://1.2.3.4:6443\nkubernetesVersion: 1.30.0\n"
				for _, line := range tt.extraYAML {
					yaml += line + "\n"
				}
				yaml += "nodes:\n  - host: n1\n    role: worker\n"

				cfg, _, err := LoadFromFile(writeTestConfig(t, sub, yaml))
				if err != nil {
					t.Fatal(err)
				}
				if cfg.PatchesDir != tt.wantPatchesDir(absDir) {
					t.Errorf("PatchesDir: got %s, want %s", cfg.PatchesDir, tt.wantPatchesDir(absDir))
				}
				if cfg.SecretsPath != tt.wantSecrets(absDir) {
					t.Errorf("SecretsPath: got %s, want %s", cfg.SecretsPath, tt.wantSecrets(absDir))
				}
			})
		}
	})

	t.Run("deprecatedConfigDir", func(t *testing.T) {
		_, _, err := LoadFromFile(writeTestConfig(t, t.TempDir(), `clusterName: test
clusterEndpoint: https://1.2.3.4:6443
kubernetesVersion: 1.30.0
configDir: _config
nodes:
  - host: n1
    role: worker
`))
		if err == nil {
			t.Fatal("expected error for deprecated configDir field")
		}
		if !strings.Contains(err.Error(), "configDir") || !strings.Contains(err.Error(), "patchesDir") {
			t.Errorf("expected error mentioning configDir and patchesDir, got: %v", err)
		}
	})
}

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
