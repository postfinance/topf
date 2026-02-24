# Changelog

## [0.2.1] - 2026-02-17

### Bug Fixes

- Skew secret cert clock to past to handle time drift ([855755b](https://github.com/postfinance/topf/commit/855755b46f97633be37ae0dd700e23515bb4b52a))
- **lint**: Resolve gosec log injection warning ([23eca64](https://github.com/postfinance/topf/commit/23eca6438a8a0a61994ec6594fa6da66efea5f6b))

### Miscellaneous

- Update changelog ([6bedb54](https://github.com/postfinance/topf/commit/6bedb5426d28d9d0d5a44f646821f0c9d975b9df))
- **ci**: Fix git-cliff warnings for merge commits ([952a107](https://github.com/postfinance/topf/commit/952a107a5f251a36658efa33248ac0353bb52086))

### Build

- **deps**: Bump actions/setup-python from 5 to 6 ([afd3e79](https://github.com/postfinance/topf/commit/afd3e79c27131506d51fd6cf3aec2b8eb4235ece))
- **deps**: Bump actions/checkout from 5 to 6 ([0501cbf](https://github.com/postfinance/topf/commit/0501cbffb236d3e77f32790c77464960e4608471))
- **deps**: Bump actions/setup-go from 5 to 6 ([a54a89e](https://github.com/postfinance/topf/commit/a54a89eb799cd8db47576c58261f98b93b41054b))
- **deps**: Bump golangci/golangci-lint-action from 8 to 9 ([36fd37a](https://github.com/postfinance/topf/commit/36fd37a247cd0ae5426c0ff9ad3ee3f76e013c8e))
- **deps**: Bump actions/upload-pages-artifact from 3 to 4 ([c603e7c](https://github.com/postfinance/topf/commit/c603e7c6b5860871094170795f627164211f696a))
- **deps**: Bump go.yaml.in/yaml/v4 from 4.0.0-rc.3 to 4.0.0-rc.4 ([c1e23c2](https://github.com/postfinance/topf/commit/c1e23c2dbcf0e0463105bc8d67a7ebb4e4dd74aa))
- **deps**: Bump github.com/cosi-project/runtime from 1.13.0 to 1.14.0 ([2bffe91](https://github.com/postfinance/topf/commit/2bffe914d176e85318ac6da93bca3941dd196788))
## [0.2.0] - 2026-02-13

### Features

- **kubeconfig**: Make validity configurable via flag ([85e17b5](https://github.com/postfinance/topf/commit/85e17b58fc5bd21a4b2a802649462034c56f2b32))
- **BREAKING**: Rename `patches` folder to `all` ([415b82d](https://github.com/postfinance/topf/commit/415b82d721e55d1a2ee9403315a3b2f9675e6a46))
- **upgrade**: Make reboot mode configurable (#11) ([1d7f30e](https://github.com/postfinance/topf/commit/1d7f30e542a2d6df38e8e031098cdd18a61d0040))

### Bug Fixes

- **apply**: Only throw warning when --auto-bootstrap has no CP nodes ([2891762](https://github.com/postfinance/topf/commit/289176224d6f83cb58ec173ed6fb8de7fa3a70a3))

### Documentation

- Add early stage disclaimer ([f25d7b4](https://github.com/postfinance/topf/commit/f25d7b47a474216f07db1ebde86c5a9ff2ca1c0c))
- Topf schematic id and philosophy ([ed9f0b0](https://github.com/postfinance/topf/commit/ed9f0b0c099680d893ad75f8a1ca8bf000459e33))
- Installer-image and upgrade improvements ([48ca530](https://github.com/postfinance/topf/commit/48ca530448412f7e16c9093ea50d54c2a5f9b6e0))
- **readme**: Add demo link ([01e67b8](https://github.com/postfinance/topf/commit/01e67b8f15406275fc4344e5bc8729c51ce1ae1c))
- **migration**: Precise secrets relocation ([e129b69](https://github.com/postfinance/topf/commit/e129b691f088ddcf7a17eb90a4ef119efeb2e8da))
- **configuration-model**: Rm slash from role ([b875b9e](https://github.com/postfinance/topf/commit/b875b9e8a210cee3319f9cb6caca4589bbc03cfb))
- **apply**: Add --dry-run flag ([97a9493](https://github.com/postfinance/topf/commit/97a94933ac121ad08b4452d52830113401d6084f))

### Miscellaneous

- **kubeconfig**: Use context in username ([50b9c7f](https://github.com/postfinance/topf/commit/50b9c7f50551e10aa49046d3ae7933af2bd54c4a))
- Fix linter warning ([3b0992d](https://github.com/postfinance/topf/commit/3b0992da8b8c6b175c764a8540446b1c153c1517))
- Don't gitignore cmd/topf folder ([d65e45d](https://github.com/postfinance/topf/commit/d65e45d19bdeddd12b0e21e479a174c9664ec98f))
- **cliff**: Improve changelog and breaking changes visibility ([39f981a](https://github.com/postfinance/topf/commit/39f981a9da8fa312c646264961b0b881c04de058))

### Build

- Add dependabot.yml ([c2facce](https://github.com/postfinance/topf/commit/c2facce6853eec37dbdb8ee162daf07b045ee7ba))
- **deps**: Update go to v1.26.0, talos to 1.12.4 and k8s to 1.35 ([c9d71a9](https://github.com/postfinance/topf/commit/c9d71a9cd9b9e9ca0f6c1395d4066634f376a809))
- Add dev-release workflow ([3a017a7](https://github.com/postfinance/topf/commit/3a017a7a8a1d53e2a675e1b6a724812ed20dd9a3))
## [0.1.3] - 2026-02-11

### Miscellaneous

- Update changelog ([f605c93](https://github.com/postfinance/topf/commit/f605c9386a3a4a86ae2f84925e7e35ff0c04c142))

### Build

- **goreleaser**: 'v' prefix for container tags ([e5efd52](https://github.com/postfinance/topf/commit/e5efd526a63387dd8f5ffed8db0af3e4b91e0745))
## [0.1.2] - 2026-02-10

### Miscellaneous

- Update changelog ([bf838b4](https://github.com/postfinance/topf/commit/bf838b498013b9d4c6751d726725c140fbd56332))

### Build

- **brew**: Remove quarantine flag on topf binary ([b0eb4d6](https://github.com/postfinance/topf/commit/b0eb4d696a98a2367ac198718a279009461ef59f))
## [0.1.1] - 2026-02-10

### Documentation

- Create mkdocs documentation ([d87c57c](https://github.com/postfinance/topf/commit/d87c57c4ff4009f4a94a9e5c1ab1ae742cad5098))

### Miscellaneous

- Update changelog ([e6dee58](https://github.com/postfinance/topf/commit/e6dee58d305f767a476b06ed757b6639e3ed3df3))

### Build

- Configure homebrew integration ([8a65146](https://github.com/postfinance/topf/commit/8a6514617b76e23ac5c43e90a4912c7df9587849))
## [0.1.0] - 2026-02-10

### Features

- Add ability to add arbitrary data in topf.yaml for templating ([e2ca12f](https://github.com/postfinance/topf/commit/e2ca12f53298bbea08eb39dd8604101eea99e581))
- **nodes**: Permit exporting the machineconfigs to a directory ([2051f9e](https://github.com/postfinance/topf/commit/2051f9ede8c308e870ba2a7865ec4df91ba6a66f))
- **talosconfig**: Add all CP as endpoints and all nodes as nodes per default ([624b440](https://github.com/postfinance/topf/commit/624b440d659dd94d3701ef9f532fc2d63048bfc0))
- **topf.yaml**: Decode SOPS-encrypted config and patches ([c07f126](https://github.com/postfinance/topf/commit/c07f1268d887841674d14f1d95a7891990b12920))
- Merge apply and bootstrap cmds ([4cd174a](https://github.com/postfinance/topf/commit/4cd174a30522754d517d1f7fd979217ee4df8c13))
- Add --confirm flag to reset command ([63b6ca9](https://github.com/postfinance/topf/commit/63b6ca98560eb765903aee2fb95241742256bab2))
- Add --config flag for config patches directory ([7b6bb7b](https://github.com/postfinance/topf/commit/7b6bb7ba0a44107bfa926a2003317a990ca5edc7))
- Add allow-not-ready flag ([98b0e62](https://github.com/postfinance/topf/commit/98b0e623bfbbf4f75a36af1b24d52af6bd34c599))
- **bootstrap**: Check whether etcd is bootstrapped first ([8d6c03c](https://github.com/postfinance/topf/commit/8d6c03c16f2f7a981949524dc70ccd78c2930435))
- Add dry-run mode for apply and upgrade ([4f4d1b6](https://github.com/postfinance/topf/commit/4f4d1b610d5f8ff964cd4485bf70935c83bbfe79))

### Bug Fixes

- **nodes**: Output error message correctly with -o yaml ([e6e66fb](https://github.com/postfinance/topf/commit/e6e66fb87c1c0081ff5646e2c83462317c2ef84d))
- **config**: Also filter topf.yaml:nodes entries with the regexp ([313b4a4](https://github.com/postfinance/topf/commit/313b4a4c4b3aaeef5fb59d04b7caf9cd3789536b))
- **patch**: Issue an error for missing keys and apply patches based on role ([a958763](https://github.com/postfinance/topf/commit/a958763d8a93a6645a55a047ffc6e1a4ed5ffb52))
- Log proper result and warn about extraneous args ([f5bd40e](https://github.com/postfinance/topf/commit/f5bd40e447a6eb0e8dda30ba7a8505d231af1b83))
- Prevent access nil pointer and use node getters ([3f9dffe](https://github.com/postfinance/topf/commit/3f9dffeb7db15fd562cdd047b6a8782f21e53742))
- Skip empty patches instead of aborting ([1d0caa5](https://github.com/postfinance/topf/commit/1d0caa58cba2216077e3bf0b2207acf3052a0651))
- **bootstrap**: Check etcd service state instead of counting members ([24d8089](https://github.com/postfinance/topf/commit/24d80890c6bc2a8698e60164a9a01ca4c5fa24d0))
- Use go/yaml/v4 in secrets.go ([7e7a5b4](https://github.com/postfinance/topf/commit/7e7a5b4306f752228dc024e08fdb567bd0f2948b))
- Add validation when retrieving secrets ([bbdf93d](https://github.com/postfinance/topf/commit/bbdf93d297323a48859c98a09612fe78f58b7ab2))

### Documentation

- Update readme with combined bootstrap/apply command ([3c44395](https://github.com/postfinance/topf/commit/3c44395706f806e6711a710d35c16d7857d77e68))
- Update README.md ([b7b26fd](https://github.com/postfinance/topf/commit/b7b26fdda90fe3cbb6044d1b799848f622f9455e))
- Add 'Alternatives' section at the end of the readme ([7e5b8cd](https://github.com/postfinance/topf/commit/7e5b8cd31bf9f4d896c0349725244f6fdedfeb72))
- Use git-cliff for changelog ([6a5416d](https://github.com/postfinance/topf/commit/6a5416d0a15499acb910bfdc49039c4f2464733e))

### Miscellaneous

- Fail early if a patch is considered JSON patch ([1ab45ee](https://github.com/postfinance/topf/commit/1ab45eeba0b152e9c7d676617115d9c1259c34fc))
- Move sops decryption function to utils/ ([ed9963a](https://github.com/postfinance/topf/commit/ed9963a5d7d02901312e9163f20b716389fb121e))
- Add node.Attrs() function for simpler logging ([483331d](https://github.com/postfinance/topf/commit/483331d1794130ab1675ce0777a4a0a0a23f15b9))
- Validate config-dir argument ([29b409b](https://github.com/postfinance/topf/commit/29b409bf8cc1fb8b61f012da6675b48e44aade4b))
- Move config-dir to topf.yaml ([3ccdc10](https://github.com/postfinance/topf/commit/3ccdc109b397864535cb2f4a69b60ce71381d812))
- Rename nodes to nodes-filter ([ad90a46](https://github.com/postfinance/topf/commit/ad90a461a554036652461f62ac9aabbb3e310392))
- **bootstrap**: Improve logging ([cb1db05](https://github.com/postfinance/topf/commit/cb1db05c41ee11f2a37a7d467b1021bf2d1376f4))
- Add SPDX license ([861149b](https://github.com/postfinance/topf/commit/861149b39a9bbd0cc34b03cd49d774dfa7668e11))

### Build

- Add version flag ([266ccbd](https://github.com/postfinance/topf/commit/266ccbd973f6869f65953946ae591f7a26acb1e9))
- Dependencies upgrade ([88e9bb8](https://github.com/postfinance/topf/commit/88e9bb8baac47f09b63fb2c2369c0a8977d36a4e))
- Refactor NewTopfRuntime to use a struct ([83c8323](https://github.com/postfinance/topf/commit/83c832305408d23171dbd7dff9fa6495af57816d))
- Add goreleaser workflow ([1343686](https://github.com/postfinance/topf/commit/134368618feb480f49c4896ac2e15d41e51dbd57))
