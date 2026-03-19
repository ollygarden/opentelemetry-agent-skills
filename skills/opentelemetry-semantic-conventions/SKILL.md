---
name: opentelemetry-semantic-conventions
description: OpenTelemetry semantic convention lookup and naming guidance. Use when selecting released semantic convention groups, attributes, or span naming rules, or when checking semantic convention compliance.
---

# OpenTelemetry Semantic Conventions

Use this skill when you need released semantic convention guidance for naming, attributes, or compliance checks.

If the broader task is manual instrumentation design or review, pair this with `opentelemetry-manual-instrumentation`.

## Workflow

1. Start with released semantic conventions, not model memory.
- do not load the full semantic convention spec into context
- use the bundled lookup script to query only the needed released group or attribute

2. Choose the closest released group before inventing custom keys.
- identify the boundary type such as `http`, `db`, `messaging`, `rpc`, `network`, `gen-ai`, or `mcp`
- pick one primary group first, then add related groups only when they add needed context
- see `references/semconv-selection.md`

3. Query only the released guidance you need.
- list groups: `./scripts/query-otel-semantic-conventions.sh --groups`
- inspect one group: `./scripts/query-otel-semantic-conventions.sh http`
- inspect one kind: `./scripts/query-otel-semantic-conventions.sh http spans`
- inspect one exact attribute or entry: `./scripts/query-otel-semantic-conventions.sh http http.request.method`
- see `references/otel-semantic-conventions.md`

4. Apply the released naming and attribute rules directly.
- use required and recommended attributes before optional ones
- derive semconv-governed span names directly from the released naming rule
- do not prepend or append protocol labels, hostnames, product names, business hints, or other custom prose to semconv-governed span names
- if the released naming rule does not provide a low-cardinality target, use the simpler fallback allowed by that convention
- if no released key exists, use a stable custom namespace and keep values bounded

5. Return the result with exact source context.
- include the group name, released version, and source URL from the script output
- call out any concrete compatibility limitation if the implementation cannot fully match the released guidance
