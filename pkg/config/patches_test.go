// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"bytes"
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
