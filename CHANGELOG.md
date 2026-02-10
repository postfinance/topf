# Changelog

## [0.1.2] - 2026-02-10

### Build

- **brew**: Remove quarantine flag on topf binary ([b0eb4d6](https://github.com/postfinance/topf/commit/b0eb4d696a98a2367ac198718a279009461ef59f))

### Miscellaneous

- Update changelog ([bf838b4](https://github.com/postfinance/topf/commit/bf838b498013b9d4c6751d726725c140fbd56332))
## [0.1.1] - 2026-02-10

### Build

- Configure homebrew integration ([8a65146](https://github.com/postfinance/topf/commit/8a6514617b76e23ac5c43e90a4912c7df9587849))

### Documentation

- Create mkdocs documentation ([d87c57c](https://github.com/postfinance/topf/commit/d87c57c4ff4009f4a94a9e5c1ab1ae742cad5098))

### Miscellaneous

- Update changelog ([e6dee58](https://github.com/postfinance/topf/commit/e6dee58d305f767a476b06ed757b6639e3ed3df3))
## [0.1.0] - 2026-02-10

### Bug Fixes

- Add validation when retrieving secrets ([bbdf93d](https://github.com/postfinance/topf/commit/bbdf93d297323a48859c98a09612fe78f58b7ab2))
- Use go/yaml/v4 in secrets.go ([7e7a5b4](https://github.com/postfinance/topf/commit/7e7a5b4306f752228dc024e08fdb567bd0f2948b))
- **bootstrap**: Check etcd service state instead of counting members ([24d8089](https://github.com/postfinance/topf/commit/24d80890c6bc2a8698e60164a9a01ca4c5fa24d0))
- Skip empty patches instead of aborting ([1d0caa5](https://github.com/postfinance/topf/commit/1d0caa58cba2216077e3bf0b2207acf3052a0651))
- Prevent access nil pointer and use node getters ([3f9dffe](https://github.com/postfinance/topf/commit/3f9dffeb7db15fd562cdd047b6a8782f21e53742))
- Log proper result and warn about extraneous args ([f5bd40e](https://github.com/postfinance/topf/commit/f5bd40e447a6eb0e8dda30ba7a8505d231af1b83))
- **patch**: Issue an error for missing keys and apply patches based on role ([a958763](https://github.com/postfinance/topf/commit/a958763d8a93a6645a55a047ffc6e1a4ed5ffb52))
- **config**: Also filter topf.yaml:nodes entries with the regexp ([313b4a4](https://github.com/postfinance/topf/commit/313b4a4c4b3aaeef5fb59d04b7caf9cd3789536b))
- **nodes**: Output error message correctly with -o yaml ([e6e66fb](https://github.com/postfinance/topf/commit/e6e66fb87c1c0081ff5646e2c83462317c2ef84d))

### Build

- Add goreleaser workflow ([1343686](https://github.com/postfinance/topf/commit/134368618feb480f49c4896ac2e15d41e51dbd57))
- Refactor NewTopfRuntime to use a struct ([83c8323](https://github.com/postfinance/topf/commit/83c832305408d23171dbd7dff9fa6495af57816d))
- Dependencies upgrade ([88e9bb8](https://github.com/postfinance/topf/commit/88e9bb8baac47f09b63fb2c2369c0a8977d36a4e))
- Add version flag ([266ccbd](https://github.com/postfinance/topf/commit/266ccbd973f6869f65953946ae591f7a26acb1e9))

### Documentation

- Use git-cliff for changelog ([6a5416d](https://github.com/postfinance/topf/commit/6a5416d0a15499acb910bfdc49039c4f2464733e))
- Add 'Alternatives' section at the end of the readme ([7e5b8cd](https://github.com/postfinance/topf/commit/7e5b8cd31bf9f4d896c0349725244f6fdedfeb72))
- Update README.md ([b7b26fd](https://github.com/postfinance/topf/commit/b7b26fdda90fe3cbb6044d1b799848f622f9455e))
- Update readme with combined bootstrap/apply command ([3c44395](https://github.com/postfinance/topf/commit/3c44395706f806e6711a710d35c16d7857d77e68))

### Features

- Add dry-run mode for apply and upgrade ([4f4d1b6](https://github.com/postfinance/topf/commit/4f4d1b610d5f8ff964cd4485bf70935c83bbfe79))
- **bootstrap**: Check whether etcd is bootstrapped first ([8d6c03c](https://github.com/postfinance/topf/commit/8d6c03c16f2f7a981949524dc70ccd78c2930435))
- Add allow-not-ready flag ([98b0e62](https://github.com/postfinance/topf/commit/98b0e623bfbbf4f75a36af1b24d52af6bd34c599))
- Add --config flag for config patches directory ([7b6bb7b](https://github.com/postfinance/topf/commit/7b6bb7ba0a44107bfa926a2003317a990ca5edc7))
- Add --confirm flag to reset command ([63b6ca9](https://github.com/postfinance/topf/commit/63b6ca98560eb765903aee2fb95241742256bab2))
- Merge apply and bootstrap cmds ([4cd174a](https://github.com/postfinance/topf/commit/4cd174a30522754d517d1f7fd979217ee4df8c13))
- **topf.yaml**: Decode SOPS-encrypted config and patches ([c07f126](https://github.com/postfinance/topf/commit/c07f1268d887841674d14f1d95a7891990b12920))
- **talosconfig**: Add all CP as endpoints and all nodes as nodes per default ([624b440](https://github.com/postfinance/topf/commit/624b440d659dd94d3701ef9f532fc2d63048bfc0))
- **nodes**: Permit exporting the machineconfigs to a directory ([2051f9e](https://github.com/postfinance/topf/commit/2051f9ede8c308e870ba2a7865ec4df91ba6a66f))
- Add ability to add arbitrary data in topf.yaml for templating ([e2ca12f](https://github.com/postfinance/topf/commit/e2ca12f53298bbea08eb39dd8604101eea99e581))

### Miscellaneous

- Add SPDX license ([861149b](https://github.com/postfinance/topf/commit/861149b39a9bbd0cc34b03cd49d774dfa7668e11))
- **bootstrap**: Improve logging ([cb1db05](https://github.com/postfinance/topf/commit/cb1db05c41ee11f2a37a7d467b1021bf2d1376f4))
- Rename nodes to nodes-filter ([ad90a46](https://github.com/postfinance/topf/commit/ad90a461a554036652461f62ac9aabbb3e310392))
- Move config-dir to topf.yaml ([3ccdc10](https://github.com/postfinance/topf/commit/3ccdc109b397864535cb2f4a69b60ce71381d812))
- Validate config-dir argument ([29b409b](https://github.com/postfinance/topf/commit/29b409bf8cc1fb8b61f012da6675b48e44aade4b))
- Add node.Attrs() function for simpler logging ([483331d](https://github.com/postfinance/topf/commit/483331d1794130ab1675ce0777a4a0a0a23f15b9))
- Move sops decryption function to utils/ ([ed9963a](https://github.com/postfinance/topf/commit/ed9963a5d7d02901312e9163f20b716389fb121e))
- Fail early if a patch is considered JSON patch ([1ab45ee](https://github.com/postfinance/topf/commit/1ab45eeba0b152e9c7d676617115d9c1259c34fc))
