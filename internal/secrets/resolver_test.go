// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package secrets

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// newTestResolver returns a resolver and the secrets it reported.
func newTestResolver() (Resolver, *[]string) {
	var found []string

	return NewResolver(func(s []string) { found = append(found, s...) }), &found
}

func TestReadFile(t *testing.T) {
	t.Run("missing file returns fs.ErrNotExist", func(t *testing.T) {
		_, err := NewResolver(nil).ReadFile(filepath.Join(t.TempDir(), "missing.yaml"))
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("expected fs.ErrNotExist, got %v", err)
		}
	})

	t.Run("plain file is returned unchanged without secrets", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "plain.yaml")
		writeFile(t, path, "foo: bar\n")

		r, found := newTestResolver()

		content, err := r.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "foo: bar\n" {
			t.Errorf("content = %q, want %q", content, "foo: bar\n")
		}

		if len(*found) != 0 {
			t.Errorf("expected no secrets, got %v", *found)
		}
	})

	t.Run("results are cached by absolute path", func(t *testing.T) {
		dir := t.TempDir()

		path := filepath.Join(dir, "plain.yaml")
		writeFile(t, path, "foo: bar\n")

		r := NewResolver(nil)
		if _, err := r.ReadFile(path); err != nil {
			t.Fatal(err)
		}

		writeFile(t, path, "foo: changed\n")
		t.Chdir(dir)

		content, err := r.ReadFile("plain.yaml")
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "foo: bar\n" {
			t.Errorf("expected cached content, got %q", content)
		}
	})

	t.Run("errors are not cached", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "late.yaml")
		r := NewResolver(nil)

		if _, err := r.ReadFile(path); err == nil {
			t.Fatal("expected error for missing file")
		}

		writeFile(t, path, "foo: bar\n")

		if _, err := r.ReadFile(path); err != nil {
			t.Fatalf("expected success after file was created, got %v", err)
		}
	})

	t.Run("symlinks share the cache entry of their target", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "plain.yaml")
		link := filepath.Join(dir, "link.yaml")
		writeFile(t, path, "foo: bar\n")

		if err := os.Symlink(path, link); err != nil {
			t.Fatal(err)
		}

		r := NewResolver(nil)
		if _, err := r.ReadFile(path); err != nil {
			t.Fatal(err)
		}

		writeFile(t, path, "foo: changed\n")

		content, err := r.ReadFile(link)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "foo: bar\n" {
			t.Errorf("expected cached content via symlink, got %q", content)
		}
	})
}

func TestEval(t *testing.T) {
	r, found := newTestResolver()
	in := []byte("foo: bar\nlist: [a, b]\n")

	out, err := r.Eval(in)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, in) {
		t.Errorf("Eval() = %q, want unchanged %q", out, in)
	}

	if len(*found) != 0 {
		t.Errorf("expected no secrets, got %v", *found)
	}
}
