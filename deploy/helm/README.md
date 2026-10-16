# SnakeRule Helm Chart

This chart deploys the SnakeRule API, admin frontend, and Nginx reverse proxy. It does not create database or Redis workloads because rule and invocation repositories are in memory; backend replicas are therefore pinned to one. Pulse Agent is included in the API image. The documentation site remains outside this Kubernetes chart.

## Prerequisites

- Kubernetes cluster with a default StorageClass (or set `backend.persistence.storageClass`).
- Helm 3.
- SnakeRule backend and admin images available to cluster nodes. Defaults use `snakerule-server:prod` and `snakerule-admin:prod`; build and publish them for your registry, then override the image repositories and tags.
- Unique secrets, provided through a Kubernetes Secret or a private values file. Do not commit either file.

Recommended: create the runtime Secret separately so the credentials are not stored in Helm release values:

```sh
kubectl create namespace snakerule --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret generic snakerule-runtime -n snakerule \
  --from-literal=AUTH_PASSWORD='use-a-long-random-password' \
  --from-literal=LICENSE_ENCRYPTION_KEY='64-character-stable-hex-key' \
  --from-literal=JWT_SECRET='use-a-long-random-signing-secret'
```

The Secret must contain `AUTH_PASSWORD`, `LICENSE_ENCRYPTION_KEY`, and `JWT_SECRET`. Point the chart at it in `snakerule-private.yaml`:

```yaml
backend:
  existingSecret: snakerule-runtime
  # Change this value after rotating the external Secret to restart the API Pod.
  secretChecksum: "rotation-1"
  image:
    repository: registry.example.com/snakerule-server
    tag: 0.1.0
admin:
  image:
    repository: registry.example.com/snakerule-admin
    tag: 0.1.0
```

Alternatively, omit `existingSecret` and provide secret values in a private values file (Helm stores those values in its release record):

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
nginx:
  service:
    type: LoadBalancer
```

`licenseEncryptionKey` must remain stable across restarts/upgrades to read the existing license file. You can set `backend.env.pulseProductPublicKey` if authorization uploads should be verified; with `existingSecret`, put it in the Secret under `PULSE_PRODUCT_PUBLIC_KEY` and set `backend.env.pulseProductPublicKeyFromSecret: true`. Leave it unset when licensing is not configured. Changing values-managed secrets or the Nginx config automatically rolls the corresponding Pods.

## Install and upgrade

```sh
make helm-lint
make helm-install HELM_VALUES=snakerule-private.yaml
```

`helm-install` uses `helm upgrade --install`, creates the namespace, and waits for workloads to become ready. To use Ingress instead of a LoadBalancer service, set `nginx.service.type: ClusterIP` and configure `ingress.enabled`, `ingress.className`, `ingress.hosts`, and `ingress.tls` in the values file.

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
