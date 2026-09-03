# v1.14 field mapping reference

Condensed field-by-field mapping from deprecated v1alpha1 paths to their new
multi-doc document kinds. Built from the official document map:
https://docs.siderolabs.com/talos/v1.14/reference/configuration/document-map

Each new kind has a reference page at:
`https://docs.siderolabs.com/talos/v1.14/reference/configuration/<group>/<kind-lowercase>/`

---

## machine.install → UnattendedInstallConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/runtime/unattendedinstallconfig

| v1alpha1 path | New kind | New field | Notes |
|---|---|---|---|
| `machine.install.disk` | `UnattendedInstallConfig` | `provisioning.diskSelector.match` | Use a CEL expression: `disk.dev_path == "/dev/nvme0n1"` |
| `machine.install.diskSelector` | `UnattendedInstallConfig` | `provisioning.diskSelector.match` | Convert selector fields to CEL expression |
| `machine.install.image` | `UnattendedInstallConfig` | `installer.image` | |
| `machine.install.wipe` | `UnattendedInstallConfig` | `provisioning.wipe` | |
| `machine.install.extraKernelArgs` | — | — | Deprecated, no multi-doc equivalent. Use Image Factory/imager. |
| `machine.install.extensions` | — | — | Deprecated, use custom install image. |

**Example**:
```yaml
# v1alpha1
machine:
  install:
    disk: /dev/nvme0n1

# v1.14 multi-doc
---
apiVersion: v1alpha1
kind: UnattendedInstallConfig
provisioning:
    diskSelector:
        match: disk.dev_path == "/dev/nvme0n1"
```

---

## machine.network → multiple network documents

### interfaces

