// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package decryption provides a unified interface for reading files with
// automatic SOPS decryption and vals reference evaluation.
package decryption

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/postfinance/topf/internal/sops"
	"github.com/postfinance/topf/internal/vals"
)

// ReadFile reads a file, automatically decrypts it with SOPS if encrypted,
// then evaluates any vals references. It returns the final content and
// a combined list of plaintext secret values discovered during both
// SOPS decryption and vals evaluation (for output redaction).
// Returns an error wrapping fs.ErrNotExist if the file doesn't exist.
func ReadFile(path string) ([]byte, []string, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("read file %q: %w", path, fs.ErrNotExist)
		}

		return nil, nil, fmt.Errorf("read file %q: %w", path, err)
	}

	var (
		content     []byte
		sopsSecrets []string
	)

	isEncrypted, err := sops.IsEncrypted(path)
	if err != nil {
		return nil, nil, err
	}

	if isEncrypted {
		content, sopsSecrets, err = sops.Decrypt(path)
		if err != nil {
			return nil, nil, err
		}
	} else {
		//nolint:gosec // files read through a variable in our control
		content, err = os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
	}

	valsContent, valsSecrets, err := vals.EvalContent(content)
	if err != nil {
		return nil, nil, err
	}

	allSecrets := make([]string, 0, len(sopsSecrets)+len(valsSecrets))
	allSecrets = append(allSecrets, sopsSecrets...)
	allSecrets = append(allSecrets, valsSecrets...)

	return valsContent, allSecrets, nil
}
