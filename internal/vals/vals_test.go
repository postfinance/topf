// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package vals

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

func TestCollectValsPaths(t *testing.T) {
	data := map[string]any{
		"plain":     "hello",
		"vault_ref": "ref+vault://secret/data/foo#/bar",
		"nested": map[string]any{
			"deep":    "ref+sops://path/to/file#/key",
			"normal":  "world",
		},
		"list": []any{
			"normal",
			"ref+awsssm://my/param",
			map[string]any{
				"inner": "ref+vault://secret/data/baz#/qux",
			},
		},
		"empty":   "",
		"numeric": 42,
		"nil_val": nil,
	}

	var paths [][]any
	collectValsPaths(data, nil, &paths)

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
		"vault_ref",
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

func TestEvalContent(t *testing.T) {
	if _, err := exec.LookPath("vals"); err != nil {
		t.Skip("vals not in PATH")
	}

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
		if err := writefile(secretFile, "plaintext-value"); err != nil {
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
		if err := writefile(secretFile, "deep-secret"); err != nil {
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

func writefile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}