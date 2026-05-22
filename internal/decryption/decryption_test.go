// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package decryption

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	t.Run("non-existent file returns nil", func(t *testing.T) {
		content, secrets, err := ReadFile("/nonexistent/path/file.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != nil {
			t.Errorf("expected nil content, got %s", content)
		}
		if secrets != nil {
			t.Errorf("expected nil secrets, got %v", secrets)
		}
	})

	t.Run("plain file returns content without secrets", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "plain.yaml")

		if err := os.WriteFile(path, []byte("foo: bar\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		content, secrets, err := ReadFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(content) != "foo: bar\n" {
			t.Errorf("unexpected content: %q", string(content))
		}
		if len(secrets) != 0 {
			t.Errorf("expected no secrets, got %v", secrets)
		}
	})
}