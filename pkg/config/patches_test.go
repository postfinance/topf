// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
	"os"
	"testing"

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

func TestLoadFile(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantPatch bool // true → expect a non-nil, non-JSON patch
		// applyAndCheck, if non-nil, applies the patch to minimalMachineConfig
		// and runs extra assertions on the merged YAML bytes.
		applyAndCheck func(t *testing.T, merged []byte)
	}{
		{
			name:      "empty file is skipped",
			content:   "",
			wantPatch: false,
		},
		{
			name:      "whitespace-only file is skipped",
			content:   "   \n\n  ",
			wantPatch: false,
		},
		{
			name:      "comment-only file is skipped",
			content:   "# intentionally empty\n",
			wantPatch: false,
		},
		{
			name: "valid strategic merge patch is loaded and applied",
			content: `machine:
  network:
    hostname: mynode`,
			wantPatch: true,
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
			wantPatch: true,
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
			wantPatch: false,
		},
	}

	p := &PatchContext{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "*.yaml")
			if err != nil {
				t.Fatalf("creating temp file: %v", err)
			}

			if _, err := f.WriteString(tt.content); err != nil {
				t.Fatalf("writing temp file: %v", err)
			}

			f.Close()

			patch, err := p.loadFile(f.Name())
			if err != nil {
				t.Fatalf("loadFile() unexpected error: %v", err)
			}

			if (patch != nil) != tt.wantPatch {
				t.Fatalf("loadFile() patch = %v, wantPatch %v", patch, tt.wantPatch)
			}

			if patch == nil || tt.applyAndCheck == nil {
				return
			}

			out, err := configpatcher.Apply(configpatcher.WithBytes([]byte(minimalMachineConfig)), []configpatcher.Patch{patch})
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

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "completely empty",
			content:  "",
			expected: true,
		},
		{
			name: "only whitespace",
			content: `

  `,
			expected: true,
		},
		{
			name: "only comments",
			content: `# This is a comment
# Another comment`,
			expected: true,
		},
		{
			name:     "empty yaml map",
			content:  "{}",
			expected: true,
		},
		{
			name:     "empty yaml list",
			content:  "[]",
			expected: true,
		},
		{
			name:     "yaml null",
			content:  "null",
			expected: true,
		},
		{
			name:     "yaml tilde (null)",
			content:  "~",
			expected: true,
		},
		// Scalar types: yaml.v3 decodes these into bool/int/float64/string/time.Time,
		// none of which match the nil/empty-map/empty-slice checks, so all must
		// return false (not empty).
		{
			name:     "boolean scalar",
			content:  "true",
			expected: false,
		},
		{
			name:     "integer scalar",
			content:  "42",
			expected: false,
		},
		{
			name:     "float scalar",
			content:  "3.14",
			expected: false,
		},
		{
			name:     "string scalar",
			content:  "hello",
			expected: false,
		},
		{
			name:     "quoted empty string scalar",
			content:  `""`,
			expected: false,
		},
		{
			name:     "date scalar (decoded as time.Time)",
			content:  "2024-01-01",
			expected: false,
		},
		{
			name: "yaml content",
			content: `machine:
  type: worker`,
			expected: false,
		},
		{
			name: "yaml list with data",
			content: `- op: add
  path: /machine/network`,
			expected: false,
		},
		{
			name: "multi-doc all empty documents",
			content: `---
# comment only
---
`,
			expected: true,
		},
		{
			name: "multi-doc with empty first document and valid second",
			content: `---
# intentionally empty
---
machine:
  network:
    hostname: mynode`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmpty([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("isEmpty() = %v, want %v for content:\n%q", result, tt.expected, tt.content)
			}
		})
	}
}
