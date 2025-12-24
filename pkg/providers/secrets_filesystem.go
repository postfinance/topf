package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
	// Check if file exists
	_, err := os.Stat(s.path)
	if os.IsNotExist(err) {
		// not an error, just no secret available
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Check if file is encrypted with SOPS
	// #nosec G204 required as long as we don't inline sops encryption
	statusCmd := exec.Command("sops", "filestatus", s.path)
	statusOutput, statusErr := statusCmd.Output()

	var status struct {
		Encrypted bool `json:"encrypted"`
	}

	isEncrypted := statusErr == nil &&
		json.Unmarshal(statusOutput, &status) == nil &&
		status.Encrypted

	if !isEncrypted {
		return os.ReadFile(s.path)
	}

	// File is encrypted: try to decrypt
	// #nosec G204 required as long as we don't inline sops encryption
	decryptCmd := exec.Command("sops", "decrypt", s.path)

	output, err := decryptCmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("sops decryption failed: %s", string(exitErr.Stderr))
		}

		return nil, fmt.Errorf("failed to run sops decrypt: %w", err)
	}

	return output, nil
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
