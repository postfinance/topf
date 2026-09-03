---
name: migrate-talhelper-to-topf
description: >
  Migrate a Talos cluster from talhelper (budimanjojo/talhelper, now archived) to
  TOPF (postfinance/topf). Use when the user wants to convert a talconfig.yaml-based
  setup to topf.yaml + patch files, asks "how do I move off talhelper", or says they
  want to migrate/switch/transition to topf. Triggers: talhelper, talconfig.yaml,
  talsecret.sops.yaml, talenv.sops.yaml, "migrate to topf", "switch from talhelper",
  "talhelper is archived". Covers generating the new topf.yaml, extracting inline
  node config into patch files, rewriting JSON patches as strategic merge, moving
  envsubst/talenv values into SOPS-encrypted `data` + Go templates, and renaming
  talsecret.sops.yaml to secrets.yaml.
---

# Migrating from talhelper to TOPF

talhelper is archived (since Aug 2026). TOPF is a recommended successor and the
upstream migration guide is canonical:
https://postfinance.github.io/topf/main/migration-from-talhelper/

## Critical: stay on the v1.13 config format for the migration

TOPF does not yet support the Talos v1.14 multi-document machine config. When
migrating an *existing* cluster:

1. Migrate the cluster to TOPF **keeping the old (v1.13) single-document config
   format**.
2. Do the v1.14 multi-doc migration **later**, as a separate step, using the
   `talos-v114-migration` skill.

Concretely: do not enable v1.14-style multi-doc patches during this migration.
Keep single-document strategic-merge patch files (`machine:` / `cluster:` maps).
The examples below all assume the v1.13 single-doc format.

## When to use

Use this skill when the task involves ANY of:

- Migrating a cluster managed by talhelper to TOPF
- Converting `talconfig.yaml` to `topf.yaml` + patch files
- A user asks to "migrate to topf", "switch from talhelper", or "move off talhelper"
- Handling `talenv.sops.yaml` / `talsecret.sops.yaml` as part of a topf migration

Do NOT use this skill for:

- The Talos v1.14 multi-doc config format migration (use `talos-v114-migration`)
- Writing a topf config from scratch with no existing talhelper setup
- Talos version upgrades that don't involve a tooling migration

## How talhelper and TOPF differ

| Aspect       | talhelper                                 | TOPF                                                     |
| ------------ | ----------------------------------------- | -------------------------------------------------------- |
| Config file  | `talconfig.yaml`                          | `topf.yaml`                                              |
| Patches      | Inline in config or separate files        | Separate files in `all/`, `<role>/`, `node/<host>/`      |
| Patch format | Strategic merge + JSON patches (RFC 6902) | Strategic merge only (with `$patch: delete` support)     |
| Secrets file | `talsecret.sops.yaml`                     | `secrets.yaml` (same format, just renamed)               |
| Env secrets  | `talenv.sops.yaml` + envsubst             | SOPS-encrypted `data` fields in `topf.yaml`              |
| Templating   | envsubst / talhelper variables            | Go templates (`.Data`, `.Node.Data`, sprig functions)    |
| Workflow     | `talhelper genconfig` then `talosctl apply-config` | `topf apply` (generates + applies in one step)  |

## Reference documentation

- **Upstream migration guide**:
  https://postfinance.github.io/topf/main/migration-from-talhelper/
- **TOPF configuration reference**:
  https://postfinance.github.io/topf/main/configuration/
- **TOPF configuration model (templating, secret resolution)**:
  https://postfinance.github.io/topf/main/configuration-model/
- **talhelper config reference (archived)**:
  https://budimanjojo.github.io/talhelper/latest/reference/configuration/
- **Full talhelper → topf field mapping**: see [`field-mapping.md`](field-mapping.md)
  in this skill directory.

## Workflow

Work through the user's `talconfig.yaml` in this order. Always read the actual
`talconfig.yaml` (and `talenv.sops.yaml` / `talsecret.sops.yaml` if present) in the
repo before editing — do not guess at field names.

### 1. Inventory the talhelper setup

Read `talconfig.yaml` and record:

- Top-level cluster fields (`clusterName`, `endpoint`, `talosVersion`,
  `kubernetesVersion`, `domain`, `allowSchedulingOnMasters`, etc.)
