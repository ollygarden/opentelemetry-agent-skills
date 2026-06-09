# `redaction` processor

| | |
|-|-|
| Kind | processor |
| Signals | traces, logs, metrics |
| Stability | Beta (traces); Alpha (logs, metrics) |
| Distributions | contrib, k8s |
| `type` | `redaction` |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/redactionprocessor> |

The trace path is Beta (production viable, breaking changes rare). The log and metric paths are Alpha — configuration and behavior may change between releases. The URL sanitizer and DB sanitizer are newer surface area; verify masking against representative data before relying on this processor as your only PII control.

## Description

An allow/block-list redaction processor that enforces a data-governance policy on the attributes flowing through a pipeline. It masks or removes sensitive attribute keys and values across span, log, and metric data points, and can additionally sanitize URLs and database query strings. Two mechanisms compose:

1. **Allow-listing keys** — with `allowed_keys` set, any attribute key *not* on the list is deleted outright. This **fails closed**: an empty `allowed_keys` removes *all* attributes. Set `allow_all_keys: true` to keep every key and rely solely on value masking.
2. **Masking values** — attribute values matching a `blocked_values` regex (or whose key matches a `blocked_key_patterns` regex) are masked with asterisks or replaced by a hash, even when the key itself is allowed.

On top of attribute redaction it can **sanitize URLs** (strip high-cardinality path segments and query content) and **sanitize database queries** (SQL, Redis, Memcached, MongoDB, OpenSearch, Elasticsearch) so query text and span names don't leak literal values.

The processor **modifies in place** — it never drops or reroutes telemetry. It only deletes/masks attributes and rewrites specific string values. Pair it with a downstream `filter` or routing component if you also need to drop whole records. For map-formatted log bodies, key allow-listing and value masking apply to the body too, with a separate `redaction.body.*` audit prefix.

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

## Typical config

```yaml
processors:
  redaction:
    allow_all_keys: false
    allowed_keys:
      - http.method
      - http.route
      - http.status_code
    blocked_values:
      - '4[0-9]{12}(?:[0-9]{3})?'   # Visa-like card numbers
    summary: info

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, redaction]
      exporters: [otlp]
```

### Configuration reference

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `allow_all_keys` | bool | `false` | When `true`, the `allowed_keys` list is disabled and every key is kept. `blocked_values`/`blocked_key_patterns` still apply. |
| `allowed_keys` | []string | `[]` | Keys to retain. **Fails closed** — an empty list removes *all* attributes. Any key not listed is deleted. |
| `ignored_keys` | []string | `[]` | Keys that bypass all redaction checks (never removed, never masked). |
| `ignored_key_patterns` | []string | `[]` | Regex patterns for keys to bypass all redaction. |
| `blocked_key_patterns` | []string | `[]` | Regex patterns; matching keys have their **values masked** (the key is kept). |
| `blocked_values` | []string | `[]` | Regex patterns matched against attribute **values**; matches are masked or hashed. |
| `allowed_values` | []string | `[]` | Regex patterns for values considered safe. **Takes precedence** over `blocked_values` when both match. |
| `redact_all_types` | bool | `false` | When `true`, non-string values are checked via their `AsString()` form against `blocked_values`. When `false`, only string values are checked. |
| `hash_function` | string | `""` | Replace masked values with a hash instead of asterisks. One of `md5`, `sha1`, `sha3`, `hmac-sha256`, `hmac-sha512`. Empty = asterisk masking. |
| `hmac_key` | string | `""` | Secret key for `hmac-sha256` / `hmac-sha512`. Supports env expansion (`${env:REDACTION_HMAC_KEY}`). |
| `summary` | string | `info` | Audit verbosity: `debug` (key names + counts), `info` (counts only), `silent` (no audit attributes). |
| `url_sanitizer.enabled` | bool | `false` | Enable URL sanitization. |
| `url_sanitizer.attributes` | []string | `[]` | Attribute keys whose URL values are sanitized. |
| `url_sanitizer.sanitize_span_name` | bool | `true` | Sanitize span names containing `/`. |
| `db_sanitizer.sanitize_span_name` | bool | `true` | Sanitize database query span names. |
| `db_sanitizer.<engine>.enabled` | bool | `false` | Enable per-engine query sanitization. `<engine>` is one of `sql`, `redis`, `memcached`, `mongo`, `opensearch`, `es`. |
| `db_sanitizer.<engine>.attributes` | []string | `[]` | Attributes holding the query/command text for that engine. |

## Verification

`redaction` ships in the `contrib` and `k8s` distributions.

