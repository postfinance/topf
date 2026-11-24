// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/postfinance/topf/internal/secrets"
	"github.com/postfinance/topf/pkg/config"
	"github.com/postfinance/topf/pkg/providers"
)

// newSecretsProvider returns the configured secrets provider binary, or the
// filesystem provider for cfg.SecretsPath.
func newSecretsProvider(cfg *config.TopfConfig, resolver secrets.Resolver) providers.SecretsProvider {
	if cfg.SecretsProvider != "" {
		return providers.NewBinarySecretsProvider(cfg.SecretsProvider)
	}

	return &filesystemSecrets{path: cfg.SecretsPath, resolver: resolver}
}

// filesystemSecrets reads and writes a local secrets.yaml.
// Writes are SOPS-encrypted if sops can encrypt the path, plaintext otherwise.
type filesystemSecrets struct {
	path     string
	resolver secrets.Resolver
}

func (s *filesystemSecrets) Get(_ string) ([]byte, error) {
	return s.resolver.ReadFile(s.path)
}

func (s *filesystemSecrets) Put(_ string, bundle []byte) error {
	// #nosec G204 required as long as we don't inline sops encryption
	cmd := exec.Command("sops", "encrypt", "--filename-override", s.path)
	cmd.Stdin = bytes.NewReader(bundle)

	if output, err := cmd.Output(); err == nil {
		bundle = output
	}

	if err := os.WriteFile(s.path, bundle, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