- Top-level `patches:`, `controlPlane:` block, `worker:` block,
  `inlineManifests:`
- Per-node fields (every key under each `nodes:` entry)
- Whether `talenv.sops.yaml` / `talenv.yaml` exists and what keys it holds
- Whether `talsecret.sops.yaml` / `talsecret.yaml` exists

### 2. Create `topf.yaml`

Translate the top-level cluster fields and the node list. Only these fields have
direct equivalents in `topf.yaml`; **everything else becomes a patch** (Step 3).

| talconfig.yaml              | topf.yaml            | Notes                                            |
| --------------------------- | -------------------- | ------------------------------------------------ |
| `clusterName`               | `clusterName`        | unchanged                                        |
| `endpoint`                  | `clusterEndpoint`    | renamed                                          |
| `kubernetesVersion`         | `kubernetesVersion`  | unchanged (keep the same value, e.g. `v1.32.8`) |
| `talosVersion`              | `talosVersion`       | unchanged                                        |
| node `hostname`             | node `host`          | renamed                                          |
| node `ipAddress`            | node `ip`            | renamed                                          |
| node `controlPlane: true`   | node `role: control-plane` | bool → enum                                |
| node `controlPlane: false`  | node `role: worker`        | bool → enum                                |

Example minimal `topf.yaml`:

```yaml
clusterName: mycluster
clusterEndpoint: https://192.168.1.100:6443
kubernetesVersion: v1.32.8
talosVersion: v1.12.0
nodes:
  - host: node-01
    ip: 192.168.1.1
    role: control-plane
  - host: node-02
    ip: 192.168.1.2
    role: worker
```

For the full field map (including fields with no direct equivalent), see
[`field-mapping.md`](field-mapping.md).

### 3. Extract node config into patch files

Patches live in a directory tree next to `topf.yaml` (default: same dir; override
with `patchesDir`). TOPF loads, in order, for each node:

```
<patchesDir>/all/             # applied to every node
<patchesDir>/control-plane/   # applied to control-plane nodes only
<patchesDir>/worker/          # applied to worker nodes only
<patchesDir>/node/<host>/     # applied to one specific node
```

Files are matched by `*.yaml`, `*.yml`, or `*.tpl` (templated). They are applied in
filesystem walk order within each folder, so prefix with two-digit numbers to
control ordering (`01-…`, `02-…`).

Map each talhelper source to a target directory:

| talhelper source                            | TOPF patch directory |
| ------------------------------------------- | -------------------- |
| top-level `patches:`                        | `all/`               |
| top-level `controlPlane: patches:`          | `control-plane/`     |
| top-level `worker: patches:`                | `worker/`            |
| top-level `inlineManifests:`                | `all/`               |
| node-level `patches:`                       | `node/<host>/`       |
| node-level config fields (installDisk, networkInterfaces, nodeLabels, …) | `node/<host>/` |

Each patch file is a single standalone YAML document — a strategic merge patch
against the Talos v1.13 machine config (`machine:` / `cluster:` maps).

Example per-node install + network patches:

```yaml
# node/node-01/01-install.yaml
machine:
  install:
    disk: /dev/nvme0n1
```

```yaml
# node/node-01/02-network.yaml
machine:
  network:
    interfaces:
      - interface: eno1
        dhcp: true
```

For a control-plane VIP shared by all CP nodes, put it under `control-plane/`:

```yaml
# control-plane/01-vip.yaml
machine:
  network:
    interfaces:
      - interface: eno1
        vip:
          ip: 192.168.1.100
```

### 4. Convert JSON patches to strategic merge

talhelper accepts RFC 6902 JSON patches (arrays of `{op, path, value}`). TOPF
does **not** — it only accepts strategic merge patches, and will reject a document
that is a YAML array with an error like *"document at index N looks like a JSON
patch (array of operations), which is not supported"*.

**Remove a field** (e.g. a node label):

```yaml
# Before (RFC 6902)
- op: remove
  path: /machine/nodeLabels/node.kubernetes.io~1exclude-from-external-load-balancers
```

