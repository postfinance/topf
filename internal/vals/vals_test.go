// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package vals

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasValsRefs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "no refs",
			content: "foo: bar\nbaz: qux\n",
			want:    false,
		},
		{
			name:    "has vault ref",
			content: "foo: ref+vault://secret/data/foo#/bar\n",
			want:    true,
		},
		{
			name:    "has sops ref",
			content: "foo: ref+sops://path/to/file#/key\n",
			want:    true,
		},
		{
			name:    "has awsssm ref",
			content: "foo: ref+awsssm://my/param\n",
			want:    true,
		},
		{
			name:    "ref in nested map",
			content: "nested:\n  deep: ref+vault://secret/data/foo#/bar\n",
			want:    true,
		},
		{
			name:    "ref in list",
			content: "items:\n  - ref+vault://secret/data/foo#/bar\n",
			want:    true,
		},
		{
			name:    "plain string with ref not at start",
			content: "foo: something ref+vault://secret/data/foo#/bar more\n",
			want:    false,
		},
		{
			name:    "string starting with ref but no provider",
			content: "foo: ref+noprovider\n",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasValsRefs([]byte(tt.content))
			if got != tt.want {
				t.Errorf("hasValsRefs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalContent(t *testing.T) {
	t.Run("no vals refs returns content unchanged", func(t *testing.T) {
		input := []byte("foo: bar\nbaz: 42\n")
		gotContent, secrets, err := EvalContent(input)
		if err != nil {
			t.Fatal(err)
		}
		if string(gotContent) != string(input) {
			t.Errorf("content should be unchanged when no refs present")
		}
		if len(secrets) != 0 {
			t.Errorf("expected no secrets, got %v", secrets)
		}
	})

	t.Run("echo provider resolves refs", func(t *testing.T) {
		dir := t.TempDir()

		secretFile := filepath.Join(dir, "mysecret")
		if err := os.WriteFile(secretFile, []byte("plaintext-value"), 0o644); err != nil {
			t.Fatal(err)
		}

		input := []byte(fmt.Sprintf("foo: ref+file://%s\nbar: plain-value\n", secretFile))
		content, secrets, err := EvalContent(input)
		if err != nil {
			t.Fatal(err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "plaintext-value") {
			t.Errorf("vals-evaluated content missing resolved secret, got: %s", contentStr)
		}
		if !strings.Contains(contentStr, "plain-value") {
			t.Errorf("vals-evaluated content missing plain value, got: %s", contentStr)
		}

		if len(secrets) != 1 {
			t.Fatalf("expected 1 secret, got %d: %v", len(secrets), secrets)
		}
		if secrets[0] != "plaintext-value" {
			t.Errorf("secret = %q, want %q", secrets[0], "plaintext-value")
		}
	})

	t.Run("nested vals refs", func(t *testing.T) {
		dir := t.TempDir()

		secretFile := filepath.Join(dir, "nestedsecret")
		if err := os.WriteFile(secretFile, []byte("deep-secret"), 0o644); err != nil {
			t.Fatal(err)
		}

		input := []byte(fmt.Sprintf("level1:\n  level2: ref+file://%s\n", secretFile))
		_, secrets, err := EvalContent(input)
		if err != nil {
			t.Fatal(err)
		}

		if len(secrets) != 1 {
			t.Fatalf("expected 1 secret, got %d: %v", len(secrets), secrets)
		}
		if secrets[0] != "deep-secret" {
			t.Errorf("secret = %q, want %q", secrets[0], "deep-secret")
		}
	})
}
