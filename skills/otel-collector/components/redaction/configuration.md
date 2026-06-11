# `redaction`: configuration

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

## Configuration reference

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

## Processing order and precedence

Attributes are evaluated in a fixed order:

1. **Ignored keys** (`ignored_keys`, `ignored_key_patterns`) — always pass through untouched.
2. **Key allow-listing** — with `allowed_keys` set (and `allow_all_keys: false`), keys not on the list are deleted.
3. **Value matching** — surviving attributes have `blocked_values`/`blocked_key_patterns` applied. If a value matches both `blocked_values` and `allowed_values`, **`allowed_values` wins** and the value is kept.
4. **Mask or hash** — marked values become asterisks, or a hash when `hash_function` is set.

Key allow-listing is fail-closed; value matching is not. Relying only on `blocked_values` (with `allow_all_keys: true`) is a much weaker guarantee — value regexes can miss formats they weren't written for.

## Audit attributes

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

For map-formatted log bodies, the same metrics are recorded under a `redaction.body.*` prefix. Key allow-listing and value masking apply to the body too.
