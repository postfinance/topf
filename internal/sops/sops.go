// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

// Package sops provides utilities for checking SOPS encryption status
// and decrypting SOPS-encrypted files.
package sops

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/postfinance/topf/internal/yamlutils"
)

// IsEncrypted checks whether a file is encrypted with SOPS by running
// `sops filestatus`. Returns false with no error if the sops binary is
// not available (graceful degradation). Returns an error if sops is
// available but produces unexpected output.
func IsEncrypted(path string) (bool, error) {
	// #nosec G204 required as long as we don't inline sops decryption
	statusCmd := exec.Command("sops", "filestatus", path)
	statusOutput, cmdErr := statusCmd.Output()

	// sops not installed or filestatus failed — treat as not encrypted
	//nolint:nilerr // intentional: graceful degradation when sops is unavailable
	if cmdErr != nil {
		return false, nil
	}

	var status struct {
		Encrypted bool `json:"encrypted"`
	}

	if err := json.Unmarshal(statusOutput, &status); err != nil {
		return false, fmt.Errorf("failed to parse sops filestatus output: %w", err)
	}

	return status.Encrypted, nil
}

// Decrypt decrypts a SOPS-encrypted file and returns the decrypted content
// along with the plaintext values of any SOPS-encrypted fields (identified
// by the "ENC[" prefix) for output redaction.
func Decrypt(path string) ([]byte, []string, error) {
	// Read the encrypted file to discover which fields are encrypted
	//nolint:gosec // files read through a variable in our control
	encryptedBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// #nosec G204 required as long as we don't inline sops decryption
	decryptCmd := exec.Command("sops", "decrypt", path)

	var stderr strings.Builder

	decryptCmd.Stderr = &stderr

	content, err := decryptCmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, nil, fmt.Errorf("sops decryption failed: %s: %w", strings.TrimSpace(stderr.String()), err)
		}

		return nil, nil, fmt.Errorf("sops decryption failed: %w", err)
	}

	secrets := yamlutils.ExtractSecrets(encryptedBytes, content, func(s string) bool {
		return strings.HasPrefix(s, "ENC[")
	})

	return content, secrets, nil
}
