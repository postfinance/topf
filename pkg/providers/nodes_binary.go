package providers

import (
	"fmt"
	"os"
	"os/exec"
)

// NewBinaryNodesProvider returns a NodesProvider that delegates getting nodes to a binary
func NewBinaryNodesProvider(binaryPath string) NodesProvider {
	return &binaryNodes{
		binaryPath: binaryPath,
	}
}

type binaryNodes struct {
	binaryPath string
}

func (n *binaryNodes) GetNodes(clusterName string) ([]byte, error) {
	//nolint:gosec // launching arbitrary binary is part of the design
	cmd := exec.Command(n.binaryPath, "nodes", clusterName)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute nodes provider command: %w", err)
	}

	return output, nil
}
