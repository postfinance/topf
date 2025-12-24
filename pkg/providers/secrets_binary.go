package providers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// NewBinarySecretsProvider returns a SecretsProvider that delegates reading and writing of secrets.yaml to a binary.
func NewBinarySecretsProvider(binaryPath string) SecretsProvider {
	return &binarySecrets{
		binaryPath: binaryPath,
	}
}

type binarySecrets struct {
	binaryPath string
}

func (s *binarySecrets) Get(clusterName string) ([]byte, error) {
	//nolint:gosec // launching arbitrary binary is part of the design
	cmd := exec.Command(s.binaryPath, "secrets", "get", clusterName)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets from binary: %w", err)
	}

	return output, nil
}

func (s *binarySecrets) Put(clusterName string, bundle []byte) error {
	//nolint:gosec // launching arbitrary binary is part of the design
	cmd := exec.Command(s.binaryPath, "secrets", "put", clusterName)
	cmd.Stdin = bytes.NewReader(bundle)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to put secrets via binary: %w", err)
	}

	return nil
}
