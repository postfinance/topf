// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package sops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCollectEncryptedPaths(t *testing.T) {
	data := map[string]any{
		"plain":  "hello",
		"secret": "ENC[AES256_GCM,data:abc,type:str]",
		"nested": map[string]any{
			"deep":   "ENC[AES256_GCM,data:xyz,type:str]",
			"normal": "world",
		},
		"list": []any{
			"normal",
			"ENC[AES256_GCM,data:123,type:str]",
			map[string]any{
				"inner": "ENC[AES256_GCM,data:456,type:str]",
			},
		},
		"empty":   "",
		"numeric": 42,
		"nil_val": nil,
	}

	var paths [][]any
	collectEncryptedPaths(data, nil, &paths)

	got := make([]string, len(paths))
	for i, p := range paths {
		parts := make([]string, len(p))
		for j, seg := range p {
			parts[j] = fmt.Sprintf("%v", seg)
		}
		got[i] = strings.Join(parts, ".")
	}
	sort.Strings(got)

	want := []string{
		"list.1",
		"list.2.inner",
		"nested.deep",
		"secret",
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d paths, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("path[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestResolvePath(t *testing.T) {
	data := map[string]any{
		"a": map[string]any{
			"b": "deep-value",
		},
		"list": []any{
			"first",
			map[string]any{"key": "nested-in-list"},
		},
		"top": "top-value",
	}

	tests := []struct {
		name   string
		path   []any
		want   any
		wantOk bool
	}{
		{"nested map", []any{"a", "b"}, "deep-value", true},
		{"top level", []any{"top"}, "top-value", true},
		{"list index", []any{"list", 0}, "first", true},
		{"list nested map", []any{"list", 1, "key"}, "nested-in-list", true},
		{"missing key", []any{"nonexistent"}, nil, false},
		{"index on map", []any{"a", 0}, nil, false},
		{"key on list", []any{"list", "nope"}, nil, false},
		{"index out of bounds", []any{"list", 99}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolvePath(data, tt.path)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadFileWithSOPS(t *testing.T) {
	if _, err := exec.LookPath("sops"); err != nil {
		t.Skip("sops not in PATH")
	}
	if _, err := exec.LookPath("age-keygen"); err != nil {
		t.Skip("age-keygen not in PATH")
	}

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

		content, secrets, err := ReadFileWithSOPS(secretsPath)
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

		content, secrets, err := ReadFileWithSOPS(topfPath)
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
