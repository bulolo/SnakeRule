# SnakeRule Server

Go API server and rule evaluation core. The current starter implements a JSON condition DSL, ruleset CRUD, and synchronous evaluation. Rules are held in memory for the MVP, so they are cleared when the process exits.

## Run locally

```bash
cd backend
cp .env.example .env
go run ./server/cmd/main
```

The API listens on `http://localhost:8080`. `GET /healthz` checks process health and `GET /readyz` checks that the HTTP service is ready. This initial version does not require PostgreSQL or Redis.

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

## Package boundaries

- `cmd/main`: process signals and shutdown
- `internal/bootstrap`: dependency wiring and HTTP server settings
- `internal/config`: environment and `.env` configuration
- `internal/domain`: ruleset, rule, condition, action, and decision contracts
- `internal/engine/evaluator`: deterministic condition evaluation
- `internal/repo`: repository contract and in-memory development adapter
- `internal/service`: ruleset validation and decision orchestration
- `internal/transport/http`: JSON API, request limits, health endpoints, and error mapping

See [project architecture](../../../docs/architecture.md) for the planned persistence, publishing, decisions, and worker split.
