# `resource`: configuration

The processor operates on **resource attributes only** — the attributes attached to the `Resource` that describes the entity producing the telemetry. It does **not** touch span, log-record, or metric-datapoint attributes (use the [`attributes`](../attributes/README.md) processor for those). Its only field is a required ordered list of `attributes` actions; there is **no `include`/`exclude` matching block** (unlike `attributes`), so every resource flowing through the pipeline is processed.

## Root configuration

| Field | Type | Default | Required | Description |
|-------|------|---------|----------|-------------|
| `attributes` | list | none | Yes | Ordered list of actions to perform on the resource's attributes. At least one is required (an empty list is invalid); applied in the order specified. |

There is no default configuration — the processor must be explicitly configured with at least one action.

```yaml
processors:
  resource:
    attributes:
      - key: service.name
        value: my-service
        action: upsert
```

## Action fields

Each entry in `attributes` is one operation. The fields are identical to the `attributes` processor's action fields, but `key`/`from_attribute` refer to **resource** attributes.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Conditional | Resource attribute key to act on. Required for all actions except `delete`/`hash` when only a `pattern` is given. |
| `action` | string | Yes | One of `insert`, `update`, `upsert`, `delete`, `hash`, `extract`, `convert` (case-insensitive). |
| `value` | any | Conditional | Literal value to set (string/int/double/bool). Required for `insert`/`update`/`upsert` unless `from_attribute` or `from_context` is used. Mutually exclusive with `from_attribute`/`from_context`. |
| `from_attribute` | string | Conditional | Copy the value from another **resource** attribute. Mutually exclusive with `value` and `from_context`. |
| `from_context` | string | Conditional | Pull the value from request context. Mutually exclusive with `value` and `from_attribute`. See [Context values](#context-values). |
| `default_value` | any | No | Fallback value when the `from_attribute`/`from_context` source is missing (v0.152.0). Prevents the action from being skipped. |
| `pattern` | string | Conditional | Regex (Go syntax). For `delete`/`hash`, matches attribute **keys** to act on. For `extract`, the regex applied to the value (must use named capture groups). |
| `converted_type` | string | Conditional | Target type for `convert`: `int`, `double`, or `string`. Required for `convert`. |

Only **one** of `value`, `from_attribute`, or `from_context` may be set per action; setting more than one fails validation.

### Action types

| Action | Requires | Behavior |
|--------|----------|----------|
| `insert` | `key` + one of `value`/`from_attribute`/`from_context` | Adds the resource attribute only if the key does **not** already exist. No-op if the key exists or the source is missing. |
| `update` | `key` + one of `value`/`from_attribute`/`from_context` | Replaces the value only if the key **already** exists. No-op if the key is absent or the source is missing. |
| `upsert` | `key` + one of `value`/`from_attribute`/`from_context` | Insert if absent, update if present (insert + update). No-op only if the source value is missing. The most common action for static configuration. |
| `delete` | `key` and/or `pattern` | Removes the named key and/or every resource key matching `pattern`. |
| `hash` | `key` and/or `pattern` | Replaces the value(s) of the named key and/or keys matching `pattern` with their SHA-256 hash (lowercase hex string). |
| `extract` | `key` + `pattern` (named groups) | Applies the regex to the **string** value of `key` and creates one resource attribute per named capture group. Source value is left unchanged. Only acts on string values; overwrites existing attributes when group names collide. All groups must be named (`(?P<name>…)`). |
| `convert` | `key` + `converted_type` | Converts the existing value of `key` to `int`/`double`/`string`. No-op if the key is absent; on a failed conversion the original value is kept. |

Example covering several actions, all scoped to the resource:

```yaml
processors:
  resource:
    attributes:
      - key: deployment.environment.name   # always set
        value: production
        action: upsert
      - key: k8s.cluster.name              # rename legacy key
        from_attribute: k8s-cluster
        action: insert
      - key: k8s-cluster                    # drop the old key
        action: delete
      - key: host.id                        # anonymize identifier
        action: hash
      - pattern: ^internal\..*              # bulk delete by key regex
        action: delete
      - key: service.port                   # coerce type
        action: convert
        converted_type: int
```

## No include / exclude matching

Unlike the [`attributes`](../attributes/README.md) processor, `resource` has **no `include`/`exclude` block**. There is no way to scope its actions to a subset of resources by service, attribute, or any other matcher — every resource that passes through the pipeline is processed. To apply resource-attribute edits only to matching telemetry, use the `attributes` processor (its match block supports `resources:` matchers) or `transform` (OTTL conditions).

## Context values

`from_context` pulls a value from request context into a resource attribute. Three source kinds:

- **`client.address`** — the client IP (plain key, no prefix). Only available for network receivers (OTLP, Jaeger, …), not file/push ingestion.
- **`metadata.*`** — gRPC metadata / HTTP headers, e.g. `metadata.x-trace-id`. Requires the receiver to set `include_metadata: true`. Multiple values are joined with `;`.
- **`auth.*`** — data from a server authenticator extension, e.g. `auth.username`, `auth.tenant_id`. Requires an authenticator on the receiver; available keys depend on the authenticator. If the `auth.` key doesn't match an auth attribute, the processor falls back to checking metadata with the full key.

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        include_metadata: true   # required for metadata.* keys

processors:
  resource/context:
    attributes:
      - key: client.ip
        from_context: client.address
        action: insert
      - key: tenant.id
        from_context: auth.tenant_id
        action: insert
```
