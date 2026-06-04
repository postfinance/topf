// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/postfinance/topf/internal/decryption"
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

				cfg, _, err := LoadFromFile(writeTestConfig(t, sub, yaml), "", decryption.NewCache())
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
`), "", decryption.NewCache())
		if err == nil {
			t.Fatal("expected error for deprecated configDir field")
		}
		if !strings.Contains(err.Error(), "configDir") || !strings.Contains(err.Error(), "patchesDir") {
			t.Errorf("expected error mentioning configDir and patchesDir, got: %v", err)
		}
	})
}
