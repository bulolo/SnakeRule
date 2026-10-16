# SnakeRule Server

Go API server and rule evaluation core. The current starter implements a JSON condition DSL, ruleset CRUD, and synchronous evaluation. Rules are held in memory for the MVP, so they are cleared when the process exits.

## Run locally

```bash
make dev-init
make dev-up
```

Open `http://localhost:8080` for the React admin. The Go API is available at `http://localhost:8081`; Nginx routes the unified `8080` entry to the admin and API. `make dev-up` runs Compose in the foreground; stop the stack with Ctrl+C or run `make dev-down` from another terminal. `make dev-status` checks container state and API health. `GET /healthz` checks process health and `GET /readyz` checks that the HTTP service is ready. PostgreSQL and Redis are optional services enabled with `make infra-up`.

## Authentication

The admin signs in by email through `POST /api/v1/auth/login`. A successful login sets an HttpOnly, SameSite cookie; the browser checks it through `GET /api/v1/auth/session`. `POST /api/v1/auth/logout` revokes the session and clears the cookie. Ruleset CRUD, workspace management, and decision evaluation require a valid session. Configure the super administrator with `AUTH_EMAIL`, `AUTH_NAME`, and `AUTH_PASSWORD` in `backend/.env`; `backend/.env.example` contains a non-secret password placeholder. Production mode rejects that placeholder. Sessions and workspace selections are held in memory and expire after `AUTH_SESSION_TTL` (24 hours by default); restarting the API ends existing sessions. Multi-user RBAC is not implemented yet.

## API sketch

Create or replace a ruleset with `POST /api/v1/rulesets`:

```json
{
  "id": "order-risk",
  "name": "Order risk",
  "version": "1.0.0",
  "enabled": true,
  "rules": [
    {
      "id": "large-new-order",
      "name": "Large order from a new customer",
      "priority": 100,
      "enabled": true,
      "conditions": {
        "all": [
          { "fact": "order.amount", "operator": "greaterThan", "value": 10000 },
          { "fact": "customer.isNew", "operator": "equal", "value": true }
        ]
      },
      "actions": [{ "type": "tag", "config": { "label": "REVIEW" } }]
    }
  ]
}
```

Evaluate it with `POST /api/v1/engine/evaluate`:

```json
{
  "rulesetId": "order-risk",
  "facts": { "order": { "amount": 12000 }, "customer": { "isNew": true } }
}
```

Conditions support recursive `all`, `any`, and `not` groups plus fact operators `equal`, `notEqual`, numeric comparisons, `in`, `notIn`, `contains`, `isNull`, and `isNotNull`. Priority is evaluated from highest to lowest; all matching enabled rules are returned. Actions are currently declarative output only and do not trigger side effects.

The service seeds an `order-risk` demo ruleset in the default workspace at startup. The admin provides a workspace switcher, workspace CRUD in Platform Settings, a drag-and-drop rule canvas, JSON advanced editing, and simulated decision execution. Rulesets are isolated by the active workspace but stored in memory; restarting the API clears them and resets workspaces to the default workspace.

## Package boundaries

- `cmd/main`: process signals and shutdown
- `internal/bootstrap`: dependency wiring and HTTP server settings
- `internal/config`: environment and `.env` configuration
- `internal/domain`: ruleset, rule, condition, action, and decision contracts
- `internal/engine/evaluator`: deterministic condition evaluation
- `internal/repo`: repository contract and in-memory development adapter
- `internal/service`: ruleset validation and decision orchestration
- `internal/transport/http`: JSON API, request limits, health endpoints, and error mapping

See [project architecture](../../../docs/architecture.md) for current module boundaries, data lifecycle, deployment limits, and the staged evolution plan.
