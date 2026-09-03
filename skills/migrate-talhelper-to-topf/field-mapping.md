# talhelper → topf field mapping reference

Field-by-field mapping from `talconfig.yaml` (talhelper) to `topf.yaml` + patch
files. Built from the talhelper config reference
(https://budimanjojo.github.io/talhelper/latest/reference/configuration/) and the
TOPF config reference (https://postfinance.github.io/topf/main/configuration/).

Fields marked **patch** have no direct `topf.yaml` equivalent — they become
strategic-merge patch files under the indicated directory. Keep them in the
v1.13 single-document `machine:` / `cluster:` format.

---

## Top-level cluster fields

| talconfig.yaml                  | topf.yaml / patch                                          | Notes |
| ------------------------------- | ---------------------------------------------------------- | ----- |
| `clusterName`                   | `clusterName`                                              | direct |
| `endpoint`                      | `clusterEndpoint`                                          | renamed |
| `talosVersion`                  | `talosVersion`                                             | direct |
| `kubernetesVersion`             | `kubernetesVersion`                                        | direct |
| `domain`                        | **patch** `all/`: `cluster.network.dnsDomain`              | |
| `allowSchedulingOnMasters`      | **patch** `all/`: `cluster.allowSchedulingOnControlPlanes` | deprecated alias |
| `allowSchedulingOnControlPlanes`| **patch** `all/`: `cluster.allowSchedulingOnControlPlanes` | |
| `additionalMachineCertSans`     | **patch** `all/`: `machine.certSANs`                       | deprecated in talhelper; merge with existing |
| `additionalApiServerCertSans`   | **patch** `all/`: `cluster.apiServer.certSANs`             | merge with existing |
| `clusterPodNets`                | **patch** `all/`: `cluster.network.podSubnets`             | |
| `clusterSvcNets`                | **patch** `all/`: `cluster.network.serviceSubnets`         | |
| `cniConfig`                     | **patch** `all/`: `cluster.network.cni`                    | |
| `patches`                       | **patch** `all/` (one file per entry)                      | split inline `|-` blocks into separate files |
| `inlineManifests`               | **patch** `all/`: `cluster.inlineManifests`                | |
| `imageFactory`                  | `factory` (partial)                                        | only `registryURL` → `factory`; URL templates not supported — flag if customized |

### `controlPlane:` / `worker:` blocks

| talhelper block field   | TOPF patch directory | Notes |
| ----------------------- | -------------------- | ----- |
| `controlPlane.patches`  | `control-plane/`     | one file per entry |
| `worker.patches`        | `worker/`            | one file per entry |
| `controlPlane.<other>`  | `control-plane/`     | node-config fields (networkInterfaces, nodeLabels, …) applied to all CP nodes |
| `worker.<other>`        | `worker/`            | same, for worker nodes |

The `controlPlane:` / `worker:` blocks in talhelper mirror the `NodeConfigs`
fields below — translate each field the same way, just target the role directory.

---

## Node fields

| talconfig.yaml node field        | topf.yaml / patch                                       | Notes |
| -------------------------------- | ------------------------------------------------------- | ----- |
| `hostname`                       | node `host`                                             | renamed |
| `ipAddress`                      | node `ip`                                               | renamed |
| `controlPlane: true`             | node `role: control-plane`                              | bool → enum |
| `controlPlane: false`            | node `role: worker`                                     | bool → enum |
| `installDisk`                    | **patch** `node/<host>/`: `machine.install.disk`        | |
| `installDiskSelector`            | **patch** `node/<host>/`: `machine.install.diskSelector`| |
| `networkInterfaces`              | **patch** `node/<host>/`: `machine.network.interfaces`  | |
| `nameservers`                    | **patch** `node/<host>/`: `machine.network.nameservers` | |
| `kernelModules`                  | **patch** `node/<host>/`: `machine.kernel.modules`      | |
| `machineFiles`                   | **patch** `node/<host>/`: `machine.files`               | |
| `nodeLabels`                     | **patch** `node/<host>/`: `machine.nodeLabels`          | |
| `nodeAnnotations`                | **patch** `node/<host>/`: `machine.nodeAnnotations`     | |
| `nodeTaints`                     | **patch** `node/<host>/`: `machine.nodeTaints`           | |
| `disableSearchDomain`            | **patch** `node/<host>/`: `machine.network.disableSearchDomain` | |
| `certSANs`                       | **patch** `node/<host>/`: `machine.certSANs`            | |
| `schematic`                      | `schematicId` (prefer `@schematic.yaml`) or **patch**   | move schematic block into its own file referenced via `schematicId: @schematic-<host>.yaml` |
| `imageSchematic`                 | flag for manual handling                                | no direct equivalent; used for ISO/boot image only |
| `talosImageURL`                  | derive from `factory` + `schematicId` + `talosVersion`  | flag if a specific pinned URL must be preserved |
| `machineSpec`                    | flag for manual handling                                | only used by talhelper `genurl image`; TOPF uses `platform`/`secureboot`/`talosVersion` |
| `volumes`                        | **patch** `node/<host>/`: `machine.volumes`             | Talos v1.13 block volume config |
| `userVolumes`                    | **patch** `node/<host>/`: `machine.userVolumes`         | |
| `ingressFirewall`                | **patch** `node/<host>/`: `machine.ingressFirewall`     | |
| `extensionServices`              | **patch** `node/<host>/`: `machine.extensionServices`   | |
| `patches`                        | **patch** `node/<host>/` (one file per entry)           | split inline `|-` blocks; `@./file.yaml` includes → copy the file in |
| `extraManifests`                 | **patch** `node/<host>/`                                | deprecated in talhelper; treat like `patches` |
| `overridePatches`                | no equivalent                                           | TOPF always appends; use `$patch: delete` to override |
| `overrideExtraManifests`         | no equivalent                                           | as above |
| `overrideMachineCertSANs`        | no equivalent                                           | as above |
| `ignoreHostname`                 | **patch** `node/<host>/`: `machine.network.hostname: ""` + stable hostname | |
| `noSchematicValidate`            | no equivalent                                           | drop |

### Node `NodeConfigs` fields (apply to `controlPlane:` / `worker:` blocks too)

These are the fields talhelper groups under `NodeConfigs`, used in both node
entries and role blocks. They all map to patch files as shown above.

---

## Secrets and env files

| talhelper file             | TOPF equivalent                              | Notes |
| -------------------------- | -------------------------------------------- | ----- |
| `talsecret.sops.yaml`      | `secrets.yaml` (SOPS-encrypted)              | same format, renamed; default path next to `topf.yaml` |
| `talsecret.yaml`           | `secrets.yaml`                               | same, unencrypted |
| `talenv.sops.yaml`         | `data:` in `topf.yaml` (SOPS-encrypted)      | no envsubst in TOPF; reference via `{{ .Data.<key> }}` in `.tpl` patches |
| `talenv.yaml`              | `data:` in `topf.yaml`                       | same, unencrypted |

---

## Templating context

| talhelper                    | TOPF template                 |
| ---------------------------- | ----------------------------- |
| `{{ .ClusterName }}`         | `{{ .ClusterName }}`          |
| `{{ .Hostname }}`            | `{{ .Node.Host }}`            |
| `{{ .IPAddress }}`           | `{{ .Node.IP }}`              |
| `{{ .Role }}`                | `{{ .Node.Role }}`            |
| `{{ .MachineConfig.… }}`     | not available — use a patch   |
| envsubst `${VAR}`            | `{{ .Data.VAR }}`             |
| node-specific envsubst var   | `{{ .Node.Data.<key> }}`      |

TOPF templates use the sprig function library and `missingkey=error`. A `.tpl`
file suffix triggers template rendering; plain `.yaml` files are not templated.

---

## Patch file conventions

- One strategic-merge document per file (v1.13 `machine:` / `cluster:` format).
- Directories: `all/`, `control-plane/`, `worker/`, `node/<host>/`.
- Applied in filesystem walk order — prefix `01-`, `02-` to control ordering.
- `*.yaml` / `*.yml` = plain; `*.tpl` = Go template.
- No RFC 6902 JSON patches (arrays of `{op, path}`). Use `$patch: delete` for
  removals.
- SOPS-encrypted patch files are decrypted at load time.