```yaml
# After (strategic merge, $patch: delete)
machine:
  nodeLabels:
    node.kubernetes.io/exclude-from-external-load-balancers:
      $patch: delete
```

Note `/` in keys is written literally — no `~1` escaping.

**Add/replace a field**: write the strategic merge directly:

```yaml
# Before
- op: add
  path: /machine/kubelet/extraArgs/rotate-server-certificates
  value: "true"
```

```yaml
# After
machine:
  kubelet:
    extraArgs:
      rotate-server-certificates: "true"
```

If a talhelper patch was already strategic merge (a mapping, not an array), it
carries over unchanged — just move it into the right file.

### 5. Migrate envsubst / talenv to Go templates

talhelper reads `talenv.sops.yaml` (or `talenv.yaml`) into env vars and runs
envsubst on `talconfig.yaml` and patch files. TOPF has no envsubst.

**Pattern A — non-secret values**: move them under `data:` in `topf.yaml`:

```yaml
# topf.yaml
data:
  controlPlaneEndpoint: 192.168.1.100
  domain: mycluster.local
```

```yaml
# control-plane/01-extra-SANs.yaml.tpl
cluster:
  apiServer:
    certSANs:
      - {{ .Data.controlPlaneEndpoint }}
```

The `.tpl` suffix triggers Go template rendering. Templates use the sprig function
library and `missingkey=error` (a missing key is a hard error, not a blank).