Config (`redaction-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  redaction:
    allow_all_keys: true
    blocked_values:
      - '4[0-9]{12}(?:[0-9]{3})?'   # Visa-like card numbers
    summary: info
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [redaction]
      exporters: [debug]
```

Generate spans carrying an attribute value that matches the blocked pattern. The span-level attribute flag is `--telemetry-attributes` (resource-level `--otlp-attributes` would attach to the Resource, not the span); the value must be quoted (`key="value"`):

```bash
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 \
  --traces 5 --telemetry-attributes 'cc_number="4111111111111111"'
```

**What proves it worked:** the `debug` exporter shows the matching value masked (replaced with asterisks, or hashed if `hash_function` is set) while non-matching attributes pass through; with `summary: info` the span carries a `redaction.masked.count` audit attribute (and, because nothing is removed under `allow_all_keys: true`, an `redaction.allowed.count`).

## Advanced use-cases

### Allowlist mode as a strict schema

Set `allow_all_keys: false` and enumerate `allowed_keys` to define exactly which attribute keys may leave the boundary. Any key not on the list — including newly added, unanticipated attributes — is dropped by default. This is the strongest egress guarantee the processor offers.

```yaml
processors:
  redaction:
    allow_all_keys: false
    allowed_keys:
      - http.method
      - http.route
      - http.status_code
      - service.version
    ignored_keys:
      - safe.attribute        # bypasses all checks
```

### Hashing for correlatable-but-masked values

By default a masked value becomes a run of asterisks. Set `hash_function` when you need to **correlate** redacted values without revealing them — e.g. confirm two spans carry the same user id without storing the id. Use `hmac-sha256`/`hmac-sha512` with `hmac_key` so the hash is not a plain digest an attacker could rainbow-table.

```yaml
processors:
  redaction:
    allow_all_keys: true
    blocked_values:
      - '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}'   # email
    hash_function: hmac-sha256
    hmac_key: ${env:REDACTION_HMAC_KEY}
```

### URL and DB query sanitization

URL sanitization strips high-cardinality path segments (UUIDs, timestamps, numeric IDs) and query content from URL-valued attributes, preserving routing structure for grouping. DB sanitization rewrites query/command text (and optionally span names) so literals don't leak; it only fires on spans carrying `db.system`/`db.system.name` with `CLIENT` or `SERVER` span kind.

```yaml
processors:
  redaction:
    allow_all_keys: true
    url_sanitizer:
      enabled: true
      attributes: [url.full, http.url]
      sanitize_span_name: true
    db_sanitizer:
      sanitize_span_name: true
      sql:
        enabled: true
        attributes: [db.statement, db.query.text]
      redis:
        enabled: true
        attributes: [db.statement]
```

### `blocked_key_patterns` vs `blocked_values`

- `blocked_key_patterns` matches **key names** (e.g. `.*token.*`); when a key matches, its value is masked but the key is kept.
- `blocked_values` matches **value content** (e.g. a card-number regex) regardless of the key name.
- To remove a key entirely rather than mask its value, leave it off `allowed_keys` (with key filtering enabled) instead of using `blocked_key_patterns`.

### The audit summary

When `summary` is not `silent`, the processor records what it did as attributes on each record (zero-count attributes are omitted):

| Attribute | Verbosity | Description |
|-----------|-----------|-------------|
| `redaction.redacted.keys` | debug | Names of keys removed by allow-listing |
| `redaction.redacted.count` | info | Count of removed keys |
| `redaction.masked.keys` | debug | Names of keys whose values were masked |
| `redaction.masked.count` | info | Count of masked values |
| `redaction.allowed.keys` | debug | Names of keys that passed through |
| `redaction.allowed.count` | info | Count of allowed attributes |
| `redaction.ignored.count` | info | Count of ignored attributes |

For map-formatted log bodies, the same metrics are recorded under a `redaction.body.*` prefix.

## Known quirks

### `allowed_keys` fails closed — the footgun

With `allow_all_keys: false` (the default) and an empty `allowed_keys`, **every** attribute is dropped — allow-list semantics, not block-list. If attributes vanish unexpectedly, this is almost always why. Either enumerate the keys you want to keep in `allowed_keys`, or set `allow_all_keys: true` to disable key filtering and rely on value masking alone.

### Evaluation order

Attributes are evaluated in a fixed order:

1. **Ignored keys** (`ignored_keys`, `ignored_key_patterns`) — always pass through untouched.
2. **Key allow-listing** — with `allowed_keys` set (and `allow_all_keys: false`), keys not on the list are deleted.
3. **Value matching** — surviving attributes have `blocked_values`/`blocked_key_patterns` applied. If a value matches both `blocked_values` and `allowed_values`, **`allowed_values` wins** and the value is kept.
4. **Mask or hash** — marked values become asterisks, or a hash when `hash_function` is set.

Key allow-listing is fail-closed; value matching is not. Relying only on `blocked_values` (with `allow_all_keys: true`) is a much weaker guarantee — value regexes can miss formats they weren't written for.

### Masking vs. removal

`blocked_key_patterns` masks the **value** of a matching key but keeps the key itself. To drop a key entirely, leave it off `allowed_keys` with key filtering enabled. `blocked_values` masks values, never removes keys.

### Only string values by default

`blocked_values` matches only string values unless `redact_all_types: true` is set, which checks non-string values via their `AsString()` form. Secrets stored in numeric or boolean values slip through otherwise.

### `summary: debug` writes key names into telemetry

`debug` verbosity records the redacted/masked/allowed **key names** as attributes. If a key name is itself sensitive, keep `summary: info` (counts only) in production and reserve `debug` for short-lived investigation.

### Per-signal stability

Traces are Beta; logs and metrics are Alpha. The URL sanitizer and DB sanitizer are newer surface area. Verify masking against representative log/metric data before relying on the processor as your only PII control on those signals.

## Related components

- `attributes` — general add/update/delete/hash of attributes (no fail-closed allow list).
- `transform` — OTTL-based conditional transformation, including `replace_pattern` masking (does not fail closed).
- `filter` — drops entire records (redaction never drops).
- `resource` — manipulate resource-level attributes.
