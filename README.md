# TOPF - Talos Orchestrator by PostFinance

TOPF is managing [Talos](https://www.talos.dev/) based Kubenetes clusters. It provides functionality for bootstrapping new clusters, resetting existing ones, and applying configuration changes.

## Quickstart

Boot at least one Talos machine to maintenance mode.

Create a new folder for you cluster with a `topf.yaml` file:

```yaml
kubernetesVersion: 1.34.1
clusterEndpoint: https://192.168.1.100:6443
clusterName: mycluster

nodes:
- host: node1
  ip: 172.20.10.2
  role: control-plane
```

Create a new patch to specify the install disk and desired talos version:

`patches/01-installation.yaml`:

```yaml
machine:
  install:
    disk: /dev/vda
    image: factory.talos.dev/metal-installer/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba:v1.11.5
```

Then run `topf bootstrap` in that cluster.

Once finished use `topf kubeconfig` to create a Admin kubeconfig for the cluster and use `topf talosconfig` to create a valid talosconfig.


## Patches

In almost all cases you want to apply some talos patches to your cluster. These will go into your cluster folder like so:

```bash
.
├── control-plane
│   ├── 01-vip.yaml
│   ├── 02-disable-discovery.yaml
│   └── 03-allow-cp-scheduling.yaml
├── nodes
│   └── node1
│       └── 01-some-nodespecific-patch.yaml
├── patches
│   └── 01-installation.yaml
└── topf.yaml
```

You can add patches for all nodes (`patches/`), control plane nodes (`control-plane/`) and individual nodes (`nodes/<host>`).

### Templating Patches

If patches end with `yaml.tpl`, you can use go templating in them. There you can use the following fields:

* `.ClusterName`
* `.ClusterEndpoint`
* `.Node.Host`
* `.Node.Role`
* `.Node.IP` (if set)
* `.Node.Data.<key>` (if set)

Example:

```yaml
machine:
  kubelet:
    extraArgs:
      provider-id: {{ .Data.uuid }}
```