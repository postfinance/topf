# Secrets Command

The `secrets` command retrieves or generates the `secrets.yaml` bundle for the cluster.

The secrets bundle contains all sensitive cluster material including private keys, tokens, and certificates.

## Example Usage

```bash
# Output secrets to stdout
topf secrets

# Save secrets to a file
topf secrets > secrets.yaml
```
