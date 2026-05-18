// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package schematic

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/postfinance/topf/pkg/config"
)

func testFactory(srv *httptest.Server) string {
	return strings.TrimPrefix(srv.URL, "https://")
}

func writeSchematicFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolve_DefaultSchematicOnFactory(t *testing.T) {
	tmpDir := t.TempDir()
	schematicFile := filepath.Join(tmpDir, "schematic.yaml")

	if err := os.WriteFile(schematicFile, []byte("customization: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(tmpDir, "test")

	got, err := r.Resolve(context.Background(), "@schematic.yaml", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaultSchematic := "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"
	if got != defaultSchematic {
		t.Errorf("expected default schematic %s, got %s", defaultSchematic, got)
	}
}

func TestResolve_PlainID(t *testing.T) {
	r := NewResolver(".", "test")

	got, err := r.Resolve(context.Background(), "SOME_PLAIN_ID", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "SOME_PLAIN_ID" {
		t.Errorf("expected plain ID to pass through, got %s", got)
	}
}

func TestResolve_AtPrefixedFile(t *testing.T) {
	tmpDir := t.TempDir()
	schematicFile := filepath.Join(tmpDir, "schematic.yaml")

	content := []byte("customization:\n  systemExtensions:\n    officialExtensions:\n      - siderolabs/qemu-guest-agent\n")

	if err := os.WriteFile(schematicFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewResolver(tmpDir, "test")

	got, err := r.Resolve(context.Background(), "@schematic.yaml", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 64 {
		t.Errorf("expected 64-char schematic ID, got %q (len=%d)", got, len(got))
	}
}

func TestResolve_TemplateFile(t *testing.T) {
	tmpDir := t.TempDir()
	schematicFile := filepath.Join(tmpDir, "schematic.yaml.tpl")

	content := []byte(`customization:
  systemExtensions:
    officialExtensions:
{{- range .Data.extensions }}
      - {{ . }}
{{- end }}
`)

	if err := os.WriteFile(schematicFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	tmplData := &config.PatchContext{
		Data: map[string]any{"extensions": []string{"siderolabs/qemu-guest-agent"}},
	}

	r := NewResolver(tmpDir, "test")

	got, err := r.Resolve(context.Background(), "@schematic.yaml.tpl", tmplData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 64 {
		t.Errorf("expected 64-char schematic ID, got %q (len=%d)", got, len(got))
	}
}

func TestResolve_FileNotFound(t *testing.T) {
	r := NewResolver(".", "test")

	_, err := r.Resolve(context.Background(), "@nonexistent.yaml", nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
