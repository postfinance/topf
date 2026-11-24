// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/postfinance/topf/internal/secrets"
)

// NewFilesystemSecretsProvider returns a SecretsProvider that reads and writes
// secrets.yaml at the given path with optional SOPS support.
func NewFilesystemSecretsProvider(secretsPath string, resolver secrets.Resolver) SecretsProvider {
	return &filesystemSecrets{
		path:     secretsPath,
		resolver: resolver,
	}
}

type filesystemSecrets struct {
	path     string
	resolver secrets.Resolver
}

func (s *filesystemSecrets) Get(_ string) ([]byte, error) {
	return s.resolver.ReadFile(s.path)
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
