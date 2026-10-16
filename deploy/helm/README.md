# SnakeRule Helm Chart

This chart deploys the SnakeRule API, admin frontend, Nginx reverse proxy, and a single PostgreSQL StatefulSet with a persistent volume. PostgreSQL stores all application state, including organizations, projects, rulesets, users, credentials, alerts, notifications, invocation logs, sessions, license state, and platform branding. Redis is not required. The documentation site remains outside this Kubernetes chart.

The API remains pinned to one replica because repository instances keep process-local caches. PostgreSQL is the durable source of truth. The initial database schema is created by the embedded Goose migration on first startup; persistent application state is stored in entity tables.

## Configuration

Set these values in a private Helm values file:

```yaml
backend:
  env:
    authEmail: superadmin@example.com
    authPassword: "replace-with-a-strong-password"
    licenseEncryptionKey: "64-hex-characters-kept-stable"
    jwtSecret: "replace-with-a-long-random-secret"
database:
  password: "URL-safe-random-database-password"
  storage:
    size: 20Gi
```

The database password is used in the backend connection URL and should use URL-safe characters. If `backend.existingSecret` is set, that Secret must contain `AUTH_PASSWORD`, `LICENSE_ENCRYPTION_KEY`, `JWT_SECRET`, and `POSTGRES_PASSWORD`; include `PULSE_PRODUCT_PUBLIC_KEY` when licensing uploads are enabled.

PostgreSQL state is kept in its own PVC. The API's `/data` PVC is retained for importing older JSON state. Back up the database PVC regularly and keep the license encryption key unchanged across upgrades.

## Install and upgrade

```sh
make helm-lint
make helm-install HELM_VALUES=snakerule-private.yaml
```

`helm-install` runs `helm upgrade --install`, creates the namespace, and waits for workloads to become ready. To use Ingress instead of a LoadBalancer service, set `nginx.service.type: ClusterIP` and configure `ingress.enabled`, `ingress.className`, `ingress.hosts`, and `ingress.tls` in the values file.

To render manifests without applying them:

```sh
make helm-template HELM_VALUES=snakerule-private.yaml
```