**Pattern B — secret values**: put them under `data:` in `topf.yaml` and encrypt
the whole `topf.yaml` with SOPS (or use [vals](https://github.com/helmfile/vals)
references like `ref+vault://…`). SOPS-encrypted values in `topf.yaml` are
decrypted at load time and redacted from output by default.

```yaml
# topf.yaml (SOPS-encrypted)
data:
  controlPlaneEndpoint: ENC[AES256_GCM,data:...,type:str]
```

Template files reference them the same way: `{{ .Data.controlPlaneEndpoint }}`.

**Per-node data**: a node's `data:` block is reachable in templates as
`.Node.Data.<key>` (the node the patch is being rendered for). Use this for
per-node values that used to be envsubst'd with node-specific vars.

**talhelper template variables** like `{{ .MachineConfig … }}` do **not** exist in
TOPF. Replace with the TOPF template context:

| talhelper var          | TOPF template                           |
| ---------------------- | --------------------------------------- |
| `{{ .ClusterName }}`   | `{{ .ClusterName }}`                    |
| (hostname)             | `{{ .Node.Host }}`                      |
| (node IP)              | `{{ .Node.IP }}`                        |
| (node data `foo`)      | `{{ .Node.Data.foo }}`                  |
| (global data `foo`)    | `{{ .Data.foo }}`                       |
| `.MachineConfig.…`     | not available — restructure as a patch  |

### 6. Rename the secrets bundle

`talsecret.sops.yaml` (or `talsecret.yaml`) → `secrets.yaml`. The Talos secrets
bundle format is identical between talhelper and TOPF; only the filename changes.
TOPF looks for `secrets.yaml` next to `topf.yaml` by default (override with
`secretsPath`). Keep it SOPS-encrypted.

```bash
git mv talsecret.sops.yaml secrets.yaml
```

Delete `talenv.sops.yaml` / `talenv.yaml` — there is no equivalent in TOPF. Its
contents either become `data:` in `topf.yaml` (if still needed) or are dropped.

### 7. Apply

```bash
# Before (talhelper)
talhelper genconfig
talosctl apply-config --insecure -n 192.168.1.1 --file clusterconfig/mycluster-node-01.yaml
# ...repeat per node

# After (TOPF) — single command, all nodes
topf apply
```

For an existing cluster being migrated (nodes already running Talos), use
`--dry-run` first to diff the generated config against the running nodes and
confirm nothing unexpected changes:

```bash
topf apply --dry-run
topf apply
```

`topf apply` generates the machine config from patches + secrets and applies it to
each node over the Talos API (not `--insecure` file apply). It needs a valid
`talosconfig` — generate one with `topf talosconfig > talosconfig` and merge it, or
reuse the existing one from the talhelper-managed cluster.

### 8. Verify

- Re-read the generated `topf.yaml` and every patch file to confirm the YAML is
  valid and field names match the Talos v1.13 machine config reference.
- Run `topf nodes` to confirm the config compiles and all nodes are discovered.
- Run `topf apply --dry-run` to diff against the live cluster. For an
  already-running cluster, the diff should ideally show no unexpected changes.
- Only run `topf apply` with the user's explicit approval.

## Gotchas

### `installDisk` is not a topf.yaml node field

It goes in a patch: `machine.install.disk: /dev/nvme0n1` under `node/<host>/`.

### `networkInterfaces` is not a topf.yaml node field

It goes in a patch under `machine.network.interfaces` (per-node) or
`control-plane/` if shared by CP nodes.

### `schematic` / extensions

In TOPF, set `schematicId` in `topf.yaml` (or per node). Prefer the `@schematic.yaml`
reference form (see the topf configuration docs) over hand-computing the hash. The
schematic YAML is the same shape as talhelper's `schematic:` block — move it into
its own file.

### `imageFactory` (self-hosted factory)

Set `factory:` in `topf.yaml` (or per node). The URL template customization
talhelper exposes is not available in TOPF; if the user relied on a non-default
template, flag it and ask how to proceed.

### `allowSchedulingOnMasters` / `allowSchedulingOnControlPlanes`

Becomes `cluster.allowSchedulingOnControlPlanes: true` in an `all/` patch.

### `additionalApiServerCertSans` / `additionalMachineCertSans`

Becomes `cluster.apiServer.certSANs: […]` (and/or `machine.certSANs`) in an `all/`
patch. Merge with any existing certSANs.

### `cniConfig`, `clusterPodNets`, `clusterSvcNets`, `domain`

Becomes `cluster.network.cni`, `cluster.network.podSubnets`,
`cluster.network.serviceSubnets`, `cluster.network.dnsDomain` in an `all/` patch.

### `patches:` entries starting with `@./file.yaml` (talhelper file includes)

The referenced file is already a standalone patch — copy it into the appropriate
patch directory as-is (rename to `*.yaml` if needed).

### `extraManifests:` (deprecated in talhelper)

Treat like `patches:` — move the referenced files into the patch tree.

### `overridePatches: true` on a node

TOPF always *appends* node patches after role patches; there's no override flag.
If the user relied on override semantics, inspect the role-level patch the node
was overriding and decide whether to edit the role patch or put an explicit
`$patch: delete` in the node patch.

### `filenameTmpl`

No equivalent. TOPF doesn't write per-node files to disk by default; it applies
directly. Ignore this field.

### `talosImageURL`

In TOPF the installer image comes from `factory` + `schematicId` + `talosVersion`.
If the user pinned a specific `talosImageURL`, translate to the matching
`factory`/`schematicId`/`talosVersion` triple, or flag it.

### `machineSpec`

Only used by talhelper for `genurl image`. TOPF's `topf upgrade` and image
generation use `talosVersion`, `schematicId`, `platform`, `secureboot` instead.
Map what you can; flag the rest.

## Verification checklist

Before telling the user they're done, confirm:

1. `topf.yaml` exists with `clusterName`, `clusterEndpoint`, `kubernetesVersion`,
   and a `nodes:` list where every node has `host`, `ip`, and `role`.
2. No talhelper-only fields (`hostname`, `ipAddress`, `controlPlane`,
   `installDisk`, `networkInterfaces`, `patches`, etc.) remain on node entries.
3. Patch tree exists: at least `all/`, plus `control-plane/` and/or `worker/` if
   there were role-level patches, plus `node/<host>/` for any per-node config.
4. No patch file is a YAML array at the top level (that would be a JSON patch) —
   all are mappings, or use `$patch: delete`.
5. All `{{ .… }}` template refs in `.tpl` files resolve against the TOPF context
   (`.Data`, `.Node.Data`, `.ClusterName`, `.Node.Host`, `.Node.IP`, …) — no
   envsubst `${VAR}` and no talhelper `.MachineConfig.…` left.
6. `secrets.yaml` exists (renamed from `talsecret.sops.yaml`); `talenv.sops.yaml`
   is removed or its values folded into `data:`.
7. `topf apply --dry-run` runs clean and the diff against the live cluster matches
   expectations.
