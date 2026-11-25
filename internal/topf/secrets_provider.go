// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package topf

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

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
// Writes are SOPS-encrypted unless sops is not installed or not configured
// for the path, in which case they are stored in plaintext.
type filesystemSecrets struct {
	path     string
	resolver secrets.Resolver
}

func (s *filesystemSecrets) Get(_ string) ([]byte, error) {
	return s.resolver.ReadFile(s.path)
}

func (s *filesystemSecrets) Put(_ string, bundle []byte) error {
	content, err := s.encrypt(bundle)
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.path, content, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// encrypt encrypts the bundle with SOPS. It falls back to plaintext only if
// sops is not installed or not configured for the secrets file.
func (s *filesystemSecrets) encrypt(bundle []byte) ([]byte, error) {
	// #nosec G204 required as long as we don't inline sops encryption
	cmd := exec.Command("sops", "encrypt", "--filename-override", s.path)
	cmd.Stdin = bytes.NewReader(bundle)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}

	if errors.Is(err, exec.ErrNotFound) {
		slog.Warn("storing secrets.yaml unencrypted", "path", s.path)
		return bundle, nil
	}

	// These sops errors mean SOPS is not set up for the secrets file (no
	// .sops.yaml or no matching creation rule), so plaintext is intended.
	sopsErr := strings.TrimSpace(stderr.String())

	for _, msg := range []string{
		"config file not found, or has no creation rules",
		"no matching creation rules found",
	} {
		if strings.Contains(sopsErr, msg) {
			slog.Warn("sops isnot configure for secrets.yaml, storing unencrypted", "path", s.path, "sops", sopsErr)
			return bundle, nil
		}
	}

	return nil, fmt.Errorf("failed to encrypt %s with sops: %w: %s", s.path, err, sopsErr)
}
