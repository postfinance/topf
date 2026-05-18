// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package schematic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

	got, err := r.Resolve(context.Background(), "factory.talos.dev", "@schematic.yaml", nil)
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

	got, err := r.Resolve(context.Background(), "factory.talos.dev", "SOME_PLAIN_ID", nil)
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

	got, err := r.Resolve(context.Background(), "factory.talos.dev", "@schematic.yaml", nil)
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

	got, err := r.Resolve(context.Background(), "factory.talos.dev", "@schematic.yaml.tpl", tmplData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 64 {
		t.Errorf("expected 64-char schematic ID, got %q (len=%d)", got, len(got))
	}
}

func TestResolve_CachedResult(t *testing.T) {
	var requests atomic.Int32

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	}))
	defer srv.Close()

	dir := t.TempDir()
	r := NewResolver(dir, "test", WithClient(srv.Client()))

	content := "customization: {}"
	writeSchematicFile(t, dir, "schematic.yaml", content)

	got1, err := r.Resolve(context.Background(), testFactory(srv), "@schematic.yaml", nil)
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}

	got2, err := r.Resolve(context.Background(), testFactory(srv), "@schematic.yaml", nil)
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}

	if got1 != got2 {
		t.Errorf("expected same ID for same content (cache hit), got %s and %s", got1, got2)
	}

	if requests.Load() != 1 {
		t.Errorf("expected 1 server request (cache hit on second call), got %d", requests.Load())
	}
}

func TestResolve_FileNotFound(t *testing.T) {
	r := NewResolver(".", "test")

	_, err := r.Resolve(context.Background(), "factory.talos.dev", "@nonexistent.yaml", nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
