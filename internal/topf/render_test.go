// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siderolabs/talos/pkg/machinery/config/encoder"
)

// hostnameProvider is a patches provider setting the hostname of every node
const hostnameProvider = `cat <<EOF
apiVersion: v1alpha1
kind: HostnameConfig
auto: "off"
hostname: from-provider
EOF
`

func writeProviderScript(t *testing.T, script string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "provider.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}

	return path
}

// newTestRuntime writes a topf.yaml for a two node cluster into dir and
// creates a runtime from it.
func newTestRuntime(t *testing.T, dir string, extraYAML ...string) (Topf, error) {
	t.Helper()

	yaml := "clusterName: test\nclusterEndpoint: https://10.0.0.100:6443\nkubernetesVersion: 1.35.1\n"
	for _, line := range extraYAML {
		yaml += line + "\n"
	}
	yaml += "nodes:\n  - host: cp1\n    role: control-plane\n  - host: w1\n    role: worker\n"

	configPath := filepath.Join(dir, "topf.yaml")
	if err := os.WriteFile(configPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	return NewTopfRuntime(RuntimeConfig{
		ConfigPath: configPath,
		Logger:     slog.New(slog.DiscardHandler),
	})
}

// renderConfigs renders the machine configs of all nodes, keyed by host
func renderConfigs(t *testing.T, rt Topf) map[string]string {
	t.Helper()

	nodes, err := rt.Render(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}

	configs := make(map[string]string, len(nodes))

	for _, node := range nodes {
		if node.Error != nil {
			t.Fatalf("%s: %v", node.Node.Host, node.Error)
		}

		configBytes, err := node.ConfigProvider().EncodeBytes(encoder.WithComments(encoder.CommentsDisabled))
		if err != nil {
			t.Fatal(err)
		}

		configs[node.Node.Host] = string(configBytes)
	}

	return configs
}

func TestRenderWithPatchesProvider(t *testing.T) {
	t.Run("localPatchesOverrideProvider", func(t *testing.T) {
		dir := t.TempDir()

		nodePatchDir := filepath.Join(dir, "node", "cp1")
		os.MkdirAll(nodePatchDir, 0o755)
		os.WriteFile(filepath.Join(nodePatchDir, "01-hostname.yaml"), []byte("apiVersion: v1alpha1\nkind: HostnameConfig\nauto: \"off\"\nhostname: from-file\n"), 0o644)

		rt, err := newTestRuntime(t, dir, "patchesProvider: "+writeProviderScript(t, hostnameProvider))
		if err != nil {
			t.Fatal(err)
		}

		configs := renderConfigs(t, rt)

		want := map[string]string{
			"cp1": "hostname: from-file",
			"w1":  "hostname: from-provider",
		}

		for host, hostname := range want {
			if !strings.Contains(configs[host], hostname) {
				t.Errorf("%s: rendered config missing %q; got:\n%s", host, hostname, configs[host])
			}
		}
	})

	t.Run("patchesDirIsOptional", func(t *testing.T) {
		dir := t.TempDir()

		if _, err := newTestRuntime(t, dir, "patchesDir: missing"); err == nil {
			t.Fatal("expected error for missing patchesDir without patches provider")
		}

		rt, err := newTestRuntime(t, dir, "patchesDir: missing", "patchesProvider: "+writeProviderScript(t, hostnameProvider))
		if err != nil {
			t.Fatal(err)
		}

		for host, config := range renderConfigs(t, rt) {
			if !strings.Contains(config, "hostname: from-provider") {
				t.Errorf("%s: rendered config missing provider hostname; got:\n%s", host, config)
			}
		}
	})

	t.Run("providerFailure", func(t *testing.T) {
		rt, err := newTestRuntime(t, t.TempDir(), "patchesProvider: "+writeProviderScript(t, "exit 1"))
		if err != nil {
			t.Fatal(err)
		}

		nodes, err := rt.Render(context.Background(), false)
		if err != nil {
			t.Fatal(err)
		}

		for _, node := range nodes {
			if node.Error == nil || !strings.Contains(node.Error.Error(), "patches from provider") {
				t.Errorf("%s: expected patches provider error, got: %v", node.Node.Host, node.Error)
			}
		}
	})
}
