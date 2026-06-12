# `k8s_attributes` processor

| | |
|-|-|
| Kind | processor |
| Type | `k8s_attributes` |
| Signals | traces (Beta), metrics (Beta), logs (Beta), profiles (Development) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/k8sattributesprocessor> |
| Rename | `k8sattributes` → `k8s_attributes` in v0.146.0; old name kept as a deprecated alias. |

## Description

Enriches telemetry with Kubernetes metadata. The processor watches the Kubernetes API for pod, namespace, node, and workload events and keeps an in-memory cache of their metadata. For each incoming span, metric, or log record it picks a pod by matching one of the configured **pod-association** sources (by default the pod IP, falling back to the connection IP), then adds the configured pod, namespace, node, and workload attributes (`k8s.pod.name`, `k8s.namespace.name`, `k8s.deployment.name`, …) to the record's **resource**.

Association is the crux: enrichment only happens when a record can be tied back to a cached pod. Records arriving with a pod IP (or over a direct connection from the pod) match out of the box; records that have lost that signal — for example after batching or load balancing strips the connection — need an explicit `pod_association` on a resource attribute the data still carries. Memory scales with the number of pods watched, so node/namespace filtering is the main lever for large clusters. The processor needs access to a **Kubernetes API** — in-cluster via a ServiceAccount, or out-of-cluster via `auth_type: kubeConfig` — since there is no cluster to enrich from otherwise; for non-Kubernetes resource attributes, use `resourcedetection`.

## Main use-cases

Use it when:
- The collector runs in Kubernetes and you want telemetry correlated with pod/namespace/node/workload metadata.
- You want service discovery from pod labels/annotations (e.g. deriving `service.name`).
- You run an agent/gateway split: a `passthrough` agent stamps the pod IP, a gateway does the full enrichment.

Avoid it when:
- The collector runs **outside** Kubernetes — use `resourcedetection` instead.
- The pod already injects its metadata via the Kubernetes downward API (simpler for sidecars).
- The metadata is already on the telemetry — re-deriving it only adds overhead.

## Related components

- `resourcedetection` — detects cloud/host/k8s-node resource attributes; the non-Kubernetes-API counterpart. Run it **before** `k8s_attributes`.
- `resource` — sets/edits/removes resource attributes by static value or OTTL. Run it **after** `k8s_attributes` to rename or drop what was added.
- `transform` — OTTL attribute manipulation; the supported replacement for the removed `regex` value-extraction on labels/annotations (see [Known quirks](quirks.md)).

## Details

- [Configuration](configuration.md) — full config table, the `extract` / `filter` / `pod_association` blocks, default-extracted fields, and the available metadata fields.
- [Verification](verification.md) — end-to-end recipe against a real (kind) cluster: telemetrygen → `k8s_attributes` → `debug` showing pod attributes added.
- [Advanced use-cases](advanced.md) — RBAC, agent/gateway passthrough split, namespace/node filtering, labels & annotations, `deployment_name_from_replicaset`, `wait_for_metadata`.
- [Known quirks](quirks.md) — association failures, connection-based association vs batching, memory, the removed `regex` field, semconv feature gates, stability caveats.
