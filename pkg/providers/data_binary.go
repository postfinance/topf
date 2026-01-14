package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// NewBinaryDataProvider returns a DataProvider that delegates getting data to a binary
func NewBinaryDataProvider(binaryPath string) DataProvider {
	return &binaryData{
		binaryPath: binaryPath,
	}
}

type binaryData struct {
	binaryPath string
}

func (d *binaryData) GetData(clusterName string) ([]byte, error) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//nolint:gosec // launching arbitrary binary is part of the design
	cmd := exec.CommandContext(ctx, d.binaryPath, clusterName)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		// Check if timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, errors.New("data provider command timed out after 30s")
		}

		return nil, fmt.Errorf("failed to execute data provider command: %w", err)
	}

	return output, nil
}
