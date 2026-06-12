# `resource`: advanced use-cases

All actions here operate on **resource** attributes and use the same grammar as the [`attributes`](../attributes/README.md) processor. Actions run top-to-bottom; order matters (see [quirks](quirks.md)).

## Normalize `service.name`

Force a canonical service name regardless of what the SDK set, or backfill one when it's missing:

```yaml
processors:
  resource/canonical-name:
    attributes:
      - key: service.name        # always overwrite
        value: checkout
        action: upsert
  resource/default-name:
    attributes:
      - key: service.name        # only if the SDK left it unset
        value: unknown-service
        action: insert
```

## Set `deployment.environment.name`

Tag all telemetry from a collector instance with its environment (semantic conventions renamed `deployment.environment` → `deployment.environment.name`):

```yaml
processors:
  resource:
    attributes:
      - key: deployment.environment.name
        value: production
        action: upsert
      - key: cloud.region
        value: us-east-1
        action: upsert
```

## Drop high-cardinality / sensitive resource keys

Remove resource attributes that inflate cardinality or leak internal data — by exact key or regex:

```yaml
processors:
  resource:
    attributes:
      - key: process.command_line     # noisy, often unbounded
        action: delete
      - pattern: ^internal\..*        # bulk delete by key regex
        action: delete
```

## Hash identifying resource attributes

Anonymize identifiers while keeping them usable as a stable join key (SHA-256, deterministic):

```yaml
processors:
  resource:
    attributes:
      - key: host.id
        action: hash
      - pattern: ^container\.id$
        action: hash
```

## Rename legacy keys with ordered actions

`extract`/`from_attribute` read; `insert`/`upsert` write; `delete`/`hash` clean up. Read before you delete:

```yaml
processors:
  resource:
    attributes:
      # copy legacy key to the semantic-convention key
      - key: k8s.cluster.name
        from_attribute: k8s-cluster
        action: insert
      # parse a region out of the AZ before anything removes it
      - key: cloud.availability_zone
        pattern: ^(?P<cloud_region>[a-z]+-[a-z]+-\d+)[a-z]$
        action: extract
      # only now drop the legacy key
      - key: k8s-cluster
        action: delete
```

## Named instances

Like every component, `resource` supports the `type/name` form so the same type can be configured more than once and selected per pipeline:

```yaml
processors:
  resource/prod-tags:
    attributes:
      - key: deployment.environment.name
        value: production
        action: upsert
  resource/strip-internal:
    attributes:
      - pattern: ^internal\..*
        action: delete

service:
  pipelines:
    traces/prod:
      processors: [resource/prod-tags]
    traces/egress:
      processors: [resource/strip-internal]
```

## Combine with `resourcedetection`

The canonical pattern: `resourcedetection` discovers infrastructure attributes from the environment, then `resource` normalizes, renames, and adds static values. Place `resourcedetection` **first**:

```yaml
processors:
  resourcedetection:
    detectors: [env, system, docker, ec2]
    override: false
  resource:
    attributes:
      - key: deployment.environment.name
        value: production
        action: upsert
      - key: k8s.cluster.name
        from_attribute: k8s-cluster
        action: insert
      - key: k8s-cluster
        action: delete

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resourcedetection, resource]
      exporters: [otlp]
```

## Combine with `k8sattributes`

Let `k8sattributes` provide dynamic pod/namespace/node metadata from the k8s API, and use `resource` only for static cluster-level values that the API can't supply. Don't hard-code pod-level attributes in `resource` — they're per-pod and dynamic:

```yaml
processors:
  k8sattributes:
    extract:
      metadata: [k8s.pod.name, k8s.namespace.name, k8s.node.name]
  resource:
    attributes:
      - key: k8s.cluster.name        # static, not pod-specific
        value: prod-us-east
        action: upsert

service:
  pipelines:
    traces:
      processors: [k8sattributes, resource]
```

## Relationship to `attributes`

`resource` and [`attributes`](../attributes/README.md) share an identical action set (`insert`/`update`/`upsert`/`delete`/`hash`/`extract`/`convert`) and the same `value`/`from_attribute`/`from_context`/`pattern`/`converted_type` fields. The only differences:

- **Scope** — `resource` edits resource attributes; `attributes` edits span/log/datapoint attributes.
- **Matching** — `attributes` has `include`/`exclude` (which can even match on resource attributes); `resource` has none.

If you need to edit resource attributes *only for matching telemetry*, reach for `attributes` (with a `resources:` matcher) or `transform`, not `resource`.
