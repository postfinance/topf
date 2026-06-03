// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"testing"
)

func TestDns1123Label(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "main", "main"},
		{"dotted version", "v1.2.3", "v1.2.3"},
		{"slash in git ref", "feat/kube-bench-1-2-5-kubelet-ca", "feat-kube-bench-1-2-5-kubelet-ca"},
		{"underscore and dot kept", "feature/ABC_123.x", "feature-ABC_123.x"},
		{"leading and trailing separators trimmed", "/leading/slash/", "leading-slash"},
		{"only invalid chars", "///", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dns1123Label(tt.in); got != tt.want {
				t.Errorf("dns1123Label(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDns1123LabelTruncatesAndTrims(t *testing.T) {
	in := "x/" + strings.Repeat("a", 80)

	got := dns1123Label(in)
	if len(got) > 63 {
		t.Errorf("dns1123Label(%q) length = %d, want <= 63", in, len(got))
	}

	if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") {
		t.Errorf("dns1123Label(%q) = %q, must not start or end with %q", in, got, "-")
	}
}
