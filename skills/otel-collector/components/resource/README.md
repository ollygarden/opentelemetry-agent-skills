# `resource` processor

| | |
|-|-|
| Kind | processor |
| Type | `resource` |
| Signals | traces (Beta), metrics (Beta), logs (Beta), profiles (Development) |
| Distributions | core, contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourceprocessor> |

## Description

Modifies the **resource** attributes of telemetry through an ordered list of `attributes` actions. The resource describes the entity producing the telemetry (a service instance, host, container, …), and its attributes are shared across every span, log record, and metric data point from that resource. Each action names an attribute `key` and one of `insert`, `update`, `upsert`, `delete`, `hash`, `extract`, or `convert`; actions run in the order listed, so later actions see the effect of earlier ones. Values can be literal (`value`), copied from another resource attribute (`from_attribute`), or pulled from request context (`from_context` — client IP, transport metadata, or authenticator data).

It uses the **same action grammar as the [`attributes`](../attributes/README.md) processor**, but scoped to the resource: set `service.name`, add `deployment.environment.name`, rename legacy keys to semantic conventions, or drop noisy resource keys. Unlike `attributes`, it has **no `include`/`exclude` matching** — every resource that flows through the pipeline is processed.

## Main use-cases

Use it when:
- You want to set or standardize resource identity — `service.name`, `service.namespace`, `service.instance.id` — across services (`upsert` static values, `insert` defaults).
- You need to add environment / infrastructure context to all telemetry from a collector (`deployment.environment.name`, `cloud.region`, static tags).
- You want to rename or migrate legacy resource keys to semantic conventions (`insert` from `from_attribute`, then `delete` the old key).
- You need to drop high-cardinality or sensitive resource keys, or `hash` identifying resource attributes (`host.id`, container IDs).

Avoid it when:
- You need to edit **span / log / datapoint** attributes — use [`attributes`](../attributes/README.md) (same action set, telemetry scope).
- You need `include`/`exclude` scoping — `resource` has none; use [`attributes`](../attributes/README.md) (which supports resource-attribute matching) or `transform`.
- You want to auto-discover resource attributes from the environment — use `resourcedetection`.
- You want Kubernetes pod/namespace/node metadata — use `k8sattributes`.
- You need conditional logic, math, or cross-field transforms — use `transform` (OTTL).

## Related components

- [`attributes`](../attributes/README.md) — the **same `insert`/`update`/`upsert`/`delete`/`hash`/`extract`/`convert` actions**, but applied to span/log/datapoint attributes instead of the resource.
- `resourcedetection` — auto-detects resource attributes (cloud, host, container, k8s) from the environment; commonly run **before** `resource`, which then normalizes/renames the detected keys.
- `k8sattributes` — enriches resources with Kubernetes pod/namespace/node metadata from the k8s API; use it for dynamic pod-level enrichment rather than static `resource` values.
- `transform` — OTTL-based mutation that can also edit resource attributes (`context: resource`); a superset of `resource`, at the cost of more verbosity.

## Details

- [Configuration](configuration.md) — the `attributes` action list (every action type and its `key`/`value`/`from_attribute`/`from_context`/`pattern`/`converted_type` fields), scoped to **resource** attributes, and why there is no include/exclude matching.
- [Verification](verification.md) — telemetrygen recipe that sets a resource attribute and confirms it in the **Resource attributes** block of `debug` output.
- [Advanced use-cases](advanced.md) — normalizing `service.name`, setting `deployment.environment.name`, dropping high-cardinality keys, hashing identifiers, ordered actions, named instances, and combining with `resourcedetection`/`k8sattributes`.
- [Known quirks](quirks.md) — operates only on resource attributes (the common `attributes` confusion), no include/exclude, action order, batch-wide effect, `delete`/`hash` semantics, and stability.
