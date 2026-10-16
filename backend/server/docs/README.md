# SnakeRule Server

Go HTTP API and deterministic rules engine. The admin client and public callers use the OpenAPI contract in this directory.

## Local development

```bash
make dev-init
make dev-up
```

Open `http://localhost:8080` for the admin console. Nginx routes requests to the Go API and serves the frontend. PostgreSQL is required; Redis is optional. Use `make dev-down` to stop the stack.

The API exposes `/healthz` and `/readyz` for health checks. Service metadata is at `/api/v1/meta`.

## API contract

The API is exposed under `/api`. Successful JSON responses use `{ "code": 0, "msg": "success", "data": ... }`. Errors use a nonzero HTTP status and `{ "code": 1, "msg": "...", "error_code": "..." }`. List responses contain `list` and `pagination`; `is_pager=0` returns the full list.

The API uses snake_case JSON fields. Management requests use the `snakerule_session` HttpOnly cookie. Rule evaluation also accepts project API Keys in `X-API-Key` and JWTs in `Authorization: Bearer`.

See [openapi.json](./openapi.json) for paths, schemas, authentication, pagination, and error responses. Regenerate the TypeScript SDK with `make gen-sdk`.

## Persistence and migrations

PostgreSQL stores organizations, users, projects, memberships, rulesets, credentials, alert channels, organization-scoped notifications, invocation logs, sessions, platform settings, and license state. Goose migrations live in `internal/repo/migrations` and run at service startup. Migration versions are recorded in `snakerule_goose_db_version`.

The repository currently keeps a process-local cache and writes changes to PostgreSQL transactionally. Run a single API replica. The initial schema is defined in `00001_initial_schema.sql`; later releases add ordered migrations without rewriting applied migrations.

## Source layout

- `cmd/main`: process entry point.
- `internal/bootstrap`: configuration, database startup, dependency wiring, and HTTP server setup.
- `internal/config`: environment configuration.
- `internal/domain`: domain types and repository contracts.
- `internal/engine/evaluator`: deterministic condition validation and evaluation.
- `internal/repo`: persistence adapters and Goose migrations.
- `internal/service`: business validation and rule orchestration.
- `internal/transport/http`: API routes, authentication, response envelope, and OpenAPI-facing handlers.
- `internal/licensing`: Pulse authorization verification and encrypted license storage.

See [project architecture](../../../docs/architecture.md) for service boundaries and deployment details.

## Code and schema conventions

- Keep HTTP routing in `transport/http/routes.go`; place handlers by resource in a matching file such as `organization.go`, `project.go`, `rulesets.go`, or `alert_channels.go`.
- Keep domain types and repository contracts in `domain`; put business validation in `service`; keep storage and SQL migration details in `repo`.
- Use Go `MixedCaps` for exported identifiers and descriptive receiver names. Repository implementations use `Persistent<Resource>Repository`; constructors use `NewPersistent<Resource>Repository`.
- Use plural `snake_case` PostgreSQL table names and `snake_case` columns. Use JSON `snake_case` fields and resource-oriented, plural, kebab-case API paths under `/api`.
- All JSON API successes use the shared envelope. Collection endpoints use `list` and `pagination` with `is_pager`, `page`, and `size`.
- Add database changes as ordered Goose migrations in `internal/repo/migrations`; the initial release defines the complete schema in `00001_initial_schema.sql`.
