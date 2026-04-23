---
title: OpenTelemetry Semantic Conventions
read_when: Choosing released semantic convention attributes for instrumentation
---

Do not load the full semantic convention spec into context. Query only the needed released group with `./scripts/query-otel-semantic-conventions.sh`.

Use:
- list groups: `./scripts/query-otel-semantic-conventions.sh --groups`
- group lookup: `./scripts/query-otel-semantic-conventions.sh http`
- exact attribute: `./scripts/query-otel-semantic-conventions.sh http http.request.method`

Rules:
- use `--groups` first if you do not already know the right group
- start with one group
- use the one-argument form first to discover the current released attribute ids and available kinds
- use the two-argument form only when you need the exact upstream definition block for one attribute
- the script resolves the latest released semantic conventions version and reads `model/<group>/registry.yaml` from that tag

Common groups:
`http`, `db`, `messaging`, `rpc`, `network`, `url`, `server`, `error`, `user-agent`, `gen-ai`, `mcp`
