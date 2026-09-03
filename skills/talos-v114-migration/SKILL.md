---
name: talos-v114-migration
description: >
  Migrate a Talos Linux machine configuration from v1.13 (or earlier) v1alpha1
  single-document format to the v1.14 multi-document config format introduced
  in Talos 1.14. Use when migrating Talos config patches, converting deprecated
  `.machine.*` / `.cluster.*` v1alpha1 fields to their new multi-doc kinds
  (UnattendedInstallConfig, KubeletConfig, KubeNodeConfig, KubeNetworkConfig,
  KubeProxyConfig, KubeControllerManagerConfig, KubeAPIServerConfig, VolumeConfig,
  ResolverConfig, CRICustomizationConfig, Layer2VIPConfig, DHCPv4Config,
  DiscoveryServiceConfig, etc.). Triggers: "migrate talos config to 1.14",
  "talos v1.14 multi-doc", "deprecated v1alpha1 fields", "document-map",
  "UnattendedInstallConfig", "KubeletConfig", "VolumeConfig", "KubeNodeConfig".
---

# Migrating Talos config to v1.14 multi-doc format

Talos 1.14 introduces a multi-document configuration model that replaces many
fields in the legacy v1alpha1 `machine:` / `cluster:` blocks with dedicated
document kinds (`apiVersion: v1alpha1` + `kind: <Kind>`). The old fields remain
supported for backwards compatibility, but new features and fields are only
available in the multi-doc format.

This skill migrates config patches from the old format to the new one. It works
for any Talos deployment — topf patch directories, raw `talosctl gen config`
output, or hand-written machine configs.

## When to use

Use this skill when the task involves ANY of:

- Migrating Talos config patches from v1.13 (or earlier) to v1.14
- Converting deprecated `.machine.*` or `.cluster.*` v1alpha1 fields to multi-doc kinds
- A user asks to "migrate to Talos 1.14 config" or "use the new multi-doc format"
- Reviewing a Talos config for deprecated v1alpha1 fields

Do NOT use this skill for:

- Talos version upgrades that don't involve config format changes
- Writing new config from scratch (use `talosctl gen config` which already emits multi-doc)
- Non-Talos Kubernetes config

## Reference documentation

- **Document map (authoritative field → kind mapping)**:
  https://docs.siderolabs.com/talos/v1.14/reference/configuration/document-map
- **v1alpha1 config reference (legacy fields)**:
  https://docs.siderolabs.com/talos/v1.14/reference/configuration/v1alpha1/config
- **v1.14.0 release notes (behavior changes)**:
  https://github.com/siderolabs/talos/releases/tag/v1.14.0

Each new document kind has its own reference page under
`https://docs.siderolabs.com/talos/v1.14/reference/configuration/<group>/<kind>/`
(for example, `kubernetes/kubeletconfig`, `block/volumeconfig`,
`network/layer2vipconfig`). Fetch the page for a kind when you need the exact
field names and types.

## Workflow

### 1. Inventory all config files

Read every patch/config file in the target directory (or the single machine
config). Record each v1alpha1 field path in use. Common locations for topf:
`patches/` (global), `control-plane/`, `worker/`, `node/<hostname>/`. For raw
talosctl: a single `controlplane.yaml` / `worker.yaml`.

### 2. Classify each field

For every field, look it up in [`mapping.md`](mapping.md) (the condensed
field-by-field reference built from the document map). Classify it as:

- **Deprecated** → migrate to the indicated multi-doc kind
- **Not deprecated** → leave as v1alpha1
- **No multi-doc equivalent** → flag for manual handling (see Gotchas)

### 3. Rewrite deprecated fields as multi-doc documents

For each deprecated field, create or update a multi-doc patch file containing
the new `apiVersion: v1alpha1` + `kind: <Kind>` document. Move the field value
to the new document, renaming fields where the mapping requires it.

Multi-doc files may contain several `---`-separated documents of different
kinds — for example, a single file can hold both `VolumeConfig` for EPHEMERAL
and `VolumeConfig` for STATE.

### 4. Keep non-deprecated v1alpha1 fields as-is

Leave fields that are NOT deprecated in their original v1alpha1 form. The two
formats coexist in v1.14. See the "NOT deprecated" table in [`mapping.md`](mapping.md)
for the list.

### 5. Verify

- Re-read every rewritten file to confirm the YAML is valid and the document
  kinds/fields match the reference pages.
- If using topf, run `topf nodes` to confirm the config compiles, then
  `topf apply` (with user approval) to push to nodes.
- For raw talosctl, run `talosctl apply-config` against a single node first.

## Gotchas

### `$patch: delete` works on multi-doc maps too

Strategic-merge `$patch: delete` directives work on both v1alpha1 fields and
multi-doc document maps. Use them to remove auto-generated defaults or specific
keys:

- Delete an entire auto-generated document (e.g. Flannel):
  ```yaml
  ---
  apiVersion: v1alpha1
  kind: KubeFlannelCNIConfig
  $patch: delete
  ```
