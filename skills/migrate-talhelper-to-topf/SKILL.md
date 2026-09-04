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
  envsubst/talenv values into SOPS-encrypted `data` + Go templates, renaming
  talsecret.sops.yaml to secrets.yaml, wrapping `@./file` inline-manifest includes into
  patches, and proving the TOPF render matches talhelper's before the first apply.
---

# Migrating from talhelper to TOPF

talhelper is archived (since Aug 2026). TOPF is a recommended successor and the
upstream migration guide is canonical:
https://postfinance.github.io/topf/main/migration-from-talhelper/

## Critical: migrate on the running config format, upgrade later

Do the tooling migration on whatever Talos version and config format the cluster runs
today, and prove the TOPF render is equivalent to talhelper's (Step 8) before the first
`topf apply`. Only then upgrade Talos or move to the v1.14 multi-document format, as a
separate change with the `talos-v114-migration` skill. Mixing the two makes the first diff
unreviewable: every hunk could be the tool or the format.

TOPF v0.6.0+ renders v1.14 multi-document configs too; that is not a reason to switch
formats during the migration. The examples below use the v1.13 single-document
`machine:` / `cluster:` format.

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
lexicographic order within each folder, so prefix with two-digit numbers to control
ordering (`01-…`, `02-…`). Lists with a merge key (`cluster.inlineManifests` by `name`,
`machine.files` by `path`) merge across scopes exactly as they did in talhelper — both
tools use Talos's own strategic-merge patcher — so a `control-plane/` patch that adds
inline manifests **appends** to the `all/` list instead of replacing it.

Map each talhelper source to a target directory:

| talhelper source                            | TOPF patch directory |
| ------------------------------------------- | -------------------- |
| top-level `patches:`                        | `all/`               |
| top-level `controlPlane: patches:`          | `control-plane/`     |
| top-level `worker: patches:`                | `worker/`            |
| top-level `inlineManifests:`                | `all/` (see the `@./file` gotcha below) |
| node-level `patches:`                       | `node/<host>/`       |
| node-level config fields (installDisk, networkInterfaces, nodeLabels, …) | `node/<host>/` |
| node `hostname:` (talhelper set it for you) | `all/05-hostname.yaml.tpl` — see gotchas |

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

Encrypt only the `data` block so the rest of `topf.yaml` stays diffable and editable
without `sops`, and move the values without ever writing plaintext to disk:

```yaml
# .sops.yaml — first matching rule wins, so list this before any catch-all
creation_rules:
  - path_regex: topf\.yaml$
    encrypted_regex: ^data$
    mac_only_encrypted: true
    age: <recipient>
```

```bash
# placeholders in data:, then encrypt in place, then overwrite each value from talenv
sops encrypt -i topf.yaml
sops set topf.yaml '["data"]["repoToken"]' "\"$(sops -d --extract '["REPO_TOKEN"]' talenv.sops.yaml)\""
sops filestatus topf.yaml    # {"encrypted":true}
```

A `.tpl` file is parsed by Go's template engine **in full, YAML comments included** — a
literal `{{` in a comment fails the render with `bad character` or `function not defined`.
Values with characters YAML could misread are safer as `{{ .Data.x | quote }}`.

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
contents either become `data:` in `topf.yaml` (if still needed) or are dropped. Also
delete `clusterconfig/` and its `.gitignore` entries (`talenv.yaml`, `talsecret.yaml`);
add `output/` instead — that is where `topf render` writes full configs, secrets included.

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
each node over the Talos API (not `--insecure` file apply). It authenticates with a
client certificate minted from `secrets.yaml`, so the talhelper-era `talosconfig` keeps
working too (same secrets bundle, same CA); `topf talosconfig` regenerates it if needed.

TOPF dials every node's `ip` (or `host`) on `:50000` **directly**. There is no
`talosctl -e <control-plane> -n <worker>` style apid proxying, so a NAT'd or LAN-only
node is only reachable when TOPF runs from a network that reaches it. Use
`--nodes-filter` to apply the reachable nodes from elsewhere.

### 8. Verify: prove the render matches talhelper's

Both tools emit Talos-machinery YAML, so the two renders can be diffed directly and
every hunk must be explainable. Take the talhelper baseline **before** deleting
`talconfig.yaml` / `talenv.sops.yaml`:

```bash
talhelper genconfig                                   # baseline, one last time
topf render -o /tmp/topf-out
diff <(yq -P . clusterconfig/<cluster>-<host>.yaml) <(yq -P . /tmp/topf-out/<host>.yaml)
```

Harmless hunks: multi-document **order** (TOPF emits documents in patch order — `all/`,
then role, then node — where talhelper put node and role documents first; Talos keys
documents by kind + name, so order is irrelevant), and any comment or quoting you changed
inside inline manifests. Anything else is a real difference —
typical culprits are a missing hostname patch, a role schematic that changed, or a field
talhelper defaulted (`machine.install.wipe`, `certSANs`) that TOPF does not.

- `topf schematic-ids` must print the IDs already in each node's `machine.install.image`;
  only a genuinely new schematic needs `--submit-to-factory`.
- `topf apply --dry-run` (exit code 2 = changes) shows Talos's own diff against the
  **running** config. That diff is textual, so document reordering appears even when the
  parsed config is identical — Talos still reports it as applicable without a reboot.
  Anything that was changed in talhelper but never applied to the node surfaces here too;
  review it, do not let the migration apply it unnoticed.
- Delete the rendered files afterwards: they contain the cluster secrets in plaintext.
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

talhelper also accepts `controlPlane.schematic` / `worker.schematic`, and a node-level
`schematic` **replaces** the role one rather than merging. Express that as one
`schematic.yaml.tpl` (`schematicId: "@schematic.yaml.tpl"`) keyed on `.Node.Role`, or as
per-node `schematicId` entries, and check the IDs with `topf schematic-ids`:

```yaml
customization:
  systemExtensions:
    officialExtensions:
      - siderolabs/crun
{{- if eq .Node.Role "worker" }}
      - siderolabs/intel-ucode
{{- end }}
```

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

### `inlineManifests[].contents: "@./file.yaml"` and `skipEnvsubst`

talhelper reads the file into `contents`; TOPF has no include mechanism (no `readFile`
template function). Two options:

- **Wrap the file into a plain patch** — no new tooling, and the manifest is visible in
  `topf apply` diffs. Go through a temp file: a single environment string is capped at
  128 KiB and an Argo CD or Cilium bundle exceeds that.

  ```bash
  SRC=./argocd-install.yaml yq -n \
    '{"cluster": {"inlineManifests": [{"name": "argo-install", "contents": load_str(strenv(SRC))}]}}' \
    > all/60-argo-install.yaml
  ```

- **vals `ref+file://`** in a plain patch (`contents: ref+file://argocd-install.yaml`).
  Needs the `vals` binary, resolves the path against the working directory, and TOPF
  redacts vals-resolved values, so the manifest shows as `*** redacted ***` in diffs.

`skipEnvsubst: true` has no equivalent and needs none: plain `.yaml` patches are never
templated, so `$f`, `${TMP}` and friends survive untouched. The trap is reversed — a
manifest containing `{{` must NOT live in a `.tpl` patch.

### Hostname: talhelper set it, TOPF does not

talhelper emitted `machine.network.hostname` (or a `HostnameConfig` document on newer
Talos) from each node's `hostname:`. TOPF's `host` is a display and selection label only.
Without a patch, the first `topf apply` renames every node to `talos-xxx-xxx` and the
render diff shows the hostname document missing:

```yaml
# all/05-hostname.yaml.tpl
apiVersion: v1alpha1
kind: HostnameConfig
auto: "off"
hostname: {{ .Node.Host }}
```

Use `machine.network.hostname: {{ .Node.Host }}` instead if the baseline render used
that form — match whatever talhelper produced.

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
   is removed or its values folded into `data:`; `sops filestatus` reports both
   `topf.yaml` and `secrets.yaml` encrypted; `clusterconfig/` is gone, `output/` ignored.
7. A hostname patch exists, and `topf schematic-ids` reproduces the IDs from the
   talhelper render.
8. `talhelper genconfig` vs `topf render` differ only in document order and hunks you
   can name; the rendered files are deleted afterwards.
9. `topf apply --dry-run` runs clean and the diff against the live cluster matches
   expectations.
