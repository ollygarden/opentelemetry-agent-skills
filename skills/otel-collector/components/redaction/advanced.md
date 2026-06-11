# `redaction`: advanced use-cases

## Allowlist mode as a strict schema

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

## Hashing for correlatable-but-masked values

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

## URL and DB query sanitization

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

## `blocked_key_patterns` vs `blocked_values`

- `blocked_key_patterns` matches **key names** (e.g. `.*token.*`); when a key matches, its value is masked but the key is kept.
- `blocked_values` matches **value content** (e.g. a card-number regex) regardless of the key name.
- To remove a key entirely rather than mask its value, leave it off `allowed_keys` (with key filtering enabled) instead of using `blocked_key_patterns`.

## The audit summary

When `summary` is not `silent`, the processor records what it did as attributes on each record (zero-count attributes are omitted). See the [audit attributes table](configuration.md#audit-attributes) for the full list of `redaction.*` (and `redaction.body.*`) names and their verbosity levels. Use `summary: info` for counts in production; reserve `summary: debug` (which records key names) for short-lived investigation.
