// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package patches

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/postfinance/topf/internal/secrets"
	"github.com/postfinance/topf/internal/tmpl"
	"github.com/postfinance/topf/pkg/config"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
)

const minimalMachineConfig = `version: v1alpha1
debug: false
persist: true
machine:
  type: worker
  token: abc.123456789012
  ca:
    crt: ""
    key: ""
  certSANs: []
  kubelet: {}
  network: {}
  install:
    disk: /dev/sda
    image: ""
    bootloader: true
    wipe: false
cluster:
  id: testid
  secret: testsecret
  controlPlane:
    endpoint: https://127.0.0.1:6443
  clusterName: test
  network:
    dnsDomain: cluster.local
    podSubnets:
      - 10.244.0.0/16
    serviceSubnets:
      - 10.96.0.0/12
  token: abc.0123456789abcdef
  secretboxEncryptionSecret: ""
  ca:
    crt: ""
    key: ""
  aggregatorCA:
    crt: ""
    key: ""
  serviceAccount:
    key: ""
  apiServer: {}
  controllerManager: {}
  scheduler: {}
  etcd: {}
`

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()

	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writePatch(t *testing.T, dir, rel, hostname string) {
	t.Helper()
	writeFile(t, dir, rel, "machine:\n  network:\n    hostname: "+hostname+"\n")
}

func symlink(t *testing.T, dir, target, link string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, link)), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(filepath.Join(dir, target), filepath.Join(dir, link)); err != nil {
		t.Fatal(err)
	}
}

func load(dir string, node config.Node) ([]configpatcher.Patch, error) {
	l := &Loader{Dir: dir, Secrets: secrets.NewResolver(nil)}

	return l.Load(&tmpl.Context{
		TopfConfig: &config.TopfConfig{ClusterName: "test-cluster"},
		Node:       tmpl.Node{Node: node},
	})
}

// hostname loads and applies the patches of node and returns the resulting hostname.
func hostname(t *testing.T, dir string, node config.Node) string {
	t.Helper()

	patches, err := load(dir, node)
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

func TestLoadMergeOrder(t *testing.T) {
	dir := t.TempDir()
	writePatch(t, dir, "all/01.yaml", "from-all")
	writePatch(t, dir, "worker/01.yaml", "from-worker")
	writePatch(t, dir, "class/hw/01.yaml", "from-hw")
	writePatch(t, dir, "class/gpu/01.yaml", "from-gpu")
	writePatch(t, dir, "node/other/01.yaml", "from-other-node")

	t.Run("role overrides all", func(t *testing.T) {
		if got := hostname(t, dir, config.Node{Host: "n1", Role: config.RoleWorker}); got != "from-worker" {
			t.Errorf("hostname = %q, want from-worker", got)
		}
	})

	t.Run("classes apply in listed order after role", func(t *testing.T) {
		node := config.Node{Host: "n1", Role: config.RoleWorker, Classes: []string{"gpu", "hw"}}
		if got := hostname(t, dir, node); got != "from-hw" {
			t.Errorf("hostname = %q, want from-hw", got)
		}
	})

	t.Run("node patches override classes", func(t *testing.T) {
		writePatch(t, dir, "node/n2/01.yaml", "from-node")

		node := config.Node{Host: "n2", Role: config.RoleWorker, Classes: []string{"gpu"}}
		if got := hostname(t, dir, node); got != "from-node" {
			t.Errorf("hostname = %q, want from-node", got)
		}
	})

	t.Run("missing class folder is an error", func(t *testing.T) {
		node := config.Node{Host: "n1", Role: config.RoleWorker, Classes: []string{"gpuu"}}
		if _, err := load(dir, node); err == nil || !strings.Contains(err.Error(), `class "gpuu"`) {
			t.Fatalf("expected missing class error, got %v", err)
		}
	})
}

func TestLoadFiles(t *testing.T) {
	t.Run("missing folders yield no patches", func(t *testing.T) {
		patches, err := load(t.TempDir(), config.Node{Host: "n1", Role: config.RoleWorker})
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}

		if len(patches) != 0 {
			t.Errorf("got %d patches, want 0", len(patches))
		}
	})

	t.Run("non-patch files are ignored", func(t *testing.T) {
		dir := t.TempDir()
		writePatch(t, dir, "all/01.yaml", "from-all")
		writeFile(t, dir, "all/README.md", "not: [yaml")

		if got := hostname(t, dir, config.Node{Host: "n1", Role: config.RoleWorker}); got != "from-all" {
			t.Errorf("hostname = %q, want from-all", got)
		}
	})

	t.Run("symlinked folders and files are followed", func(t *testing.T) {
		dir := t.TempDir()
		writePatch(t, dir, "shared/dir/01.yaml", "from-dir")
		writePatch(t, dir, "shared/file.yaml", "from-file")
		symlink(t, dir, "shared/file.yaml", "all/01.yaml")
		symlink(t, dir, "shared/dir", "node/n1")

		if got := hostname(t, dir, config.Node{Host: "n1", Role: config.RoleWorker}); got != "from-dir" {
			t.Errorf("n1 hostname = %q, want from-dir", got)
		}

		if got := hostname(t, dir, config.Node{Host: "n2", Role: config.RoleWorker}); got != "from-file" {
			t.Errorf("n2 hostname = %q, want from-file", got)
		}
	})

	t.Run("dangling symlink is an error", func(t *testing.T) {
		dir := t.TempDir()
		writePatch(t, dir, "all/01.yaml", "from-all")
		symlink(t, dir, "does-not-exist.yaml", "all/02.yaml")

		if _, err := load(dir, config.Node{Host: "n1", Role: config.RoleWorker}); err == nil {
			t.Fatal("expected error for dangling symlink, got nil")
		}
	})
}

