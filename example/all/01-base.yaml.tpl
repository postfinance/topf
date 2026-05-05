machine:
  kubelet:
    registerWithFQDN: true
    extraConfig:
      serverTLSBootstrap: true
  nodeLabels:
    topology.kubernetes.io/region: "{{ .Data.region }}"
cluster:
  discovery:
    enabled: false
