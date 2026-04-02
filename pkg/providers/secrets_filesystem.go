// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/postfinance/topf/internal/sops"
)

// NewFilesystemSecretsProvider returns a SecretsProvider that reads and writes secrets.yaml files with optional SOPS support
func NewFilesystemSecretsProvider() SecretsProvider {
	path := "secrets.yaml"

	return &filesystemSecrets{
		path: path,
	}
}

type filesystemSecrets struct {
	path string
}

func (s *filesystemSecrets) Get(_ string) ([]byte, error) {
	content, _, err := sops.ReadFileWithSOPS(s.path)
	return content, err
}

func (s *filesystemSecrets) Put(_ string, bundle []byte) error {
	// Try to encrypt with SOPS
	// #nosec G204 required as long as we don't inline sops encryption
	cmd := exec.Command("sops", "encrypt", "--filename-override", s.path)

	cmd.Stdin = strings.NewReader(string(bundle))
	if output, err := cmd.Output(); err == nil {
		bundle = output
	}

	// Write to file with appropriate permissions
	if err := os.WriteFile(s.path, bundle, os.FileMode(0600)); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
