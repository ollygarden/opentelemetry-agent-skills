# `attributes`: configuration

The processor operates on **span / log-record / metric-datapoint attributes** — not resource attributes (use the `resource` processor for those). It has two parts: a required ordered list of `actions`, and an optional `include`/`exclude` matching block that scopes which records the actions apply to.

## Root configuration

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `actions` | list | none | Yes | Ordered list of attribute actions. At least one is required; applied in the order specified. |
| `include` | object | none | No | Match criteria; only matching records are processed. |
| `exclude` | object | none | No | Match criteria; matching records are skipped. When both are set, `include` is evaluated first, then `exclude` filters the included set. |

## Action fields

Each entry in `actions` is one operation:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Conditional | Attribute key to act on. Required for all actions except `delete`/`hash` when only a `pattern` is given. |
| `action` | string | Yes | One of `insert`, `update`, `upsert`, `delete`, `hash`, `extract`, `convert` (case-insensitive). |
| `value` | any | Conditional | Value to set (string/int/double/bool). Supports env-var expansion (`${env:VAR}`); an unset reference falls back to `default_value`. Required for `insert`/`update`/`upsert` unless `from_attribute`, `from_context`, or `default_value` is used. Mutually exclusive with `from_attribute`/`from_context`. |
| `from_attribute` | string | Conditional | Copy the value from another attribute on the same record. Mutually exclusive with `value` and `from_context`. |
| `from_context` | string | Conditional | Pull the value from request context. Mutually exclusive with `value` and `from_attribute`. See [Context values](#context-values). |
| `default_value` | any | No | Fallback value when the `value`/`from_attribute`/`from_context` source is missing (`insert`/`update`/`upsert` only; v0.152.0). Prevents the action from being skipped; ignored by other actions. |
| `pattern` | string | Conditional | Regex (Go syntax). For `delete`/`hash`, matches attribute **keys** to act on. For `extract`, the regex applied to the value (must use named capture groups). |
| `converted_type` | string | Conditional | Target type for `convert`: `int`, `double`, or `string`. Required for `convert`. |

### Action types

| Action | Requires | Behavior |
|--------|----------|----------|
| `insert` | `key` + one of `value`/`from_attribute`/`from_context` | Adds the attribute only if the key does **not** already exist. No-op if the key exists, or if the source is missing and no `default_value` is set. |
| `update` | `key` + one of `value`/`from_attribute`/`from_context` | Replaces the value only if the key **already** exists. No-op if the key is absent, or if the source is missing and no `default_value` is set. |
| `upsert` | `key` + one of `value`/`from_attribute`/`from_context` | Insert if absent, update if present (insert + update). No-op only if the source value is missing and no `default_value` is set. |
| `delete` | `key` and/or `pattern` | Removes the named key and/or every key matching `pattern`. |
| `hash` | `key` and/or `pattern` | Replaces the value(s) of the named key and/or keys matching `pattern` with their SHA-256 hash (hex string). |
| `extract` | `key` + `pattern` (named groups) | Applies the regex to the **string** value of `key` and creates one attribute per named capture group. Source value is left unchanged. Only acts on string values; overwrites existing attributes when group names collide. |
| `convert` | `key` + `converted_type` | Converts the existing value of `key` to `int`/`double`/`string`. No-op if the key is absent; on a failed conversion the original value is kept (logged at debug). |

Example covering several actions:

```yaml
processors:
  attributes:
    actions:
      - key: environment        # default value if absent
        value: production
        action: insert
      - key: db.statement       # redact in place
        value: "[REDACTED]"
        action: update
      - key: user.email         # hash PII
        action: hash
      - pattern: "^temp_.*"     # bulk delete by key regex
        action: delete
      - key: http.status_code   # coerce type
        action: convert
        converted_type: int
```

## Include / exclude matching

`include` and `exclude` share the same structure. Both are optional; if neither is set, every record is processed.

| Field | Type | Description |
|-------|------|-------------|
| `match_type` | string | `strict` (exact, case-sensitive) or `regexp` (Go regexp). Required when `include`/`exclude` is present. |
| `regexp` | object | Regex tuning, only valid with `match_type: regexp`. See [Regexp options](#regexp-options). |
| `services` | list[string] | Match by service name (traces and logs only). |
| `span_names` | list[string] | Match by span name (traces only). |
| `span_kinds` | list[string] | Match by span kind, e.g. `SPAN_KIND_INTERNAL` (traces only). |
| `log_bodies` | list[string] | Match by log body, **string bodies only** (logs only). |
| `log_severity_texts` | list[string] | Match by log severity text (logs only). |
| `log_severity_number` | object | Match by numeric severity (logs only). See below. |
| `metric_names` | list[string] | Match by metric name (metrics only). |
| `attributes` | list[object] | Match by attribute: each entry has `key` and optional `value`. Key-only matches any value. |
| `resources` | list[object] | Match by resource attribute: `key` and optional `value`. |
| `libraries` | list[object] | Match by instrumentation library: `name` and optional `version`. |

`log_severity_number` takes `min` (inclusive minimum severity number — TRACE 1-4, DEBUG 5-8, INFO 9-12, WARN 13-16, ERROR 17-20, FATAL 21-24) and `match_undefined` (when `true`, records with no severity number also match; default `false`).

### Valid fields per signal

- **Traces:** `services`, `span_names`, `span_kinds`, `attributes`, `resources`, `libraries`.
- **Logs:** `log_bodies` (string bodies), `log_severity_texts`, `log_severity_number`, `attributes`, `resources`, `libraries`.
- **Metrics:** `metric_names`, `resources`. (No `services`, `attributes`, `span_*`, or `log_*` fields.)

### Regexp options

When `match_type: regexp`, a nested `regexp` block can tune an LRU cache of match results for high-throughput pipelines:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `cacheenabled` | bool | `false` | Cache compiled-regex match results. |
| `cachemaxnumentries` | int | unlimited | Cap on cache entries when caching is enabled. |

```yaml
include:
  match_type: regexp
  regexp:
    cacheenabled: true
    cachemaxnumentries: 1000
  services: ["^auth-.*", "^payment-.*"]
```

## Context values

`from_context` pulls a value from request context. Three source kinds:

- **`client.address`** — the client IP (plain key, no prefix).
- **`metadata.*`** — gRPC metadata / HTTP headers, e.g. `metadata.x-request-id`. Requires the receiver to set `include_metadata: true`.
- **`auth.*`** — data from a server authenticator extension, e.g. `auth.subject`. Requires an authenticator on the receiver; available keys depend on the authenticator.

If a context key has multiple values, they are joined with `;`.

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        include_metadata: true   # required for metadata.* keys

processors:
  attributes/context:
    actions:
      - key: client.ip
        from_context: client.address
        action: insert
      - key: request_id
        from_context: metadata.x-request-id
        action: insert
      - key: enduser.id
        from_context: auth.subject
        action: insert
```
