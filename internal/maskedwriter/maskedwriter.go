// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package maskedwriter provides a writer that redacts sensitive strings from output.
package maskedwriter

import (
	"bytes"
	"io"
	"sync"
)

const redacted = "*** redacted ***"

// Writer wraps an io.Writer and replaces any occurrence of registered
// secrets with "*** redacted ***" before writing to the underlying writer.
type Writer struct {
	mu      sync.Mutex
	inner   io.Writer
	buf     []byte
	secrets [][]byte
}

// NewMaskedWriter returns a Writer that replaces any occurrence of the
// sensitive strings with "*** redacted ***" before writing to the
// underlying writer.
func NewMaskedWriter(writer io.Writer, sensitive []string) *Writer {
	w := &Writer{inner: writer}
	w.AddSecrets(sensitive)

	return w
}

// AddSecrets registers additional sensitive strings to be redacted.
func (w *Writer) AddSecrets(sensitive []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, s := range sensitive {
		if len(s) > 0 {
			w.secrets = append(w.secrets, []byte(s))
		}
	}
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, b := range p {
		w.buf = append(w.buf, b)

		n, full := w.bufferEndsWithSecretPrefix()
		if full {
			// flush everything before the secret
			if err := w.flush(len(w.buf) - n); err != nil {
				return len(p), err
			}

			// print the redacted message
			if _, err := w.inner.Write([]byte(redacted)); err != nil {
				return len(p), err
			}

			// clear out slice while keeping backing array the same
			w.buf = w.buf[:0]
		} else {
			// flush everything except the partial overlap
			if err := w.flush(len(w.buf) - n); err != nil {
				return len(p), err
			}
		}
	}

	return len(p), nil
}

// Flush writes any remaining buffered bytes to the underlying writer.
// Partial matches that never completed are written as-is.
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.flush(len(w.buf))
}

// flush writes the first n bytes of the buffer to the underlying writer
// and removes them from the buffer.
func (w *Writer) flush(n int) error {
	if n <= 0 {
		return nil
	}

	if _, err := w.inner.Write(w.buf[:n]); err != nil {
		return err
	}

	// cut off bytes from the beginning of slice
	// without allocation new backing array
	w.buf = append(w.buf[:0], w.buf[n:]...)

	return nil
}

// bufferEndsWithSecretPrefix returns the length of the longest secret prefix
// that matches at the end of the buffer. full is true when the match
// covers an entire secret.
func (w *Writer) bufferEndsWithSecretPrefix() (maxFound int, full bool) {
	for _, secret := range w.secrets {
		maxLen := min(len(secret), len(w.buf))

		for i := maxLen; i > maxFound; i-- {
			if bytes.HasSuffix(w.buf, secret[:i]) {
				maxFound = i
				full = i == len(secret)

				break
			}
		}
	}

	return
}
