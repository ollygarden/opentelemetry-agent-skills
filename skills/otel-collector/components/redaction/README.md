# `redaction` processor

| | |
|-|-|
| Kind | processor |
| Type | `redaction` |
| Signals | traces, logs, metrics |
| Stability | Beta (traces); Alpha (logs, metrics) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/redactionprocessor> |

The trace path is Beta (production viable, breaking changes rare). The log and metric paths are Alpha — configuration and behavior may change between releases. The URL sanitizer and DB sanitizer are newer surface area; verify masking against representative data before relying on this processor as your only PII control.

## Description

An allow/block-list redaction processor that enforces a data-governance policy on the attributes flowing through a pipeline. It masks or removes sensitive attribute keys and values across span, log, and metric data points, and can additionally sanitize URLs and database query strings. Two mechanisms compose: allow-listing keys (any key not on `allowed_keys` is deleted, failing closed) and masking values (values matching `blocked_values`, or whose key matches `blocked_key_patterns`, are masked with asterisks or replaced by a hash). On top of attribute redaction it can sanitize URLs (strip high-cardinality path segments and query content) and database queries (SQL, Redis, Valkey, Memcached, MongoDB, OpenSearch, Elasticsearch). It **modifies in place** — it never drops or reroutes telemetry.

## Main use-cases

Use it when:
- Only an explicit allow list of attribute keys is permitted to leave a trust boundary (compliance, multi-tenant egress).
- Known-sensitive values (credit cards, emails, tokens) appear in attribute values and must be masked or hashed before export.
- URLs or DB query strings carry high-cardinality or sensitive content (IDs, tokens, literals) you want sanitized.
- You want an audit trail of what was redacted, masked, or allowed through.

Avoid it when:
- You need general-purpose attribute editing (add/upsert/convert) — use `attributes` or `transform`.
- You want to drop entire spans/logs/metrics — use `filter` (redaction never drops).
- Your redaction logic needs arbitrary conditional transformation — OTTL in `transform` is more expressive (but does not fail closed).

## Related components

- `attributes` — general add/update/delete/hash of attributes (no fail-closed allow list).
- `transform` — OTTL-based conditional transformation, including `replace_pattern` masking (does not fail closed).
- `filter` — drops entire records (redaction never drops).
- `resource` — manipulate resource-level attributes.

## Details

- [Configuration](configuration.md) — open when you need config keys, defaults, processing order/precedence, or the audit-attribute names.
- [Verification](verification.md) — open to confirm masking works with a telemetrygen recipe.
- [Advanced use-cases](advanced.md) — open for allowlist-as-schema, hashing for correlation, URL/DB sanitization, and audit summary usage.
- [Known quirks](quirks.md) — open when attributes vanish unexpectedly, masking behaves differently than expected, or per-signal stability matters.
