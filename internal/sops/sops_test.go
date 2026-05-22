// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package sops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsEncrypted(t *testing.T) {
	out, err := exec.Command("age-keygen").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	var publicKey, secretKey string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "# public key: ") {
			publicKey = strings.TrimPrefix(line, "# public key: ")
		}
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			secretKey = strings.TrimSpace(line)
		}
	}
	t.Setenv("SOPS_AGE_KEY", secretKey)

	t.Run("plaintext file returns false", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "plain.yaml")
		if err := os.WriteFile(path, []byte("foo: bar\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		encrypted, err := IsEncrypted(path)
		if err != nil {
			t.Fatal(err)
		}
		if encrypted {
			t.Error("plaintext file should not be detected as encrypted")
		}
	})

	t.Run("encrypted file returns true", func(t *testing.T) {
		dir := t.TempDir()

		sopsConfig := fmt.Sprintf("creation_rules:\n  - age: %s\n", publicKey)
		if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(sopsConfig), 0o644); err != nil {
			t.Fatal(err)
		}

		content := "secret: myvalue\n"
		path := filepath.Join(dir, "secrets.yaml")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(dir, ".sops.yaml")
		cmd := exec.Command("sops", "--config", configPath, "encrypt", "--in-place", path)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sops encrypt: %s: %s", err, out)
		}

		encrypted, err := IsEncrypted(path)
		if err != nil {
			t.Fatal(err)
		}
		if !encrypted {
			t.Error("encrypted file should be detected as encrypted")
		}
	})
}

func TestDecrypt(t *testing.T) {
	out, err := exec.Command("age-keygen").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	var publicKey, secretKey string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "# public key: ") {
			publicKey = strings.TrimPrefix(line, "# public key: ")
		}
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			secretKey = strings.TrimSpace(line)
		}
	}
	t.Setenv("SOPS_AGE_KEY", secretKey)

	t.Run("secrets.yaml", func(t *testing.T) {
		dir := t.TempDir()

		sopsConfig := fmt.Sprintf("creation_rules:\n  - age: %s\n", publicKey)
		if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(sopsConfig), 0o644); err != nil {
			t.Fatal(err)
		}

		secretsYAML := `cluster:
    id: sample-cluster-id
    secret: sample-cluster-secret
secrets:
    bootstraptoken: sample-bootstrap-token
    secretboxencryptionsecret: sample-secretbox-key
trustdinfo:
    token: sample-trustd-token
certs:
    etcd:
        crt: fake-etcd-cert
        key: fake-etcd-key
    k8s:
        crt: fake-k8s-cert
        key: fake-k8s-key
    k8saggregator:
        crt: fake-k8saggregator-cert
        key: fake-k8saggregator-key
    k8sserviceaccount:
        key: fake-k8s-serviceaccount-key
    os:
        crt: fake-os-cert
        key: fake-os-key
`
		secretsPath := filepath.Join(dir, "secrets.yaml")
		if err := os.WriteFile(secretsPath, []byte(secretsYAML), 0o644); err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(dir, ".sops.yaml")
		cmd := exec.Command("sops", "--config", configPath, "encrypt", "--in-place", secretsPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sops encrypt: %s: %s", err, out)
		}

		content, secrets, err := Decrypt(secretsPath)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(content), "sample-cluster-id") {
			t.Error("decrypted content missing sample-cluster-id")
		}

		wantSecrets := []string{
			"sample-cluster-id",
			"sample-cluster-secret",
			"sample-bootstrap-token",
			"sample-secretbox-key",
			"sample-trustd-token",
			"fake-etcd-cert",
			"fake-etcd-key",
			"fake-k8s-cert",
			"fake-k8s-key",
			"fake-k8saggregator-cert",
			"fake-k8saggregator-key",
			"fake-k8s-serviceaccount-key",
			"fake-os-cert",
			"fake-os-key",
		}

		secretSet := make(map[string]bool)
		for _, s := range secrets {
			secretSet[s] = true
		}
		for _, want := range wantSecrets {
			if !secretSet[want] {
				t.Errorf("missing secret: %q", want)
			}
		}
	})

	t.Run("topf.yaml partial encryption", func(t *testing.T) {
		dir := t.TempDir()

		sopsConfig := fmt.Sprintf("creation_rules:\n  - path_regex: topf\\.yaml$\n    encrypted_comment_regex: sops:enc\n    mac_only_encrypted: true\n    age: %s\n", publicKey)
		if err := os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(sopsConfig), 0o644); err != nil {
			t.Fatal(err)
		}

		topfYAML := `kubernetesVersion: "1.34.1"
clusterEndpoint: https://192.168.1.100:6443
clusterName: mycluster
nodes:
    - host: node1
      ip: 172.20.10.2
      role: control-plane
      data:
          foo: bar
          # sops:enc
          somesecret: supersecretvalue
`
		topfPath := filepath.Join(dir, "topf.yaml")
		if err := os.WriteFile(topfPath, []byte(topfYAML), 0o644); err != nil {
			t.Fatal(err)
		}

		configPath := filepath.Join(dir, ".sops.yaml")
		cmd := exec.Command("sops", "--config", configPath, "encrypt", "--in-place", topfPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("sops encrypt: %s: %s", err, out)
		}

		content, secrets, err := Decrypt(topfPath)
		if err != nil {
			t.Fatal(err)
		}

		contentStr := string(content)
		for _, want := range []string{"1.34.1", "192.168.1.100", "mycluster", "node1", "172.20.10.2", "bar"} {
			if !strings.Contains(contentStr, want) {
				t.Errorf("decrypted content missing %q", want)
			}
		}

		if len(secrets) != 1 {
			t.Fatalf("expected 1 secret, got %d: %v", len(secrets), secrets)
		}
		if secrets[0] != "supersecretvalue" {
			t.Errorf("secret = %q, want %q", secrets[0], "supersecretvalue")
		}
	})
}
