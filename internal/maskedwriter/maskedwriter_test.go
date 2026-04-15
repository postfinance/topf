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
		name     string
		secrets  []string
		writes   []string
		expected string
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
			name:     "partial match at end needs flush",
			secrets:  []string{"abc"},
			writes:   []string{"xab"},
			expected: "x", // "ab" is buffered, flushed by Flush()
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

			// Flush remaining buffer.
			if err := w.Flush(); err != nil {
				t.Fatalf("Flush() error: %v", err)
			}

			got := buf.String()

			// For the "partial match at end" test, the flushed
			// partial is appended after.
			if tt.name == "partial match at end needs flush" {
				if got != "xab" {
					t.Errorf("got %q, want %q", got, "xab")
				}

				return
			}

			if got != tt.expected {
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

	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
}
