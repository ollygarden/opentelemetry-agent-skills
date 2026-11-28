# `failover` connector

| | |
|-|-|
| Kind | connector |
| Type | `failover` |
| Signals | traces→traces (Alpha), metrics→metrics (Alpha), logs→logs (Alpha) — same-signal only, data forwarded unchanged |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/connector/failoverconnector` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/connector/failoverconnector> |

## Description

Sends each batch to the highest-priority healthy level of an ordered `priority_levels` list. Each level is one or more pipelines. The connector is the **exporter** of the input pipeline and the **receiver** of every priority pipeline. A level fails when its consume call **returns an error synchronously**. The connector then resends the batch to the next level, and every `retry_interval` it probes higher levels with live data to fail back.

An exporter `sending_queue` or a `batch` processor in a priority pipeline returns success before the export happens, so behind such a stage the connector does not see a dead destination ([hidden failures](quirks.md#async-stages-in-a-priority-pipeline-hide-failures)). The connector has its own [`sending_queue`](configuration.md#connector-sending_queue).

## Main use-cases

Use when:
- A secondary destination (another backend, another region's gateway, a local `file` archive) should take traffic only while the primary rejects it.
- Destinations have a strict order of preference, and traffic should return to the preferred one once it recovers.

Avoid when:
- Every destination should get a copy — list several exporters in one pipeline.
- The destination depends on the data (tenant, environment, severity) — use [`routing`](../routing/README.md).
- Load should be spread across a pool of backends — use [`load_balancing`](../load_balancing/README.md).

## Related components

- [`otlp_grpc` exporter](../otlp_exporter/configuration.md#sending_queue) — the usual priority-pipeline exporter; reference for `sending_queue` and [`retry_on_failure`](../otlp_exporter/configuration.md#retry_on_failure).
- [`file_storage`](../file_storage/README.md) — makes the connector's `sending_queue` persistent.

## Details

- [Configuration](configuration.md) — keys and defaults, the failover and failback algorithm, and the connector `sending_queue`.
- [Verification](verification.md) — two-collector `telemetrygen` recipe showing failover and failback, plus an exporter-queue variant.
- [Advanced use-cases](advanced.md) — several pipelines per level, an instance per signal, `condition`, a persistent connector queue.
- [Known quirks](quirks.md) — which downstream stages hide failures, recovery delay, version traps, and validation errors.
