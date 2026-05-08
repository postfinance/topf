// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"encoding/base64"
	"net/url"
	"slices"
	"testing"

	"github.com/siderolabs/crypto/x509"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config"
	"github.com/siderolabs/talos/pkg/machinery/config/container"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
)

func mustParseURL(t *testing.T, s string) *url.URL {
	t.Helper()

	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", s, err)
	}

	return u
}

func TestCollectCurrentConfigSecrets(t *testing.T) {
	etcdCACert := []byte("test-etcd-ca-cert")
	etcdCAKey := []byte("test-etcd-ca-key")
	osCACert := []byte("test-os-cert")
	osCAKey := []byte("test-os-key")
	aggregatorCert := []byte("test-aggregator-cert")
	aggregatorKey := []byte("test-aggregator-key")
	saKey := []byte("test-service-account-key")

	clusterSecret := "cluster-secret-value"
	aesSecret := "aes-encryption-secret"
	secretboxSecret := "secretbox-encryption-secret"
	machineToken := "328hom.uqjzh6jnn2eie9oi"

	cfg := &v1alpha1.Config{
		ClusterConfig: &v1alpha1.ClusterConfig{
			ClusterSecret:                    clusterSecret,
			ClusterAESCBCEncryptionSecret:    aesSecret,
			ClusterSecretboxEncryptionSecret: secretboxSecret,
			ClusterName:                      "test-cluster",
			ControlPlane: &v1alpha1.ControlPlaneConfig{
				Endpoint: &v1alpha1.Endpoint{
					URL: mustParseURL(t, "https://127.0.0.1:6443"),
				},
			},
			EtcdConfig: &v1alpha1.EtcdConfig{
				RootCA: &x509.PEMEncodedCertificateAndKey{
					Crt: etcdCACert,
					Key: etcdCAKey,
				},
			},
			ClusterCA: &x509.PEMEncodedCertificateAndKey{
				Crt: []byte("cluster-ca-cert"),
				Key: []byte("cluster-ca-key"),
			},
			ClusterAggregatorCA: &x509.PEMEncodedCertificateAndKey{
				Crt: aggregatorCert,
				Key: aggregatorKey,
			},
			ClusterServiceAccount: &x509.PEMEncodedKey{
				Key: saKey,
			},
		},
		MachineConfig: &v1alpha1.MachineConfig{
			MachineType:  "controlplane",
			MachineToken: machineToken,
			MachineCA: &x509.PEMEncodedCertificateAndKey{
				Crt: osCACert,
				Key: osCAKey,
			},
		},
	}

	var provider talosconfig.Config
	provider, err := container.New(cfg)
	if err != nil {
		t.Fatalf("failed to create config container: %v", err)
	}

	secrets := collectCurrentConfigSecrets(provider)

	assertContains(t, secrets, base64.StdEncoding.EncodeToString(etcdCACert))
	assertContains(t, secrets, base64.StdEncoding.EncodeToString(etcdCAKey))
	assertContains(t, secrets, base64.StdEncoding.EncodeToString(osCACert))
	assertContains(t, secrets, base64.StdEncoding.EncodeToString(osCAKey))
	assertContains(t, secrets, base64.StdEncoding.EncodeToString(aggregatorCert))
	assertContains(t, secrets, base64.StdEncoding.EncodeToString(aggregatorKey))
	assertContains(t, secrets, base64.StdEncoding.EncodeToString(saKey))
	assertContains(t, secrets, clusterSecret)
	assertContains(t, secrets, aesSecret)
	assertContains(t, secrets, secretboxSecret)
	assertContains(t, secrets, machineToken)
}

func assertContains(t *testing.T, slice []string, want string) {
	t.Helper()

	if slices.Contains(slice, want) {
		return
	}

	t.Errorf("slice does not contain %q; got %v", want, slice)
}
