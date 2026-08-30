// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"testing"

	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
)

// TestParsePatchesUnattendedInstallConfig is a regression test for Talos v1.14
// support: the UnattendedInstallConfig multi-document config kind must parse.
// With machinery < v1.14 the kind is not registered and patch loading fails
// with `"UnattendedInstallConfig" "v1alpha1": not registered`.
func TestParsePatchesUnattendedInstallConfig(t *testing.T) {
	content := `apiVersion: v1alpha1
kind: UnattendedInstallConfig
installer:
  image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.14.0
provisioning:
  diskSelector:
    match: disk.dev_path == "/dev/vda"
  wipe: false
`

	patches, err := parsePatches([]byte(content))
	if err != nil {
		t.Fatalf("parsePatches() error = %v", err)
	}

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}

	// the patch must be a strategic merge patch containing the doc
	smp, ok := patches[0].(configpatcher.StrategicMergePatch)
	if !ok {
		t.Fatalf("expected StrategicMergePatch, got %T", patches[0])
	}

	found := false

	for _, doc := range smp.Documents() {
		if doc.Kind() == "UnattendedInstallConfig" && doc.APIVersion() == "v1alpha1" {
			found = true
		}
	}

	if !found {
		t.Error("patch does not contain UnattendedInstallConfig document")
	}
}