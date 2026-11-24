// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package userfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/postfinance/topf/internal/secrets"
	"github.com/postfinance/topf/internal/tmpl"
	"github.com/postfinance/topf/pkg/config"
)

func write(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestRead(t *testing.T) {
	data := &tmpl.Context{
		TopfConfig: &config.TopfConfig{ClusterName: "test"},
		Node:       tmpl.Node{Node: config.Node{Host: "n1"}},
	}
	r := secrets.NewResolver(nil)

	t.Run("plain file is not rendered", func(t *testing.T) {
		const content = "host: \"{{ .Node.Host }}\"\n"

		got, err := Read(write(t, "a.yaml", content), data, r)
		if err != nil {
			t.Fatal(err)
		}

		if string(got) != content {
			t.Errorf("Read() = %q, want %q", got, content)
		}
	})

	t.Run("template is rendered with data", func(t *testing.T) {
		got, err := Read(write(t, "a.yaml.tpl", "host: {{ .ClusterName }}-{{ .Node.Host }}\n"), data, r)
		if err != nil {
			t.Fatal(err)
		}

		if want := "host: test-n1\n"; string(got) != want {
			t.Errorf("Read() = %q, want %q", got, want)
		}
	})

	t.Run("encrypted template is rejected", func(t *testing.T) {
		_, err := Read(write(t, "a.yaml.tpl", "token: ENC[AES256_GCM,data:abc]\n"), data, r)
		if !errors.Is(err, tmpl.ErrEncryptedTemplate) {
			t.Fatalf("expected ErrEncryptedTemplate, got %v", err)
		}
	})

	t.Run("missing template returns fs.ErrNotExist", func(t *testing.T) {
		_, err := Read(filepath.Join(t.TempDir(), "missing.yaml.tpl"), data, r)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("expected fs.ErrNotExist, got %v", err)
		}
	})
}
