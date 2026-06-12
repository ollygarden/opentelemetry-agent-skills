# `attributes`: known quirks

## Action order matters

`actions` run strictly in the order listed, and each sees the result of the previous ones. A `delete` before a `from_attribute` copy removes the source first; an `insert` of a default before an `upsert` copy changes what gets copied. Order cheap actions (`delete`, `insert`) before expensive ones (`extract`, `convert`) when possible. Do **not** split related actions across two processor instances and rely on order between them — a temporary key created in `attributes/first` may not exist yet when `attributes/second` runs.

## It does NOT touch resource attributes

This is the most common confusion. `attributes` operates on **span / log-record / metric-datapoint** attributes only. To add, rename, hash, or delete **resource** attributes (e.g. `service.name`, `host.name`, `deployment.environment`), use the **`resource`** processor — it has the same `actions` model but resource scope. Note that include/exclude *matching* can read resource attributes (the `resources` match field), but the actions still only write to non-resource attributes.

## insert vs update vs upsert

- `insert` — writes only if the key is **absent**. Silently no-ops on an existing key. Use for defaults/backfill.
- `update` — writes only if the key is **present**. Silently no-ops on a missing key. Use to rewrite/redact known fields.
- `upsert` — always writes (insert if absent, update if present). The safe choice when you don't care about prior state; also the right tool for debugging "why didn't my action apply" (swap to `upsert` to rule out existence checks).

A frequent bug: using `insert` and seeing nothing change because the key already existed, or `update` and seeing nothing because the key was absent.

## `extract` quirks

- Requires **named** capture groups (`(?P<name>...)`); a pattern with unnamed groups is a config error.
- Only acts on **string** values. A non-string source (or a missing key) is a no-op.
- **Overwrites** existing attributes when a capture-group name collides with an existing key — there is no insert-only mode for `extract`.
- The source `key`'s own value is left unchanged; `extract` only adds the captured fields.

## Regexp performance

`match_type: regexp` filters and `pattern`-based `delete`/`hash` compile and run Go regexes per record. On high-throughput pipelines enable the LRU cache (`regexp.cacheenabled: true`, bound with `cachemaxnumentries`) and keep patterns simple — anchored prefixes (`^foo_.*`) are far cheaper than unanchored, backtracking-heavy expressions. Remember `.` matches any character; use `\.` for a literal dot.

## Metric support caveats (identity conflict)

For metrics, **adding new** data-point attributes is safe, but **modifying or deleting existing** data-point attributes can create an **identity conflict**: a metric's identity includes its data-point attributes, and this processor does **not** re-aggregate. Changing them can split or duplicate time series and corrupt cardinality. If you must change existing metric attributes, re-aggregate downstream (a connector, or the `metricstransform` processor), or fix it at the source. This warning does **not** apply to traces or logs — attributes there can be freely modified. Also note `attributes` does not rename metrics or change metric names — that's `metricstransform`.

## Common config errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| Action silently does nothing | `insert` on an existing key, or `update` on a missing key | Use `upsert`, or check the include/exclude filter isn't blocking the record. |
| Config validation error | More than one of `value` / `from_attribute` / `from_context` set on one action | Use exactly one value source per action. |
| Config validation error | `extract` pattern has unnamed groups | Use `(?P<name>...)` named groups. |
| Convert no-op | Value can't be parsed to the target type (e.g. `"abc"` → int), or key absent | Pre-validate; conversion failures keep the original value (logged at debug). |
| `from_context` empty | Receiver missing `include_metadata: true`, or no authenticator for `auth.*` | Configure the receiver/authenticator (see configuration.md). |
| Wrong-signal field rejected | e.g. `span_names` under a metrics pipeline | Use only the match fields valid for the signal. |

## Stability

All three signals (traces, logs, metrics) are **Beta** — production viable, breaking changes rare. Hashing uses SHA-256 on modern versions; very old collectors used SHA-1, so hashes are not comparable across a version boundary. `default_value` on an action is newer (v0.152.0) — confirm your collector version before relying on it.
