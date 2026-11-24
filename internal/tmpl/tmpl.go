// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package tmpl renders Go templates for patches and schematics.
package tmpl

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// ErrEncryptedTemplate is returned when a template source is SOPS-encrypted.
var ErrEncryptedTemplate = errors.New("SOPS-encrypted templates are not supported; use data in topf.yaml or a ref+sops:// reference")

// Render executes src as a Go template with the sprig function library and
// the given context. Missing map keys and unknown fields are errors.
// name identifies the template in error messages, typically the file path.
func Render(name string, src []byte, data *Context) ([]byte, error) {
	if bytes.Contains(src, []byte("ENC[AES256_GCM,")) {
		return nil, fmt.Errorf("%s: %w", name, ErrEncryptedTemplate)
	}

	t, err := template.New(name).
		Funcs(sprig.TxtFuncMap()).
		Option("missingkey=error").
		Parse(string(src))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
