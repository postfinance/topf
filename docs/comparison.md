# Comparison with talhelper

Both TOPF and [talhelper](https://github.com/budimanjojo/talhelper) help manage Talos cluster configurations declaratively, but they take different approaches.

## Approach

**talhelper** is a config generator. It produces full machine configuration files that you then apply with `talosctl`. It also provides helper commands for common `talosctl` operations.

**TOPF** uses the Talos Go libraries directly to generate configs and interact with nodes in a single tool — no intermediate files or dependency on `talosctl`.

## Configuration

**talhelper** keeps everything in a single `talconfig.yaml`, including inline patches and node-specific settings like `installDisk` and `networkInterfaces`.

**TOPF** separates node definitions (`topf.yaml`) from patches (individual files in `patches/`, `<role>/`, `nodes/<host>/`). This makes patches easier to review in pull requests.

## Patches

Both tools support strategic merge patches. talhelper additionally supports JSON patches (RFC 6902).

TOPF does not support JSON patches — they are incompatible with multi-document configs and are being [deprecated in Talos starting from v1.12](https://www.talos.dev/v1.12/talos-guides/configuration/patching/#json-patches).

## Templating

**talhelper** uses envsubst with variables from `talenv.yaml` / `talenv.sops.yaml`.

**TOPF** uses Go templates (`.yaml.tpl` files) with data from `topf.yaml`, offering conditionals, loops, and per-node data via `.Node.Data`.

## Secrets

**talhelper** uses a separate `talenv.sops.yaml` file for encrypted variables.

**TOPF** encrypts the `data` fields directly in `topf.yaml` with SOPS.

## Summary

|                         | talhelper                          | TOPF                                |
| ----------------------- | ---------------------------------- | ----------------------------------- |
| Approach                | Config generator + helper commands | Direct interaction via Go libraries |
| Apply / Upgrade / Reset | Via `talosctl`                     | Built-in                            |
| JSON patches            | Yes                                | No (deprecated in Talos v1.12)      |
| Strategic merge patches | Yes                                | Yes                                 |
| Templating              | envsubst                           | Go templates                        |
| SOPS integration        | `talenv.sops.yaml`                 | Directly in `topf.yaml`             |
| Node filtering          | No                                 | Yes (`--nodes-filter`)              |