func TestLoadTemplate(t *testing.T) {
	t.Run("template is rendered with context", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "all/01.yaml.tpl", "machine:\n  network:\n    hostname: {{ .ClusterName }}-{{ .Node.Host }}\n")

		if got := hostname(t, dir, config.Node{Host: "n1", Role: config.RoleWorker}); got != "test-cluster-n1" {
			t.Errorf("hostname = %q, want test-cluster-n1", got)
		}
	})

	t.Run("encrypted template is rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "all/01.yaml.tpl", "machine:\n  token: ENC[AES256_GCM,data:abc]\n")

		_, err := load(dir, config.Node{Host: "n1", Role: config.RoleWorker})
		if !errors.Is(err, tmpl.ErrEncryptedTemplate) {
			t.Fatalf("expected ErrEncryptedTemplate, got %v", err)
		}
	})
}

func TestParsePatches(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantPatchCount int
		wantErr        bool
		// applyAndCheck, if non-nil, applies the patch to minimalMachineConfig
		// and runs extra assertions on the merged YAML bytes.
		applyAndCheck func(t *testing.T, merged []byte)
	}{
		{
			name:           "empty file is skipped",
			content:        "",
			wantPatchCount: 0,
			wantErr:        false,
		},
		{
			name:           "whitespace-only file is skipped",
			content:        "   \n\n  ",
			wantPatchCount: 0,
			wantErr:        false,
		},
		{
			name:           "comment-only file is skipped",
			content:        "# intentionally empty\n",
			wantPatchCount: 0,
			wantErr:        false,
		},
		{
			name: "valid strategic merge patch is loaded and applied",
			content: `machine:
  network:
    hostname: mynode`,
			wantPatchCount: 1,
			wantErr:        false,
			applyAndCheck: func(t *testing.T, merged []byte) {
				t.Helper()
				if !bytes.Contains(merged, []byte("hostname: mynode")) {
					t.Errorf("merged config missing expected hostname; got:\n%s", merged)
				}
			},
		},
		{
			// Regression test for issue #36: isEmpty() used to parse only the
			// first YAML document. A file whose first document is empty (e.g.
			// just a separator + comment) was incorrectly skipped in full, even
			// when subsequent documents contained valid patches.
			name: "multi-doc YAML with empty first document loads second document (regression #36)",
			content: `---
# intentionally empty
---
machine:
  network:
    hostname: mynode`,
			wantPatchCount: 1,
			wantErr:        false,
			applyAndCheck: func(t *testing.T, merged []byte) {
				t.Helper()
				if !bytes.Contains(merged, []byte("hostname: mynode")) {
					t.Errorf("merged config missing expected hostname; got:\n%s", merged)
				}
			},
		},
		{
			name: "multi-doc YAML where all documents are empty is skipped",
			content: `---
# first doc is empty
---
# second doc is also empty
`,
			wantPatchCount: 0,
			wantErr:        false,
		},
		{
			name:    "invalid patch with unknown field is rejected",
			content: `foo: bar`,
			wantErr: true,
		},
		{
			name: "json patch is rejected",
			content: `- op: replace
  path: /machine/network/hostname
  value: worker1`,
			wantErr: true,
		},
		{
			name:    "string is rejected",
			content: "foobar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches, err := parsePatches([]byte(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Fatal("parsePatches() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePatches() error: %v", err)
			}

			if tt.wantPatchCount != len(patches) {
				t.Fatalf("unexpected patch count: got %d, want %d", len(patches), tt.wantPatchCount)
			}

			if tt.applyAndCheck == nil {
				return
			}

			out, err := configpatcher.Apply(configpatcher.WithBytes([]byte(minimalMachineConfig)), patches)
			if err != nil {
				t.Fatalf("configpatcher.Apply() error: %v", err)
			}

			merged, err := out.Bytes()
			if err != nil {
				t.Fatalf("out.Bytes() error: %v", err)
			}

			tt.applyAndCheck(t, merged)
		})
	}
}
