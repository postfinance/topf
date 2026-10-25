// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"reflect"
	"testing"

	"github.com/postfinance/topf/internal/decryption"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
)

// minimalMachineConfig is a bare-minimum Talos worker config sufficient for
// applying strategic merge patches in tests.  It is intentionally kept small;
// only the fields required by the Talos decoder are present.
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

func writeTempTpl(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "*.yaml.tpl")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	f.Close()

	return f.Name()
}

func TestLoadFileTemplate(t *testing.T) {
	ctx := &PatchContext{
		ClusterName:       "test-cluster",
		KubernetesVersion: "1.35.1",
		Data:              map[string]any{"region": "eu-west-1"},
		Node:              &Node{Host: "node1"},
	}

	t.Run("env function resolves environment variable", func(t *testing.T) {
		t.Setenv("TOPF_TEST_VAR", "hello-world")

		f := writeTempTpl(t, "machine:\n  network:\n    hostname: {{ env \"TOPF_TEST_VAR\" }}\n")

		content, _, err := ctx.loadFile(f)
		if err != nil {
			t.Fatalf("loadFile() error: %v", err)
		}

		if !bytes.Contains(content, []byte("hostname: hello-world")) {
			t.Errorf("expected rendered env var, got:\n%s", content)
		}
	})

	t.Run("sprig function is available", func(t *testing.T) {
		f := writeTempTpl(t, "machine:\n  network:\n    hostname: {{ .Data.region | upper | trunc 2 }}\n")

		content, _, err := ctx.loadFile(f)
		if err != nil {
			t.Fatalf("loadFile() error: %v", err)
		}

		if !bytes.Contains(content, []byte("hostname: EU")) {
			t.Errorf("expected rendered sprig functions, got:\n%s", content)
		}
	})

	t.Run("missing key returns error", func(t *testing.T) {
		f := writeTempTpl(t, "machine:\n  network:\n    hostname: {{ .Data.nonexistent }}\n")

		_, _, err := ctx.loadFile(f)
		if err == nil {
			t.Fatal("expected error for missing key, got nil")
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

// patchesProviderFunc adapts a function to a providers.PatchesProvider
type patchesProviderFunc func(ctx context.Context, patchContext []byte) ([]byte, error)

func (f patchesProviderFunc) GetPatches(ctx context.Context, patchContext []byte) ([]byte, error) {
	return f(ctx, patchContext)
}

func TestLoadFromProvider(t *testing.T) {
	ip := netip.MustParseAddr("10.0.0.1")

	patchCtx := &PatchContext{
		ClusterName:       "test-cluster",
		ClusterEndpoint:   "https://10.0.0.100:6443",
		KubernetesVersion: "1.35.1",
		TalosVersion:      "1.14.0",
		SchematicID:       "abc123",
		Data:              map[string]any{"region": "eu-west-1"},
		Node: &Node{
			Host:        "node1",
			IP:          &ip,
			Role:        RoleControlPlane,
			Data:        map[string]any{"rack": "A3"},
			RuntimeData: RuntimeData{Stage: "running"},
		},
		PatchesDir:   "/patches",
		DecryptCache: decryption.NewCache(),
	}

	t.Run("passesPatchContextAsJSON", func(t *testing.T) {
		var got []byte

		provider := patchesProviderFunc(func(_ context.Context, patchContext []byte) ([]byte, error) {
			got = patchContext
			return nil, nil
		})

		if _, err := patchCtx.LoadFromProvider(context.Background(), provider); err != nil {
			t.Fatalf("LoadFromProvider() error: %v", err)
		}

		// keys mirror the fields available in templates (e.g. .Node.Data.rack)
		// and the decryption cache is not part of the payload
		want := `{
			"ClusterName": "test-cluster",
			"ClusterEndpoint": "https://10.0.0.100:6443",
			"KubernetesVersion": "1.35.1",
			"TalosVersion": "1.14.0",
			"SchematicID": "abc123",
			"Data": {"region": "eu-west-1"},
			"Node": {
				"Host": "node1",
				"IP": "10.0.0.1",
				"Role": "control-plane",
				"Data": {"rack": "A3"},
				"TalosVersion": "",
				"SchematicID": "",
				"Factory": "",
				"Platform": "",
				"SecureBoot": false,
				"RuntimeData": {"TalosVersion": "", "SchematicID": "", "Stage": "running"}
			},
			"PatchesDir": "/patches"
		}`

		var gotValue, wantValue any
		if err := json.Unmarshal(got, &gotValue); err != nil {
			t.Fatalf("provider received invalid JSON %s: %v", got, err)
		}

		if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(gotValue, wantValue) {
			t.Errorf("unexpected patch context:\ngot:  %s\nwant: %s", got, want)
		}
	})

	errProvider := errors.New("provider failed")

	tests := []struct {
		name           string
		output         string
		err            error
		wantErr        bool
		wantPatchCount int
		// applyAndCheck, if non-nil, applies the patches to minimalMachineConfig
		// and runs extra assertions on the merged YAML bytes.
		applyAndCheck func(t *testing.T, merged []byte)
	}{
		{
			name:           "emptyOutput",
			output:         "",
			wantPatchCount: 0,
		},
		{
			name: "multiDocumentOutput",
			output: `machine:
  network:
    hostname: from-provider
---
# empty documents are skipped
---
machine:
  install:
    disk: /dev/vda
`,
			wantPatchCount: 2,
			applyAndCheck: func(t *testing.T, merged []byte) {
				t.Helper()

				for _, want := range []string{"hostname: from-provider", "disk: /dev/vda"} {
					if !bytes.Contains(merged, []byte(want)) {
						t.Errorf("merged config missing %q; got:\n%s", want, merged)
					}
				}
			},
		},
		{
			name: "jsonPatchOutput",
			output: `- op: replace
  path: /machine/network/hostname
  value: worker1`,
			wantErr: true,
		},
		{
			name:    "providerError",
			err:     errProvider,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := patchesProviderFunc(func(context.Context, []byte) ([]byte, error) {
				return []byte(tt.output), tt.err
			})

			patches, err := patchCtx.LoadFromProvider(context.Background(), provider)
			if tt.wantErr {
				if err == nil {
					t.Fatal("LoadFromProvider() expected error, got nil")
				}

				if tt.err != nil && !errors.Is(err, tt.err) {
					t.Errorf("expected error wrapping %v, got: %v", tt.err, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("LoadFromProvider() error: %v", err)
			}

			if len(patches) != tt.wantPatchCount {
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
