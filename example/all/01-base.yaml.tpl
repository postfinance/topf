machine:
  kubelet:
    extraConfig:
      serverTLSBootstrap: true
  nodeLabels:
    topology.kubernetes.io/region: "{{ .Data.region }}"
