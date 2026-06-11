# `redaction`: known quirks

## `allowed_keys` fails closed — the footgun

With `allow_all_keys: false` (the default) and an empty `allowed_keys`, **every** attribute is dropped — allow-list semantics, not block-list. If attributes vanish unexpectedly, this is almost always why. Either enumerate the keys you want to keep in `allowed_keys`, or set `allow_all_keys: true` to disable key filtering and rely on value masking alone.

## Evaluation order

Attributes are evaluated in a fixed order:

1. **Ignored keys** (`ignored_keys`, `ignored_key_patterns`) — always pass through untouched.
2. **Key allow-listing** — with `allowed_keys` set (and `allow_all_keys: false`), keys not on the list are deleted.
3. **Value matching** — surviving attributes have `blocked_values`/`blocked_key_patterns` applied. If a value matches both `blocked_values` and `allowed_values`, **`allowed_values` wins** and the value is kept.
4. **Mask or hash** — marked values become asterisks, or a hash when `hash_function` is set.

Key allow-listing is fail-closed; value matching is not. Relying only on `blocked_values` (with `allow_all_keys: true`) is a much weaker guarantee — value regexes can miss formats they weren't written for. See [Configuration](configuration.md#processing-order-and-precedence) for the full precedence detail.

## Masking vs. removal

`blocked_key_patterns` masks the **value** of a matching key but keeps the key itself. To drop a key entirely, leave it off `allowed_keys` with key filtering enabled. `blocked_values` masks values, never removes keys.

## Only string values by default

`blocked_values` matches only string values unless `redact_all_types: true` is set, which checks non-string values via their `AsString()` form. Secrets stored in numeric or boolean values slip through otherwise.

## `summary: debug` writes key names into telemetry

`debug` verbosity records the redacted/masked/allowed **key names** as attributes. If a key name is itself sensitive, keep `summary: info` (counts only) in production and reserve `debug` for short-lived investigation.

## DB sanitizer firing conditions

DB sanitization only fires on spans carrying `db.system`/`db.system.name` with `CLIENT` or `SERVER` span kind. Query text on spans without those attributes — or with a different span kind — is not sanitized. Enable per-engine sanitization explicitly (`db_sanitizer.<engine>.enabled: true`) and point it at the attributes holding the query text.

## Per-signal stability

Traces are Beta; logs and metrics are Alpha. The URL sanitizer and DB sanitizer are newer surface area. Verify masking against representative log/metric data before relying on the processor as your only PII control on those signals.
