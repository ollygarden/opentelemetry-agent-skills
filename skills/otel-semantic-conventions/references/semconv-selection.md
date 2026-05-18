# Semantic Convention Selection

Pick the closest released semantic convention group before inventing custom keys.

## Common Starting Groups

- `http` for HTTP client and server operations
- `db` for database operations
- `messaging` for queues, streams, and pub-sub
- `rpc` for remote procedure calls
- `network` for transport-level details
- `url` for URL components
- `server` for server endpoint metadata
- `error` for error classification
- `user-agent` for client user-agent details
- `gen-ai` for model and generation operations
- `mcp` for Model Context Protocol operations

## Selection Rules

1. Start with one primary group that matches the boundary.
2. Add related groups only when they add needed context.
3. Prefer required and recommended attributes before optional ones.
4. If no released key exists, use a stable custom namespace and keep values bounded.

## Lookup Workflow

Use the bundled script instead of loading large spec files:

1. `./scripts/query-otel-semantic-conventions.sh --groups`
2. `./scripts/query-otel-semantic-conventions.sh <group>`
3. `./scripts/query-otel-semantic-conventions.sh <group> <attribute-id>`

Use the one-argument form first. Use the two-argument form only when you need the exact released upstream definition for a single attribute.

## Typical Group Pairings

- HTTP server: `http`, then maybe `url`, `server`, `network`, `user-agent`, `error`
- HTTP client: `http`, then maybe `url`, `server`, `network`, `error`
- Database: `db`, then maybe `server`, `error`
- Messaging: `messaging`, then maybe `network`, `server`, `error`
- RPC: `rpc`, then maybe `server`, `network`, `error`
- GenAI: `gen-ai`, then maybe `error`
