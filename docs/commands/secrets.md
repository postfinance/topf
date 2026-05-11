# Secrets Command

The `secrets` command retrieves or generates the `secrets.yaml` bundle for the cluster.

The secrets bundle contains all sensitive cluster material including private keys, tokens, and certificates.

## Confirmation

When no existing secrets bundle is found and a new one needs to be generated, topf will prompt for confirmation before creating and storing it (unless the global `--confirm=false` flag is set, see [global flags](../configuration.md#global-flags)). This prevents accidental secret generation in interactive usage. In CI/CD pipelines, use `--confirm=false` to skip the prompt.

## Example Usage

```bash
# Output secrets to stdout
topf secrets

# Save secrets to a file
topf secrets > secrets.yaml

# Generate secrets without confirmation (e.g. in CI)
topf secrets --confirm=false
```
