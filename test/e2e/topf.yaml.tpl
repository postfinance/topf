# topf config for the e2e test. Node IPs are substituted by the workflow
# from the talosctl-cluster-action outputs. BOOT_VERSION is the Talos version
# the maintenance VMs booted from (must match test/e2e/cluster.yaml) and is
# what `topf apply` installs. The workflow then bumps talosVersion to the
# go.mod version of github.com/siderolabs/talos before running
# `topf upgrade`, so the upgrade target tracks dependency bumps automatically.
# Kubernetes version is omitted: topf falls back to the default bundled with
# the Talos version from go.mod.
clusterName: topf-e2e
clusterEndpoint: https://${CP1_IP}:6443
talosVersion: ${BOOT_VERSION}
patchesDir: patches

nodes:
  - host: controlplane-1
    ip: ${CP1_IP}
    role: control-plane
  - host: controlplane-2
    ip: ${CP2_IP}
    role: control-plane
  - host: controlplane-3
    ip: ${CP3_IP}
    role: control-plane