- Delete a specific taint key:
  ```yaml
  ---
  apiVersion: v1alpha1
  kind: KubeNodeConfig
  taints:
    node-role.kubernetes.io/control-plane:
      $patch: delete
  ```
- Delete a specific label key:
  ```yaml
  ---
  apiVersion: v1alpha1
  kind: KubeNodeConfig
  labels:
    node.kubernetes.io/exclude-from-external-load-balancers:
      $patch: delete
  ```

### `KubeFlannelCNIConfig` is auto-generated — omitting it is not enough

`talosctl gen config` emits a `KubeFlannelCNIConfig` document by default in
v1.14. If you use an external CNI (Cilium, Calico, etc.) and had
`cluster.network.cni.name: none` in v1alpha1, simply not including a
`KubeFlannelCNIConfig` patch will NOT disable Flannel — the generated default
will still be present in the final config. You must explicitly delete it:

```yaml
---
apiVersion: v1alpha1
kind: KubeFlannelCNIConfig
$patch: delete
```

### `DiscoveryServiceConfig` is auto-generated — `cluster.discovery.enabled: false` conflicts

`talosctl gen config` emits a `DiscoveryServiceConfig` document (with
`name: default`) by default in v1.14. Any v1alpha1 `cluster.discovery` block —
even `enabled: false` — is mutually exclusive with the auto-generated document
and Talos rejects the config with:

```
discovery service is already configured in .cluster.discovery of the v1alpha1 config
```

There is no multi-doc `enabled` field on `DiscoveryServiceConfig`; the new model
is that the feature is on when the document is present and off when it's absent.
To disable discovery, delete the auto-generated document (the `name: default` is
required so the `$patch: delete` targets the correct document in the
strategic-merge map):

```yaml
---
apiVersion: v1alpha1
kind: DiscoveryServiceConfig
name: default
$patch: delete
```

`DiscoveryIdentityConfig` (cluster ID/secret) is also auto-generated but is not
deprecated in the same way — leave it as-is.

### `allowSchedulingOnControlPlanes` → delete the control-plane taint

The v1alpha1 `cluster.allowSchedulingOnControlPlanes: true` maps to a
`KubeNodeConfig` document that deletes the default control-plane taint:

```yaml
---
apiVersion: v1alpha1
kind: KubeNodeConfig
taints:
  node-role.kubernetes.io/control-plane:
    $patch: delete
```

Use `$patch: delete` on the specific taint key rather than `taints: {}` (empty
map) — the former is more precise and survives if other taints are added later.

### `KmsgLogConfig` lacks `extraTags`

The new `KmsgLogConfig` document only has `name` and `url` — it does not support
the `extraTags` map that `machine.logging.destinations[].extraTags` provides.
If you rely on `extraTags`, keep using the v1alpha1 `machine.logging` field.

### Multi-doc and v1alpha1 equivalents are mutually exclusive

Talos v1.14 rejects a config that sets both a deprecated v1alpha1 field and its
multi-doc replacement (e.g. `machine.nodeLabels` and a `KubeNodeConfig`
document with `labels`). Each multi-doc kind has a `V1Alpha1ConflictValidate`
method that enforces this. Choose one form per field.

### `talosctl gen config` auto-generates v1.14 default documents

When generating config for v1.14, `talosctl gen config` (called by topf and
other tools) emits default documents that were not present in v1.13 generated
configs:

- `SecurityProfileConfig` with `workloadIsolation: true` — moves containerd,
  kubelet, and pods into a dedicated PID/mount namespace (`sandboxd`).
  Clusters **upgraded** via `talosctl upgrade` keep the old non-isolated
  behavior, but **config regeneration** (e.g. `topf apply`) gets the new
  default. If you don't want isolation, add a patch with
  `workloadIsolation: false`.
- `FilesystemTrimConfig` with `interval: 168h0m0s` — weekly fstrim on
  eligible volumes. Harmless, but be aware it's there.
- `KubeFlannelCNIConfig` — see the Flannel gotcha above.

These are NOT in your patches — they're injected at generation time. Review
the generated output (`topf` writes to `output/`, or `talosctl gen config`
writes to `controlplane.yaml`/`worker.yaml`) to see what defaults are being
applied.

### LUKS-encrypted EPHEMERAL may fail to close on first boot after upgrade

When transitioning from v1.13 (non-isolated) to v1.14 with
`workloadIsolation: true`, the first reboot may fail with:

```
error closing encrypted volume mapped to "luks2-EPHEMERAL":
  error closing luks2-EPHEMERAL: mapped device is still in use
```

This happens because stale mount references from the old namespace hold the
LUKS device (`/dev/dm-1`) open, preventing the volume controller from closing
and reopening it. A second reboot usually resolves it (the stale references
are released on the clean shutdown). If a node gets stuck in a
`failed -> failed` loop and doesn't self-recover, reset the EPHEMERAL
partition:

```bash
talosctl --nodes <ip> reset --system-labels-to-wipe=EPHEMERAL --reboot
```

The node rejoins etcd automatically (STATE partition holds the identity and is
not wiped). For control-plane nodes, ensure the cluster has quorum (3+ healthy
members) before resetting.