# `resource`: known quirks

## Operates only on resource attributes

The single most common confusion: `resource` touches **only resource attributes** — the attributes on the `Resource` that describes the producing entity. It does **not** see span, log-record, or metric-datapoint attributes. A config like this silently does nothing useful, because `http.status_code` is a span attribute, not a resource attribute:

```yaml
# WRONG: http.status_code lives on the span, not the resource
resource:
  attributes:
    - key: http.status_code
      action: delete
```

For span/log/datapoint attributes use the [`attributes`](../attributes/README.md) processor — it has the exact same action grammar. `resource` is the resource-scoped twin of `attributes`.

## No include / exclude matching

`resource` has **no `include`/`exclude` block**. Every resource flowing through the pipeline is processed — there's no way to scope its actions to a subset of resources. If you need conditional application (e.g. only rename a key for one service), use the `attributes` processor (its match block supports `resources:` matchers) or `transform` with an OTTL condition. Attempting to add an `include:` key to a `resource` config is a configuration error.

## Changes are wholesale across the batch

A `Resource` is shared by every span / log record / metric data point grouped under it in a batch. An action on a resource attribute therefore affects **all** of those records at once — there's no per-record granularity. In the `debug` exporter the attribute appears once, in the **Resource attributes** block, not repeated per span. This is what makes `resource` cheap (one edit per resource, not per record) but also means you can't vary the value by record.

## Action order matters

Actions run top-to-bottom and later actions see the effect of earlier ones. The classic mistake is deleting a key before reading it:

```yaml
# WRONG: source is gone before extract runs
- key: cloud.availability_zone
  action: delete
- key: cloud.availability_zone
  pattern: ^(?P<cloud_region>.+)[a-z]$
  action: extract   # no-op, key already deleted

# RIGHT: read first, clean up last
- key: cloud.availability_zone
  pattern: ^(?P<cloud_region>.+)[a-z]$
  action: extract
- key: cloud.availability_zone
  action: delete
```

Rule of thumb: order as `extract`/read → `insert`/`update`/`upsert`/write → `delete`/`hash`/cleanup.

## `delete` and `hash` semantics

- `delete` and `hash` accept `key` and/or `pattern` (Go regex against attribute **keys**). With both set, the named key plus all pattern matches are affected.
- `hash` replaces the value with its **SHA-256** lowercase-hex digest. It's deterministic (same input → same output) and irreversible; add salt at instrumentation time if you need stronger anonymization. Hashing has a higher per-attribute CPU cost than the simple write actions.
- A `pattern` that matches nothing is a silent no-op — not an error.

## `insert` vs `update` vs `upsert`

- `insert` only writes if the key is **absent** — using it to "change" an existing value silently does nothing.
- `update` only writes if the key **already exists** — using it to "set" a missing value silently does nothing.
- `upsert` always writes (the usual choice for guaranteeing a value).

If an attribute "won't change," check you're not using `insert` on an existing key or `update` on a missing one.

## `extract` only works on string values

`extract` applies its regex to the **string** value of `key`; non-string values are skipped. All capture groups must be **named** (`(?P<name>…)`) — anonymous `()` groups are a config error. Group names become new resource attribute keys, overwriting any existing attribute of the same name.

## `convert` failures are silent

`convert` coerces a value to `int`/`double`/`string`. A failed conversion (e.g. `"abc"` → `int`) keeps the **original** value rather than erroring; the failure is only visible at debug log level. Make sure source values are actually convertible.

## One value source per action

Exactly one of `value`, `from_attribute`, or `from_context` may be set per action. Setting more than one fails validation at startup (`error creating AttrProc due to multiple value sources being set`).

## `from_context` depends on transport

`from_context: metadata.*` requires the receiver to set `include_metadata: true`; `client.address` is only populated for network receivers (OTLP, Jaeger), not file/push ingestion; `auth.*` requires a server authenticator extension. Missing context is a silent no-op — the attribute simply isn't created (unless `default_value` is set).

## Stability

Beta for traces, metrics, and logs; **Development** for profiles. The Beta signals are production-viable; treat profiles support as experimental and subject to breaking changes. Confirm the current stability in the upstream README before relying on it for a given signal.
