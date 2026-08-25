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

## Conventions

- Conventional commits (`feat:`, `fix:`, `docs:`, …); changelog is generated from them.
- Every `.go` file starts with:

  ```go
  // Copyright 2026 PostFinance AG
  // SPDX-License-Identifier: MIT
  ```
