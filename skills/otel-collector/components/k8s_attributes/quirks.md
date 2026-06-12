# `k8s_attributes`: known quirks

## No enrichment usually means an association miss

If telemetry passes through with no `k8s.*` attributes added, the record almost never matched a pod. Walk the [pod_association](configuration.md#pod_association) order: the default starts with the `k8s.pod.ip` resource attribute, then the connection IP. Data that arrives without a pod IP and without a usable connection (e.g. through a gateway) matches nothing. Inspect the incoming record with a `debug` exporter **before** the processor to see what it actually carries, then add a `pod_association` on an attribute the data still has (`k8s.pod.uid`, or `k8s.pod.name` + `k8s.namespace.name`). RBAC gaps produce the same symptom — check with `kubectl auth can-i get pods --as=system:serviceaccount:<ns>:otel-collector`.

## Connection-based association breaks after batching or load balancing

The `from: connection` source reads the IP off the incoming gRPC/HTTP connection. Anything that re-originates the connection — a `batch` processor upstream, a load balancer, a gateway hop — drops that context, and connection association silently stops working. When telemetry crosses a network boundary, switch to a `resource_attribute` source (have a passthrough agent stamp `k8s.pod.ip`, or associate on `k8s.pod.uid`/`k8s.pod.name`).

## Memory scales with pods watched — filter

Without a `filter`, a DaemonSet caches metadata for **every pod in the cluster** (~5 KB each), which OOMKills the collector at scale. Set `filter.node_from_env_var: KUBE_NODE_NAME` (DaemonSet) or `filter.namespace` (single-namespace collector). Extracting labels/annotations `from:` a workload adds another watch (+~30% memory) — only extract what you consume.

## Container attributes need a container identifier

`container.image.name`, `container.image.tag`, `k8s.container.name`, etc. are only added when the record already carries a container identifier — `container.id` (recommended) or `k8s.container.name`. For a multi-container pod without one, the processor can't tell which container the data came from. With `k8s.container.name` but no `k8s.container.restart_count`, it may resolve the wrong container instance; add `k8s.container.restart_count` for an accurate `container.id`.

## The `regex` value-extraction field was removed

The old `extract.labels[].regex` / `extract.annotations[].regex` fields (which parsed a **value** out of a label/annotation) are disallowed by the now-Stable `k8sattr.fieldExtractConfigRegex.disallow` gate. Extract the full value here, then parse it with `transform` and OTTL's `ExtractPatterns`. This is distinct from `key_regex`, which matches label/annotation **keys** and still works.

## Semantic-convention label/annotation format is changing

By default, label/annotation attributes use the **plural** v0 format (`k8s.pod.labels.<key>`). The v1 semconv form is **singular** (`k8s.pod.label.<key>`) and is gated behind `processor.k8sattributes.EmitV1K8sConventions` (emit both) and `processor.k8sattributes.DontEmitV0K8sConventions` (drop the plural form). Migrate by enabling the first gate, repointing dashboards/queries to the singular names, then enabling the second. The gates affect only the **default** `tag_name`; an explicit `tag_name` is unchanged. (The older `k8sattr.labelsAnnotationsSingular.allow` gate is deprecated in favor of these.)

## Don't use it for sidecars, or outside Kubernetes

For a sidecar, inject pod metadata via the Kubernetes downward API as env vars — it's simpler and needs no RBAC. Outside Kubernetes there is no API to watch; use `resourcedetection` for cloud/host attributes instead. Host-network pods also can't be told apart by IP — associate them on `k8s.pod.name`/`k8s.pod.uid`.

## Stability caveats

Traces, metrics, and logs are Beta; profiles are Development. The semconv migration gates above and the v0.151.0–v0.152.0 efficiency/metric changes (e.g. the disabled `otelcol.k8s.pod.association` internal metric, `PartialObjectMetadata` informers) mean behavior and attribute names still shift between releases — confirm against the upstream README for your exact collector version.
