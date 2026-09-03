---
apiVersion: v1alpha1
kind: KubeletConfig
config:
    serverTLSBootstrap: true
---
apiVersion: v1alpha1
kind: KubeNodeConfig
labels:
    topology.kubernetes.io/region: "{{ .Data.region }}"