// Copyright 2026 PostFinance AG
// SPDX-License-Identifier: MIT

package providers

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/postfinance/topf/internal/decryption"
)

// NewFilesystemSecretsProvider returns a SecretsProvider that reads and writes
// secrets.yaml at the given path with optional SOPS support.
func NewFilesystemSecretsProvider(secretsPath string, cache *decryption.Cache) SecretsProvider {
	return &filesystemSecrets{
		path:  secretsPath,
		cache: cache,
	}
}

type filesystemSecrets struct {
	path  string
	cache *decryption.Cache
}

func (s *filesystemSecrets) Get(_ string) ([]byte, error) {
	content, _, err := s.cache.ReadFileWithSecrets(s.path)
	return content, err
}

func (s *filesystemSecrets) Put(_ string, bundle []byte) error {
	content, err := s.encrypt(bundle)
	if err != nil {
		return err
	}

	return writeFileAtomic(s.path, content, 0o600)
}

// encrypt encrypts the bundle with SOPS. It falls back to plaintext only if
// sops is not installed or not configured for the secrets file. Any other
// failure (e.g. an unavailable key) is returned as an error, so that secrets
// are never silently written unencrypted when encryption was intended.
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
		slog.Warn("sops not found, storing secrets unencrypted", "path", s.path)
		return bundle, nil
	}

	sopsErr := strings.TrimSpace(stderr.String())

	// These sops errors mean SOPS is not set up for the secrets file (no
	// .sops.yaml or no matching creation rule). In that case the bundle is
	// intentionally stored in plaintext.
	for _, msg := range []string{
		"config file not found, or has no creation rules",
		"no matching creation rules found",
	} {
		if strings.Contains(sopsErr, msg) {
			slog.Warn("sops is not configured for secrets file, storing secrets unencrypted", "path", s.path, "sops", sopsErr)
			return bundle, nil
		}
	}

	return nil, fmt.Errorf("failed to encrypt %s with sops: %w: %s", s.path, err, sopsErr)
}

// writeFileAtomic writes data to a temporary file in the same directory and
// renames it into place, so a failed write never leaves a truncated file.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tmpName := tmp.Name()

	defer func() {
		// no-op after a successful rename
		_ = os.Remove(tmpName)
	}()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
