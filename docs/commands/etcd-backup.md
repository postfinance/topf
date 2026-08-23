# Etcd Backup Command

The `etcd backup` command creates and lists snapshot backups of the cluster's etcd database, stored in S3-compatible object storage.

## Configuration

Backups require a `backups` block in `topf.yaml` describing the storage. S3 is currently the only backend:

```yaml
backups:
  s3:
    endpoint: minio.example.com:9000 # or s3.amazonaws.com; http:// implies insecure
    bucket: talos-backups
    prefix: mycluster/ # optional, defaults to "<clusterName>/"
    region: us-east-1 # optional
    insecure: false # optional, disables TLS
    accessKeyId: AKIA... # optional, see credentials below
    secretAccessKey: ... # optional, see credentials below
```

**Credentials**: when `accessKeyId`/`secretAccessKey` are set, they are used as static credentials. Like any sensitive value in `topf.yaml`, they can be [SOPS-encrypted](../configuration-model.md#secret-resolution). When unset, the standard credential chain applies: `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` or `MINIO_ACCESS_KEY`/`MINIO_SECRET_KEY` environment variables, the shared AWS credentials file, and IAM/IRSA.

Any S3-compatible storage works: AWS S3, MinIO, Ceph RGW, Cloudflare R2.

!!! warning
    The snapshot contains the full etcd keyspace, **including all Kubernetes secrets in plaintext**. Treat the bucket with the same care as `secrets.yaml`: restrict access and consider server-side encryption and object locking.

## Flags

| Flag                                                     | Default | Description                                                                     |
| -------------------------------------------------------- | ------- | ------------------------------------------------------------------------------- |
| `--output`, `-o`                                         | `table` | Output format: `table` or `yaml` (`list` only)                                  |
| [`--nodes-filter`](../configuration.md#filtering-nodes) | -       | Regex pattern to filter which control-plane nodes to snapshot from (global flag) |

## Create

```bash
topf etcd backup create
```

This will:

1. Connect to the first reachable control-plane node matching the [`--nodes-filter`](../configuration.md#filtering-nodes) regex (nodes in maintenance mode are skipped)
1. Stream a consistent snapshot of the etcd database via the Talos API into a local staging file
1. Verify the snapshot offline: the sha256 integrity trailer added by the etcd maintenance API is checked against the payload (the same check the restore path performs), followed by a bbolt structure check and the etcd snapshot hash walk
1. Upload it as `<clusterName>-<timestamp>.snapshot`, followed by a manifest sidecar (`<name>.meta.yaml`) recording cluster name, source node, timestamp, Talos and Kubernetes versions, size, sha256 checksum, etcd revision and total keys

## List

```bash
topf etcd backup list
```

```text
+-------------------------------------+-------------------------+-----------------+--------+----------+----------+------+
| NAME                                | CREATED                 | NODE            | TALOS  | SIZE     | REVISION | KEYS |
+-------------------------------------+-------------------------+-----------------+--------+----------+----------+------+
| mycluster-20260823T190000Z.snapshot | 2026-08-23 19:00:00 UTC | controlplane-01 | 1.13.4 | 40.0 MiB |   123456 | 1200 |
+-------------------------------------+-------------------------+-----------------+--------+----------+----------+------+
```

## Restore

topf does not (yet) support restores.

## Example Usage

```bash
# Take a backup before an upgrade
topf etcd backup create
topf upgrade

# Snapshot from a specific control-plane node
topf --nodes-filter 'controlplane-02' etcd backup create

# List backups as YAML (e.g. for retention scripting)
topf etcd backup list -o yaml
```