`machine.network.interfaces` is deprecated but has no 1:1 multi-doc replacement.
Interface configuration splits across multiple documents (`DHCPv4Config`,
`LinkConfig`, `Layer2VIPConfig`, etc.) depending on the features used. Refer to
the [document map](https://docs.siderolabs.com/talos/v1.14/reference/configuration/document-map)
for the exact split. A simplified example for DHCP + VIP:

```yaml
# v1alpha1
machine:
  network:
    interfaces:
      - interface: eno1
        dhcp: true
        vip:
          ip: 192.168.63.80

# v1.14 multi-doc
---
apiVersion: v1alpha1
kind: DHCPv4Config
name: eno1
---
apiVersion: v1alpha1
kind: Layer2VIPConfig
name: 192.168.63.80
link: eno1
```

### Other network fields

| v1alpha1 path | New kind | Notes |
|---|---|---|
| `machine.network.hostname` | `HostnameConfig` | `hostname:` or `auto: stable\|off` |
| `machine.network.nameservers` | `ResolverConfig` | `nameservers[].address` |
| `machine.network.searchDomains` | `ResolverConfig` | `searchDomains.domains` |
| `machine.network.extraHostEntries` | `StaticHostConfig` | One document per host |
| `machine.network.kubespan` | `KubeSpanConfig` | |
| `machine.network.disableSearchDomain` | `ResolverConfig` | `searchDomains.disableDefault: true` |

---

## machine.features.hostDNS → ResolverConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/network/resolverconfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `machine.features.hostDNS.enabled` | `ResolverConfig` | `hostDNS.enabled` |
| `machine.features.hostDNS.forwardKubeDNSToHost` | `ResolverConfig` | `hostDNS.forwardKubeDNSToHost` |
| `machine.features.hostDNS.resolveMemberNames` | `ResolverConfig` | `hostDNS.resolveMemberNames` |

---

## machine.controlPlane → KubeControllerManagerConfig + KubeSchedulerConfig

| v1alpha1 path | New kind | Notes |
|---|---|---|
| `machine.controlPlane` | `KubeControllerManagerConfig` + `KubeSchedulerConfig` | Static pod overrides split into separate documents |

---

## machine.kubelet → KubeletConfig + KubeNodeConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/kubernetes/kubeletconfig
and https://docs.siderolabs.com/talos/v1.14/reference/configuration/kubernetes/kubenodeconfig

| v1alpha1 path | New kind | New field | Notes |
|---|---|---|---|
| `machine.kubelet.image` | `KubeletConfig` | `image` | |
| `machine.kubelet.clusterDNS` | `KubeletConfig` | `clusterDNS` | |
| `machine.kubelet.extraArgs` | `KubeletConfig` | `extraArgs` | |
| `machine.kubelet.extraConfig` | `KubeletConfig` | `config` | **Renamed**: `extraConfig` → `config` |
| `machine.kubelet.defaultRuntimeSeccompProfileEnabled` | `KubeletConfig` | `defaultRuntimeSeccompProfileEnabled` | |
| `machine.kubelet.credentialProviderConfig` | `KubeCredentialProviderConfig` | | |
| `machine.kubelet.registerWithFQDN` | `KubeNodeConfig` | `registerWithFQDN` | |
| `machine.kubelet.nodeIP.validSubnets` | `KubeNodeConfig` | `nodeIP.validSubnets` | |
| `machine.kubelet.skipNodeRegistration` | `KubeNodeConfig` | `skipNodeRegistration` | |
| `machine.kubelet.extraMounts` | — | — | Removed in multi-doc. No direct equivalent. |
| `machine.kubelet.disableManifestsDirectory` | — | — | Locked to `true` in multi-doc. |

---

## machine.nodeLabels / nodeTaints / nodeAnnotations → KubeNodeConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/kubernetes/kubenodeconfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `machine.nodeLabels` | `KubeNodeConfig` | `labels` |
| `machine.nodeAnnotations` | `KubeNodeConfig` | `annotations` |
| `machine.nodeTaints` | `KubeNodeConfig` | `taints` |

**Note**: `$patch: delete` on a label key has no multi-doc equivalent. Keep as v1alpha1.

---

## machine.disks → UserVolumeConfig

`machine.disks` is deprecated. The replacement `UserVolumeConfig` uses a different
model (named volumes with provisioning specs vs. raw disk partitions). Refer to
the [UserVolumeConfig reference](https://docs.siderolabs.com/talos/v1.14/reference/configuration/block/uservolumeconfig)
for field details.

---

## machine.systemDiskEncryption → VolumeConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/block/volumeconfig

| v1alpha1 path | New kind | New field | Notes |
|---|---|---|---|
| `machine.systemDiskEncryption.ephemeral` | `VolumeConfig` | `name: EPHEMERAL`, `encryption:` | One document per volume |
| `machine.systemDiskEncryption.state` | `VolumeConfig` | `name: STATE`, `encryption:` | One document per volume |

The `encryption` substructure (`provider`, `keys`, `cipher`, `keySize`,
`blockSize`, `options`) is unchanged.

**Example**:
```yaml
# v1alpha1
machine:
  systemDiskEncryption:
    ephemeral:
      provider: luks2
      keys:
        - nodeID: {}
          slot: 0

# v1.14 multi-doc
---
apiVersion: v1alpha1
kind: VolumeConfig
name: EPHEMERAL
encryption:
    provider: luks2
    keys:
        - nodeID: {}
          slot: 0
```

---

## machine.files → CRICustomizationConfig / EtcFileConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/cri/cricustomizationconfig
and https://docs.siderolabs.com/talos/v1.14/reference/configuration/runtime/etcfileconfig

| v1alpha1 path | New kind | Notes |
|---|---|---|
| `machine.files` (path `/etc/cri/conf.d/20-customization.part`) | `CRICustomizationConfig` | `name` must NOT be `customization` (reserved). Use a descriptive name. `content` holds the TOML fragment. |
| `machine.files` (path under `/etc/`) | `EtcFileConfig` | `name` is the path relative to `/etc/`. Talos-managed paths are rejected. |
| `machine.files` (other paths) | — | No general multi-doc equivalent. Review case-by-case. |

**Example** (CRI customization):
```yaml
# v1alpha1
machine:
  files:
    - path: /etc/cri/conf.d/20-customization.part
      op: create
      content: |
        [plugins."io.containerd.cri.v1.images"]
          discard_unpacked_layers = false

# v1.14 multi-doc
---
apiVersion: v1alpha1
kind: CRICustomizationConfig
name: spegel
content: |
  [plugins."io.containerd.cri.v1.images"]
    discard_unpacked_layers = false
```

---

## machine.env → EnvironmentConfig

| v1alpha1 path | New kind |
|---|---|
| `machine.env` | `EnvironmentConfig` |

---

## machine.time → TimeSyncConfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `machine.time.disabled` | `TimeSyncConfig` | (disable via document absence) |
| `machine.time.servers` | `TimeSyncConfig` | `servers` |
| `machine.time.bootTimeout` | `TimeSyncConfig` | `bootTimeout` |

---

## machine.sysctls / sysfs / kernel.modules → multi-doc

| v1alpha1 path | New kind | Notes |
|---|---|---|
| `machine.sysctls` | `SysctlConfig` | `params` map |
| `machine.sysfs` | `SysfsConfig` | |
| `machine.kernel.modules` | `KernelModuleConfig` | One document per module |

---

## machine.registries → Registry*Config

| v1alpha1 path | New kind |
|---|---|
| `machine.registries.mirrors` | `RegistryMirrorConfig` |
| `machine.registries.config` (TLS) | `RegistryTLSConfig` |
| `machine.registries.config` (auth) | `RegistryAuthConfig` |

---

## machine.features (other)

| v1alpha1 path | New kind | Notes |
|---|---|---|
| `machine.features.kubePrism` | `KubePrismConfig` | |
| `machine.features.kubernetesTalosAPIAccess` | `KubeTalosAPIAccessConfig` | |
| `machine.features.imageCache` | `ImageCacheConfig` | `local.enabled` → `local.enabled` |
| `machine.features.stableHostname` | `HostnameConfig` | Use `auto: stable` |

---

## machine.udev → UdevRulesConfig

| v1alpha1 path | New kind |
|---|---|
| `machine.udev.rules` | `UdevRulesConfig` |

---

## machine.pods → KubeStaticPodConfig

| v1alpha1 path | New kind |
|---|---|
| `machine.pods` | `KubeStaticPodConfig` |

---

## machine.baseRuntimeSpecOverrides → CRIBaseRuntimeSpecConfig

| v1alpha1 path | New kind |
|---|---|
| `machine.baseRuntimeSpecOverrides` | `CRIBaseRuntimeSpecConfig` |

---

## cluster.name / cluster.controlPlane.endpoint → KubeClusterConfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `cluster.name` | `KubeClusterConfig` | `clusterName` |
| `cluster.controlPlane.endpoint` | `KubeClusterConfig` | `endpoint` |

---

## cluster.network → KubeNetworkConfig + KubeFlannelCNIConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/kubernetes/kubenetworkconfig
and https://docs.siderolabs.com/talos/v1.14/reference/configuration/kubernetes/kubeflannelcniconfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `cluster.network.podSubnets` | `KubeNetworkConfig` | `podSubnets` |
| `cluster.network.serviceSubnets` | `KubeNetworkConfig` | `serviceSubnets` |
| `cluster.network.dnsDomain` | `KubeNetworkConfig` | `dnsDomain` |
| `cluster.network.cni.name: none` | `KubeFlannelCNIConfig` | `$patch: delete` (see note below) |
| `cluster.network.cni.name: flannel` | `KubeFlannelCNIConfig` | (default, auto-generated) |
| `cluster.network.cni.flannel.*` | `KubeFlannelCNIConfig` | `backendType`, `backendMTU`, etc. |

**Note**: `nodeCIDRMaskSizeIPv4` (default 24) and `nodeCIDRMaskSizeIPv6` (default 64) are new fields on `KubeNetworkConfig` that were previously controller-manager args.

**Warning**: `talosctl gen config` auto-generates a `KubeFlannelCNIConfig` document for v1.14. Omitting it from your patches is NOT enough to disable Flannel — the generated default will still be present. To disable Flannel (e.g. when using an external CNI like Cilium), you must explicitly delete the document with `$patch: delete`:

```yaml
---
apiVersion: v1alpha1
kind: KubeFlannelCNIConfig
$patch: delete
```

---

## cluster.controllerManager → KubeControllerManagerConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/kubernetes/kubecontrollermanagerconfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `cluster.controllerManager.extraArgs` | `KubeControllerManagerConfig` | `extraArgs` |
| `cluster.controllerManager.image` | `KubeControllerManagerConfig` | `image` |
| `cluster.controllerManager.disabled` | `KubeControllerManagerConfig` | `enabled: false` |

**Note**: `allocate-node-cidrs` and `node-cidr-mask-size-*` args may be replaced by `KubeNetworkConfig.nodeCIDRMaskSizeIPv4` / `nodeCIDRMaskSizeIPv6`.

---

## cluster.proxy → KubeProxyConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/kubernetes/kubeproxyconfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `cluster.proxy.disabled: true` | `KubeProxyConfig` | `enabled: false` |
| `cluster.proxy.image` | `KubeProxyConfig` | `image` |
| `cluster.proxy.mode` | `KubeProxyConfig` | `mode` |
| `cluster.proxy.extraArgs` | `KubeProxyConfig` | `extraArgs` |

---

## cluster.scheduler → KubeSchedulerConfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `cluster.scheduler.extraArgs` | `KubeSchedulerConfig` | `extraArgs` |
| `cluster.scheduler.image` | `KubeSchedulerConfig` | `image` |
| `cluster.scheduler.config` | `KubeSchedulerConfig` | `config` |
| `cluster.scheduler.disabled` | `KubeSchedulerConfig` | `enabled: false` |

---

## cluster.apiServer → KubeAPIServerConfig (+ others)

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/kubernetes/kubeapiserverconfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `cluster.apiServer.certSANs` | `KubeAPIServerConfig` | `certExtraSANs` | **Renamed** |
| `cluster.apiServer.extraArgs` | `KubeAPIServerConfig` | `extraArgs` |
| `cluster.apiServer.image` | `KubeAPIServerConfig` | `image` |
| `cluster.apiServer.extraVolumes` | `KubeAPIServerConfig` | `extraVolumes` |
| `cluster.apiServer.env` | `KubeAPIServerConfig` | `env` |
| `cluster.apiServer.admissionControl` | `KubeAdmissionControlConfig` | |
| `cluster.apiServer.auditPolicy` | `KubeAuditPolicyConfig` | |
| `cluster.apiServer.authorizationConfig` | `KubeAuthorizerConfig` | |
| `cluster.controlPlane.localAPIServerPort` | `KubeAPIServerConfig` | `apiPort` |

---

## cluster.discovery → DiscoveryServiceConfig

**Reference**: https://docs.siderolabs.com/talos/v1.14/reference/configuration/cluster/discoveryserviceconfig

| v1alpha1 path | New kind | Notes |
|---|---|---|
| `cluster.discovery.enabled: true` | — | Default in v1.14, can be dropped |
| `cluster.discovery.enabled: false` | `DiscoveryServiceConfig` | No multi-doc `enabled` field. `talosctl gen config` auto-generates a `DiscoveryServiceConfig` (name: `default`) which conflicts with the v1alpha1 `cluster.discovery` block. To disable discovery, delete the auto-generated document: see example below. |
| `cluster.discovery.registries.service.endpoint` | `DiscoveryServiceConfig` | `endpoint` |
| `cluster.id` | `DiscoveryIdentityConfig` | |
| `cluster.secret` | `DiscoveryIdentityConfig` | |

**Disabling discovery in v1.14**: `talosctl gen config` auto-generates a
`DiscoveryServiceConfig` document with `name: default`. Any v1alpha1
`cluster.discovery` block (even `enabled: false`) is mutually exclusive with it
and Talos rejects the config. To disable discovery, delete the auto-generated
document:

```yaml
---
apiVersion: v1alpha1
kind: DiscoveryServiceConfig
name: default
$patch: delete
```

The `name: default` is required so the `$patch: delete` targets the correct
document in the strategic-merge map.

---

## cluster.allowSchedulingOnControlPlanes → KubeNodeConfig

| v1alpha1 path | New kind | New field |
|---|---|---|
| `cluster.allowSchedulingOnControlPlanes: true` | `KubeNodeConfig` | `taints.node-role.kubernetes.io/control-plane: { $patch: delete }` |
| `cluster.allowSchedulingOnMasters: true` | `KubeNodeConfig` | Deprecated alias |

**Example**:
```yaml
---
apiVersion: v1alpha1
kind: KubeNodeConfig
taints:
  node-role.kubernetes.io/control-plane:
    $patch: delete
```

**Note**: `taints: {}` (empty map) replaces the entire taints map, which also removes the taint, but `$patch: delete` on the specific key is more precise and survives if other taints are added later.

---

## cluster.inlineManifests / extraManifests → multi-doc

| v1alpha1 path | New kind |
|---|---|
| `cluster.inlineManifests` | `KubeInlineManifestConfig` |
| `cluster.extraManifests` | `KubeExternalManifestConfig` |
| `cluster.extraManifestHeaders` | `KubeExternalManifestConfig` |

---

## cluster encryption → KubeEtcdEncryptionConfig

| v1alpha1 path | New kind |
|---|---|
| `cluster.aescbcEncryptionSecret` | `KubeEtcdEncryptionConfig` |
| `cluster.secretboxEncryptionSecret` | `KubeEtcdEncryptionConfig` |

---

## cluster CA fields → multi-doc

| v1alpha1 path | New kind |
|---|---|
| `cluster.ca` | `KubeAPIServerCAConfig` |
| `cluster.acceptedCAs` | `KubeAPIServerCAConfig` |
| `cluster.aggregatorCA` | `KubeAggregatorCAConfig` |
| `cluster.serviceAccount` | `KubeServiceAccountConfig` |

---

## cluster.coreDNS → KubeCoreDNSConfig

| v1alpha1 path | New kind |
|---|---|
| `cluster.coreDNS` | `KubeCoreDNSConfig` |

---

## NOT deprecated — keep as v1alpha1

These fields have no multi-doc replacement in v1.14 and should be left in their
v1alpha1 form:

| v1alpha1 path | Reason |
|---|---|
| `machine.logging` | No multi-doc equivalent with `extraTags` parity. `KmsgLogConfig` only has `name` + `url`. |
| `cluster.etcd` | Not deprecated in v1.14. `cluster.etcd.extraArgs`, `.image`, `.ca`, `.advertisedSubnets`, `.listenSubnets` all remain v1alpha1. |
| `machine.features.rbac` | Not deprecated. |
| `machine.features.diskQuotaSupport` | Not deprecated. |
| `machine.features.nodeAddressSortAlgorithm` | Not deprecated. |
| `machine.features.apidCheckExtKeyUsage` | Not deprecated. |
| `machine.seccompProfiles` | Not deprecated. |
| `machine.certSANs` | Not deprecated. |
| `machine.token` | Not deprecated. |
| `machine.ca` | Not deprecated. |
| `cluster.token` | Not deprecated. |
| `cluster.externalCloudProvider` | Not deprecated. |
| `cluster.adminKubeconfig` | Not deprecated. |
| `machine.nodeLabels` with `$patch: delete` | Strategic-merge `$patch: delete` has no multi-doc equivalent. |

---

## No multi-doc equivalent — flag for manual handling

| Pattern | Issue |
|---|---|
| `machine.nodeLabels.<key>: { $patch: delete }` | `KubeNodeConfig.labels` is `map[string]string`, cannot express deletion. Keep v1alpha1. |
| `machine.kubelet.extraMounts` | Removed in multi-doc, no direct replacement. |
| `machine.install.extraKernelArgs` | Deprecated, use Image Factory/imager instead. |
| `machine.install.extensions` | Deprecated, use custom install image. |
| `machine.files` (non-CRI, non-/etc paths) | No general multi-doc equivalent. Review case-by-case. |