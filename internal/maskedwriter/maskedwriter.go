// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package maskedwriter provides a writer that redacts sensitive strings from output.
//
// Incoming bytes are buffered while they could still be part of a secret.
// As soon as a match is impossible, safe bytes are flushed to the underlying
// writer. When a complete secret is found, it is replaced with a redaction
// marker. Call Close after the last Write to emit any remaining buffered bytes.
package maskedwriter

import (
	"bytes"
	"io"
	"sync"
)

const redacted = "*** redacted ***"

// Writer wraps an io.Writer and replaces any occurrence of registered
// secrets with "*** redacted ***" before writing to the underlying writer.
//
// Bytes are held in a pending buffer while they could still be the start
// of a secret. This allows secrets that are split across multiple Write
// calls to be detected and redacted.
type Writer struct {
	mu      sync.Mutex
	inner   io.Writer
	secrets [][]byte
	pending []byte // bytes not yet written; might be part of a secret
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

// Write appends p to the internal buffer and drains as many bytes as
// possible to the underlying writer. Bytes that form a potential secret
// prefix remain buffered until more input resolves the ambiguity or
// Close is called.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.pending = append(w.pending, p...)

	if err := w.drainPending(false); err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close drains all buffered bytes to the underlying writer. Buffered
// content that forms a complete secret is redacted; everything else is
// emitted verbatim. The underlying writer is not closed.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.drainPending(true)
}

// drainPending emits bytes from the pending buffer to the inner writer.
//
// The loop applies three rules in order:
//
//  1. If the buffer is a proper (not full) prefix of any secret and we are not
//     flushing, stop — more input may complete the match.
//  2. If the longest registered secret matches at the start of the
//     buffer, replace it with the redaction marker and continue.
//  3. Otherwise the first byte cannot be part of a secret starting
//     here — emit it and retry from step 1.
func (w *Writer) drainPending(flush bool) error {
	for len(w.pending) > 0 {
		// Rule 1: pause when future input could still complete a match.
		if !flush && w.couldGrowIntoSecret(w.pending) {
			return nil
		}

		// Rule 2: redact the longest secret that starts at pending[0].
		if n := w.longestMatchAtStart(w.pending); n > 0 {
			if _, err := io.WriteString(w.inner, redacted); err != nil {
				return err
			}

			w.pending = w.pending[n:]

			continue
		}

		// Rule 3: first byte is safe — emit it.
		if _, err := w.inner.Write(w.pending[:1]); err != nil {
			return err
		}

		w.pending = w.pending[1:]
	}

	return nil
}

// couldGrowIntoSecret reports whether buf is a proper prefix of at least
// one registered secret, meaning additional input could complete a match.
func (w *Writer) couldGrowIntoSecret(buf []byte) bool {
	for _, s := range w.secrets {
		if len(s) > len(buf) && bytes.HasPrefix(s, buf) {
			return true
		}
	}

	return false
}

// longestMatchAtStart returns the length of the longest registered secret
// that matches at the very beginning of buf, or 0 if none matches.
func (w *Writer) longestMatchAtStart(buf []byte) int {
	best := 0

	for _, s := range w.secrets {
		if len(s) > best && bytes.HasPrefix(buf, s) {
			best = len(s)
		}
	}

	return best
}
