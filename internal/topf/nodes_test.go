// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"testing"

	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/configpatcher"
)

func TestInstallerImagePatch(t *testing.T) {
	const image = "factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.14.0"

	t.Run("legacy contract emits machine.install.image", func(t *testing.T) {
		contract, err := talosconfig.ParseContractFromVersion("1.13.0")
		if err != nil {
			t.Fatal(err)
		}

		patch, err := installerImagePatch(image, contract)
		if err != nil {
			t.Fatalf("installerImagePatch() error = %v", err)
		}

		smp, ok := patch.(configpatcher.StrategicMergePatch)
		if !ok {
			t.Fatalf("expected StrategicMergePatch, got %T", patch)
		}

		docs := smp.Documents()
		if len(docs) != 1 || docs[0].Kind() != "v1alpha1" {
			t.Fatalf("expected single v1alpha1 document, got %+v", docs)
		}
	})

	t.Run("v1.14 contract emits UnattendedInstallConfig", func(t *testing.T) {
		contract, err := talosconfig.ParseContractFromVersion("1.14.0")
		if err != nil {
			t.Fatal(err)
		}

		patch, err := installerImagePatch(image, contract)
		if err != nil {
			t.Fatalf("installerImagePatch() error = %v", err)
		}

		smp, ok := patch.(configpatcher.StrategicMergePatch)
		if !ok {
			t.Fatalf("expected StrategicMergePatch, got %T", patch)
		}

		docs := smp.Documents()
		if len(docs) != 1 {
			t.Fatalf("expected 1 document, got %d", len(docs))
		}

		if docs[0].Kind() != "UnattendedInstallConfig" {
			t.Fatalf("expected UnattendedInstallConfig kind, got %q", docs[0].Kind())
		}

		unattended, ok := docs[0].(interface {
			InstallerImage() string
		})
		if !ok {
			t.Fatalf("document does not expose InstallerImage(): %T", docs[0])
		}

		if got := unattended.InstallerImage(); got != image {
			t.Errorf("installer image = %q, want %q", got, image)
		}
	})

	t.Run("nil contract (current version) emits UnattendedInstallConfig", func(t *testing.T) {
		patch, err := installerImagePatch(image, nil)
		if err != nil {
			t.Fatalf("installerImagePatch() error = %v", err)
		}

		smp, ok := patch.(configpatcher.StrategicMergePatch)
		if !ok {
			t.Fatalf("expected StrategicMergePatch, got %T", patch)
		}

		if docs := smp.Documents(); len(docs) != 1 || docs[0].Kind() != "UnattendedInstallConfig" {
			t.Fatalf("expected UnattendedInstallConfig document for nil contract, got %+v", docs)
		}
	})
}