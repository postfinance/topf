// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package userfile reads user-provided files such as patches and schematics,
// rendering templates and resolving secrets.
package userfile

import (
	"fmt"
	"os"
	"strings"

	"github.com/postfinance/topf/internal/secrets"
	"github.com/postfinance/topf/internal/tmpl"
)

// Read returns the content of path with secrets resolved.
// Files ending in .tpl are read as plain text, rendered with data, then passed
// through vals. Other files are SOPS-decrypted and passed through vals.
func Read(path string, data *tmpl.Context, resolver secrets.Resolver) ([]byte, error) {
	if !strings.HasSuffix(path, ".tpl") {
		return resolver.ReadFile(path)
	}

	src, err := os.ReadFile(path) //nolint:gosec // user-provided files are read by design
	if err != nil {
		return nil, err
	}

	rendered, err := tmpl.Render(path, src, data)
	if err != nil {
		return nil, err
	}

	content, err := resolver.Eval(rendered)
	if err != nil {
		return nil, fmt.Errorf("resolve vals references in %s: %w", path, err)
	}

	return content, nil
}
