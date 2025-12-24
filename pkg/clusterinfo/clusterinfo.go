// Package clusterinfo contains the output type of the clusterinfo command
package clusterinfo

import "github.com/postfinance/topf/pkg/config"

// ClusterInfo contains the general config of the cluster along with certificate material of the components
type ClusterInfo struct {
	ClusterName       string        `yaml:"clusterName"`
	ClusterEndpoint   string        `yaml:"clusterEndpoint"`
	ClusterCA         string        `yaml:"clusterCA"`
	EtcdCA            string        `yaml:"etcdCA"`
	TalosCA           string        `yaml:"talosCA"`
	KubernetesVersion string        `yaml:"kubernetesVersion"`
	Nodes             []config.Node `yaml:"nodes"`
}
