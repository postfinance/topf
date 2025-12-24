// Package clusterinfo contains the logic for the clusterinfo command
package clusterinfo

import (
	"encoding/base64"

	"github.com/postfinance/topf/internal/topf"
	"github.com/postfinance/topf/pkg/clusterinfo"
)

// Get returns the full non-sensitive information about the talos cluster configuration
func Get(t topf.Topf) (*clusterinfo.ClusterInfo, error) {
	c := t.Config()

	secretsBundle, err := t.Secrets()
	if err != nil {
		return nil, err
	}

	clusterInfo := clusterinfo.ClusterInfo{
		ClusterName:       c.ClusterName,
		ClusterEndpoint:   c.ClusterEndpoint.String(),
		ClusterCA:         base64.StdEncoding.EncodeToString(secretsBundle.Certs.K8s.Crt),
		EtcdCA:            base64.StdEncoding.EncodeToString(secretsBundle.Certs.Etcd.Crt),
		TalosCA:           base64.StdEncoding.EncodeToString(secretsBundle.Certs.OS.Crt),
		KubernetesVersion: c.KubernetesVersion,
		Nodes:             t.Config().Nodes,
	}

	return &clusterInfo, nil
}
