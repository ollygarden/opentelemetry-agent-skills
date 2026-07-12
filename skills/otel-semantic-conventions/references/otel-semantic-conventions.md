---
title: OpenTelemetry Semantic Conventions
read_when: Choosing released semantic convention attributes for instrumentation
---

Do not load the full semantic convention spec into context. Query only the needed released group with `./scripts/query-otel-semantic-conventions.sh`.

Use:
- list groups: `./scripts/query-otel-semantic-conventions.sh --groups`
- group lookup: `./scripts/query-otel-semantic-conventions.sh http`
- exact attribute: `./scripts/query-otel-semantic-conventions.sh http http.request.method`
- local checkout mode: `OTEL_SEMCONV_REPO=/path/to/semantic-conventions ./scripts/query-otel-semantic-conventions.sh http`

Rules:
- use `--groups` first if you do not already know the right group
- start with one group
- use the one-argument form first to discover the current released attribute ids and available kinds
- use the two-argument form only when you need the exact upstream definition block for one attribute
- the script resolves the latest released semantic conventions version and reads released YAML model files under `model/<group>/` from that tag
- deprecated model files and directories are excluded from lookup results
- the core semantic-conventions repository added `model/manifest.yaml` in v1.43.0; use it as release metadata, not as a convention group
- `gen_ai.*`, OpenAI, and MCP conventions moved to the dedicated OpenTelemetry GenAI semantic conventions repository in v1.42.0; do not treat deprecated stubs under the core repository as current guidance

Common groups:
`http`, `db`, `messaging`, `rpc`, `network`, `url`, `server`, `error`, `user-agent`, `service`, `cloud`, `k8s`, `process`, `otel`, `log`, `exceptions`
