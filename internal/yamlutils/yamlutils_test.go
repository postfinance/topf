// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package yamlutils

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

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
			got, ok := ResolvePath(data, tt.path)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectPaths(t *testing.T) {
	data := map[string]any{
		"plain":  "hello",
		"secret": "ENC[AES256_GCM,data:abc,type=str]",
		"nested": map[string]any{
			"deep":   "ENC[AES256_GCM,data:xyz,type=str]",
			"normal": "world",
		},
		"list": []any{
			"normal",
			"ENC[AES256_GCM,data:123,type=str]",
			map[string]any{
				"inner": "ENC[AES256_GCM,data:456,type=str]",
			},
		},
		"empty":   "",
		"numeric": 42,
		"nil_val": nil,
	}

	var paths [][]any
	CollectPaths(data, nil, &paths, func(s string) bool {
		return strings.HasPrefix(s, "ENC[")
	})

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

func TestExtractSecrets(t *testing.T) {
	before := []byte("key: ENC[data]\nother: plain\n")
	after := []byte("key: secret-value\nother: plain\n")

	secrets := ExtractSecrets(before, after, func(s string) bool {
		return strings.HasPrefix(s, "ENC[")
	})

	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d: %v", len(secrets), secrets)
	}
	if secrets[0] != "secret-value" {
		t.Errorf("secret = %q, want %q", secrets[0], "secret-value")
	}
}
