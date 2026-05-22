// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package decryption provides a unified interface for reading encrypted files,
// applying SOPS decryption and vals evaluation in sequence.
package decryption

import (
	"github.com/postfinance/topf/internal/sops"
	"github.com/postfinance/topf/internal/vals"
)

// ReadFile reads a file, automatically decrypts it with SOPS if encrypted,
// then evaluates any vals references. It returns the final content and
// a combined list of plaintext secret values discovered during both
// SOPS decryption and vals evaluation (for output redaction).
// Returns nil content if the file doesn't exist (not an error).
func ReadFile(path string) (content []byte, secrets []string, err error) {
	content, sopsSecrets, err := sops.ReadFileWithSOPS(path)
	if err != nil {
		return nil, nil, err
	}

	if content == nil {
		return nil, nil, nil
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
