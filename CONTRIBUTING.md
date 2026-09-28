# Contributing to TOPF

## Philosophy

TOPF automates what you'd otherwise do manually with `talosctl` — with health
checks, diffs, safety prompts, and a declarative config model on top.
TOPF aims to remain minimal. We don't re-implement features for which TOPF
brings no value over `talosctl` (e.g. `talosctl upgrade-k8s`); discuss any new
feature in an issue before starting to implement it.

## Development

```sh
task lint    # go vet + golangci-lint (strict)
task test    # go test -v -race ./...
```

Requires Go 1.26+, [Task](https://taskfile.dev/), and `sops`/`age`/`vals` for secrets tests.

### Integration tests

The integration tests in `internal/integration` are gated behind the
`integration` build tag and run against a live Talos cluster. When developing
them, export:

```sh
export GOFLAGS="-tags=integration"
```

so your editor and plain `go build`/`go test` invocations include the
tagged files without having to remember `-tags integration` on every command.

Against a local QEMU cluster (requires `/dev/kvm`, `talosctl` and
`qemu-system-x86_64`; the cluster boots in maintenance mode and the suite
drives the full lifecycle, so re-run `task e2e:cluster` for a fresh cluster
before each full suite run):

```sh
task e2e:cluster   # create the cluster + render test/e2e/work/topf.yaml
task e2e           # run the suite
task e2e:destroy   # tear the cluster down
```

CI runs the same suite against an equivalent cluster provisioned by
[talosctl-cluster-action](https://github.com/home-operations/talosctl-cluster-action)
(see `.github/workflows/e2e.yml`); `task e2e:cluster` issues the same
`talosctl cluster create dev` invocation the action does.

## Pull requests

Pull requests against this repo must allow maintainers to push to your branch.
On GitHub this is the "Allow edits by maintainers" option, which is only
available when your PR is based on a branch other than your fork's main (e.g.
feat/my-feature).

## Conventions

- Conventional commits (`feat:`, `fix:`, `docs:`, …); changelog is generated from them.
- Every `.go` file starts with:

  ```go
  // Copyright 2026 PostFinance AG
  // SPDX-License-Identifier: MIT
  ```
