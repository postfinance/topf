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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/postfinance/topf/pkg/config"
)

func testFactory(srv *httptest.Server) string {
	return strings.TrimPrefix(srv.URL, "https://")
}

func newMockFactoryServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var requests atomic.Int32

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost && r.URL.Path == "/schematics" {
			json.NewEncoder(w).Encode(map[string]string{
				"id":        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"schematic": "customization: {}\n",
			})

			return
		}

		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/schematics/") {
			w.Header().Set("Content-Type", "application/yaml")
			w.Write([]byte("customization: {}\n"))

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	return srv, &requests
}

func TestResolve(t *testing.T) {
	t.Run("plainID", func(t *testing.T) {
		r := NewResolver(".", "test")
		got, err := r.Resolve(context.Background(), "", "abc123", nil)
		if err != nil {
			t.Fatal(err)
		}
		if got != "abc123" {
			t.Errorf("expected passthrough, got %s", got)
		}
	})

	t.Run("localID", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "s.yaml"), []byte("customization: {}\n"), 0o644)

		r := NewResolver(dir, "test")
		got, err := r.Resolve(context.Background(), "factory.talos.dev", "@s.yaml", nil)
		if err != nil {
			t.Fatal(err)
		}

		want := "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	})

	t.Run("templateFile", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "s.yaml.tpl"), []byte(`customization:
  systemExtensions:
    officialExtensions:
{{- range .Data.extensions }}
      - {{ . }}
{{- end }}
`), 0o644)

		r := NewResolver(dir, "test")
		got, err := r.Resolve(context.Background(), "", "@s.yaml.tpl", &config.PatchContext{
			Data: map[string]any{"extensions": []string{"siderolabs/qemu-guest-agent"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 64 {
			t.Errorf("expected 64-char ID, got %q (len=%d)", got, len(got))
		}
	})

	t.Run("defaultNoNetwork", func(t *testing.T) {
		srv, reqs := newMockFactoryServer(t)
		defer srv.Close()

		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "s.yaml"), []byte("customization: {}"), 0o644)

		r := NewResolver(dir, "test", WithHTTPClient(srv.Client()))
		if _, err := r.Resolve(context.Background(), testFactory(srv), "@s.yaml", nil); err != nil {
			t.Fatal(err)
		}
		if reqs.Load() != 0 {
			t.Errorf("expected 0 requests, got %d", reqs.Load())
		}
	})

	t.Run("submitToFactory", func(t *testing.T) {
		srv, reqs := newMockFactoryServer(t)
		defer srv.Close()

		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "s.yaml"), []byte("customization: {}"), 0o644)

		r := NewResolver(dir, "test", WithSubmitToFactory(true), WithHTTPClient(srv.Client()))
		got, err := r.Resolve(context.Background(), testFactory(srv), "@s.yaml", nil)
		if err != nil {
			t.Fatal(err)
		}

		want := "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"
		if got != want {
			t.Errorf("got %s, want %s", got, want)
		}

		if reqs.Load() != 1 {
			t.Errorf("expected 1 request, got %d", reqs.Load())
		}
	})

	t.Run("submitToFactoryDedup", func(t *testing.T) {
		srv, reqs := newMockFactoryServer(t)
		defer srv.Close()

		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "s.yaml"), []byte("customization: {}"), 0o644)

		r := NewResolver(dir, "test", WithSubmitToFactory(true), WithHTTPClient(srv.Client()))

		var wg sync.WaitGroup

		results := make([]string, 5)
		errors := make([]error, 5)

		for i := range 5 {
			wg.Add(1)

			go func(idx int) {
				defer wg.Done()
				results[idx], errors[idx] = r.Resolve(context.Background(), testFactory(srv), "@s.yaml", nil)
			}(i)
		}

		wg.Wait()

		for i, err := range errors {
			if err != nil {
				t.Fatalf("goroutine %d: %v", i, err)
			}
		}

		want := "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"
		for i, got := range results {
			if got != want {
				t.Errorf("goroutine %d: got %s, want %s", i, got, want)
			}
		}

		if reqs.Load() != 1 {
			t.Errorf("expected 1 request (concurrent calls deduplicated), got %d", reqs.Load())
		}
	})

	t.Run("fileNotFound", func(t *testing.T) {
		r := NewResolver(".", "test")
		if _, err := r.Resolve(context.Background(), "", "@nope.yaml", nil); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalidSchematic", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "s.yaml"), []byte("bogus: true\n"), 0o644)

		r := NewResolver(dir, "test")
		if _, err := r.Resolve(context.Background(), "", "@s.yaml", nil); err == nil {
			t.Fatal("expected error")
		}
	})
}
