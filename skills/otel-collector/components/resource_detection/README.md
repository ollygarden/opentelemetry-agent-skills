# `resource_detection` processor

| | |
|-|-|
| Kind | processor |
| Type | `resource_detection` (renamed from `resourcedetection` in v0.153.0; the old name remains as a deprecated alias) |
| Signals | traces (Beta), metrics (Beta), logs (Beta), profiles (Development) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor> |

## Description

Detects **resource** attributes from the environment the Collector runs in — host machine, cloud provider metadata services, container runtimes, Kubernetes, and environment variables — and merges them onto every span, metric data point, and log record passing through the pipeline, in a form that follows the [OpenTelemetry resource semantic conventions](https://github.com/open-telemetry/semantic-conventions/tree/main/docs/resource). You enable detection by listing one or more named **detectors** in `detectors`; each detector contributes the attributes it knows how to find (e.g. `system` → `host.name`/`os.type`, `ec2` → `cloud.provider`/`cloud.region`/`host.id`, `gcp` → `cloud.platform`/`k8s.cluster.name`).

Detection normally runs **once at startup** and the results are cached. Detectors run in the order listed and, by default (`override: true`), the detected values overwrite any existing resource attributes on incoming telemetry; set `override: false` to keep values already present (the safer choice when telemetry arrives pre-enriched from SDKs or upstream Collectors). Unlike [`resource`](../resource/README.md), it does not take an explicit attribute action list — it *discovers* attributes rather than setting literal values, and is commonly chained **before** `resource` so you can rename or trim what it found.

## Main use-cases

Use it when:
- You want host identity on telemetry from a Collector running on a VM or bare metal — `host.name`, `host.id`, `os.type`, `host.arch` (the `system` detector).
- You run the Collector on a cloud platform and want `cloud.*` / `k8s.cluster.name` / `faas.*` attributes auto-populated (`ec2`, `gcp`, `azure`, `aks`, `eks`, … detectors).
- You want to pull resource attributes from `OTEL_RESOURCE_ATTRIBUTES` into telemetry that lacks them (the `env` detector — the default).
- You need infra context attached at the Collector rather than configured into every SDK.

Avoid it when:
- You want to set **literal/static** resource values (`service.name`, `deployment.environment.name`) — use [`resource`](../resource/README.md); detection discovers from the environment, it does not invent values.
- You need per-pod Kubernetes metadata keyed off each record's source pod — use [`k8s_attributes`](../k8s_attributes/README.md); the `k8s_api`/`kubeadm`/`openshift` detectors here describe the **node/cluster the Collector runs on**, not the workload that emitted the telemetry.
- You need to edit span/log/datapoint attributes — use [`attributes`](../attributes/README.md) or `transform`.

## Related components

- [`resource`](../resource/README.md) — sets/renames resource attributes from an explicit action list. Run `resource_detection` first to discover, then `resource` to normalize or drop what was found.
- [`k8s_attributes`](../k8s_attributes/README.md) — per-record pod/namespace/workload enrichment from the k8s API. `resource_detection`'s k8s detectors describe the host/node the Collector itself runs on; the two are complementary.
- [`attributes`](../attributes/README.md) — edits span/log/datapoint attributes (telemetry scope) rather than the resource.

## Details

- [Configuration](configuration.md) — top-level keys (`detectors`, `override`, `refresh_interval`, HTTP client settings), the per-detector `resource_attributes` enable/disable mechanism, and the full detector catalog with the attributes each populates.
- [Verification](verification.md) — telemetrygen recipe using the `env` and `system` detectors, confirming detected attributes in the **Resource attributes** block of `debug` output.
- [Advanced use-cases](advanced.md) — detector ordering, combining detectors, `override: false` for pre-enriched telemetry, chaining with `resource`, cloud tag/label capture, and `refresh_interval`.
- [Known quirks](quirks.md) — a failed detector stops the Collector from starting, the `override` footgun, "describes the Collector's host, not the workload", `refresh_interval` cardinality cost, the `resourcedetection` deprecated alias, and Docker/cloud environment caveats.
