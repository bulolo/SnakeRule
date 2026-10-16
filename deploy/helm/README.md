# SnakeRule Helm Chart

This chart deploys the SnakeRule API, admin frontend, and Nginx gateway. It does not create database, Redis, Pulse, or documentation workloads: the current product deployment does not need bundled database/Redis, Pulse Agent is included in the API image, and the documentation site is not part of this chart.

## Prerequisites

- Kubernetes cluster with a default StorageClass (or set `backend.persistence.storageClass`).
- Helm 3.
- SnakeRule backend and admin images available to cluster nodes. Defaults use `snakerule-server:prod` and `snakerule-admin:prod`; build and publish them for your registry, then override the image repositories and tags.
- A values file containing unique secrets. Do not commit that file.

Example `snakerule-private.yaml`:

```yaml
backend:
  image:
    repository: registry.example.com/snakerule-server
    tag: 0.1.0
  env:
    authPassword: "replace-with-a-long-random-password"
    licenseEncryptionKey: "replace-with-a-stable-64-character-hex-key"
    jwtSecret: "replace-with-a-long-random-signing-secret"
admin:
  image:
    repository: registry.example.com/snakerule-admin
    tag: 0.1.0
gateway:
  service:
    type: LoadBalancer
```

`licenseEncryptionKey` must remain stable across restarts/upgrades to read the existing license file. You can set `backend.env.pulseProductPublicKey` if authorization uploads should be verified. Leave it empty when licensing is not configured.

## Install and upgrade

```sh
make helm-lint
make helm-install HELM_VALUES=snakerule-private.yaml
```

`helm-install` uses `helm upgrade --install`, creates the namespace, and waits for workloads to become ready. To use Ingress instead of a LoadBalancer service, set `gateway.service.type: ClusterIP` and configure `ingress.enabled`, `ingress.className`, `ingress.hosts`, and `ingress.tls` in the values file.

To render manifests without applying them:

```sh
make helm-template HELM_VALUES=snakerule-private.yaml
```

Release operations:

```sh
helm history snakerule -n snakerule
helm rollback snakerule -n snakerule
make helm-uninstall
```

Uninstall leaves the backend PVC in place so the license data can be retained. Back up the PVC before cluster maintenance or migration.
