// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package maskedwriter

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

func TestMaskedWriter(t *testing.T) {
	tests := []struct {
		name            string
		secrets         []string
		writes          []string
		wantBeforeClose string // if set, asserted before Close() is called
		expected        string
	}{
		{
			name:     "no secrets",
			secrets:  nil,
			writes:   []string{"hello world"},
			expected: "hello world",
		},
		{
			name:     "no match",
			secrets:  []string{"xyz"},
			writes:   []string{"hello world"},
			expected: "hello world",
		},
		{
			name:     "exact match single write",
			secrets:  []string{"secret"},
			writes:   []string{"my secret value"},
			expected: "my *** redacted *** value",
		},
		{
			name:     "match split across writes",
			secrets:  []string{"secret"},
			writes:   []string{"my sec", "ret value"},
			expected: "my *** redacted *** value",
		},
		{
			name:     "false alarm then no match",
			secrets:  []string{"ab"},
			writes:   []string{"a", "a", "c"},
			expected: "aac",
		},
		{
			name:     "false alarm then match",
			secrets:  []string{"ab"},
			writes:   []string{"a", "a", "b"},
			expected: "a*** redacted ***",
		},
		{
			name:     "multiple secrets",
			secrets:  []string{"foo", "bar"},
			writes:   []string{"foo and bar"},
			expected: "*** redacted *** and *** redacted ***",
		},
		{
			name:     "overlapping no double match",
			secrets:  []string{"fufu"},
			writes:   []string{"fufufu"},
			expected: "*** redacted ***fu",
		},
		{
			name:     "byte by byte",
			secrets:  []string{"abc"},
			writes:   []string{"a", "b", "c"},
			expected: "*** redacted ***",
		},
		{
			name:            "partial match at end needs close",
			secrets:         []string{"abc"},
			writes:          []string{"xab"},
			wantBeforeClose: "x", // "ab" is still buffered
			expected:        "xab",
		},
		{
			name:     "empty secret ignored",
			secrets:  []string{"", "x"},
			writes:   []string{"x"},
			expected: "*** redacted ***",
		},
		{
			name:     "secret at start",
			secrets:  []string{"hello"},
			writes:   []string{"hello world"},
			expected: "*** redacted *** world",
		},
		{
			name:     "secret at end",
			secrets:  []string{"world"},
			writes:   []string{"hello world"},
			expected: "hello *** redacted ***",
		},
		{
			name:     "repeated secret",
			secrets:  []string{"ab"},
			writes:   []string{"ababab"},
			expected: "*** redacted ****** redacted ****** redacted ***",
		},
		{
			name:     "short match with longer partial keeps prefix",
			secrets:  []string{"d", "abce"},
			writes:   []string{"abcd"},
			expected: "abc*** redacted ***",
		},
		{
			name:     "short match with longer partial split writes",
			secrets:  []string{"cd", "abce"},
			writes:   []string{"ab", "cd"},
			expected: "ab*** redacted ***",
		},
		{
			name:            "partial match at end of all writes is lost without Close",
			secrets:         []string{"abc"},
			writes:          []string{"foo ab"},
			wantBeforeClose: "foo ", // "ab" is still buffered
			expected:        "foo ab",
		},
		{
			name:     "trailing newline flushes partial match without explicit Close",
			secrets:  []string{"abc"},
			writes:   []string{"foo ab\n"},
			expected: "foo ab\n",
		},
		{
			// longer secret doesn't complete — backtrack and redact
			// the shorter secret that was hidden in the buffer
			name:     "longer partial fails shorter secret redacted",
			secrets:  []string{"abcd", "abc"},
			writes:   []string{"abce"},
			expected: "*** redacted ***e",
		},
		{
			name:     "longer secret partial no complete",
			secrets:  []string{"aab"},
			writes:   []string{"aac"},
			expected: "aac",
		},
		{
			// shorter secret is a prefix of a longer one; the longer
			// one completes so the whole thing is redacted (no tail leak)
			name:     "overlapping prefix longer completes",
			secrets:  []string{"a", "aSECRET"},
			writes:   []string{"aSECRET"},
			expected: "*** redacted ***",
		},
		{
			name:     "overlapping prefix longer completes split",
			secrets:  []string{"abc", "abcd"},
			writes:   []string{"ab", "cd"},
			expected: "*** redacted ***",
		},
		{
			// longer secret doesn't complete; backtrack redacts the
			// shorter secret, remainder passes through
			name:     "overlapping prefix longer fails backtrack",
			secrets:  []string{"a", "aSECRET"},
			writes:   []string{"aXYZ"},
			expected: "*** redacted ***XYZ",
		},
		{
			// Close must redact a buffered byte that is itself a secret,
			// not leak it as a raw partial match
			name:            "close redacts held secret",
			secrets:         []string{"a", "abc"},
			writes:          []string{"xa"},
			wantBeforeClose: "x",
			expected:        "x*** redacted ***",
		},
		{
			name:            "longest match wins",
			secrets:         []string{"aaaaaaaaaaa", "a"},
			writes:          []string{"aaaaaaaaaaaa"},
			wantBeforeClose: "*** redacted ***",
			expected:        "*** redacted ****** redacted ***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewMaskedWriter(&buf, tt.secrets)

			for _, s := range tt.writes {
				if _, err := w.Write([]byte(s)); err != nil {
					t.Fatalf("Write() error: %v", err)
				}
			}

			if tt.wantBeforeClose != "" {
				if got := buf.String(); got != tt.wantBeforeClose {
					t.Errorf("before Close: got %q, want %q", got, tt.wantBeforeClose)
				}
			}

			if err := w.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}

			if got := buf.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestMaskedWriterConcurrency verifies that concurrent AddSecrets and Write
// calls do not race. Run with -race to detect any missing synchronization.
func TestMaskedWriterConcurrency(t *testing.T) {
	var buf bytes.Buffer
	w := NewMaskedWriter(&buf, nil)

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Go(func() {
			w.AddSecrets([]string{fmt.Sprintf("secret%d", i)})
		})
	}

	for i := range 10 {
		wg.Go(func() {
			fmt.Fprintf(w, "output line %d", i)
		})
	}

	wg.Wait()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}
