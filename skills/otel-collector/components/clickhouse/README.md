# `clickhouse` exporter

| | |
|-|-|
| Kind | exporter |
| Type | `clickhouse` |
| Signals | traces (Beta), logs (Beta), metrics (Alpha), profiles (Development) |
| Distributions | contrib |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/exporter/clickhouseexporter> |

## Description

Writes telemetry to [ClickHouse](https://clickhouse.com/docs) tables through the `clickhouse-go` v2 driver, over the native protocol (`tcp://`, `clickhouse://`) or HTTP (`http://`, `https://`). Each signal has its own tables: `otel_logs`; `otel_traces` plus a trace-ID lookup table and materialized view; one `otel_metrics_<type>` table per metric type; `otel_profiles`.

## Main use-cases

Use when:
- You store logs, traces, metrics, or profiles in ClickHouse and query them with SQL.

Avoid when:
- A metrics backend must keep sub-second timestamps in tables the exporter creates: the metric tables use second-precision `DateTime` (see [quirks](quirks.md#metric-timestamps-are-second-precision)).

## Related components

- [`failover` connector](../failover/README.md) — switches pipelines when this exporter returns errors; see [errors and retries](quirks.md#every-error-is-retried).
- [`otlp_grpc` exporter](../otlp_exporter/configuration.md#sending_queue) — reference for the shared `sending_queue` and `retry_on_failure` keys.
- [`file_storage`](../file_storage/README.md) — makes `sending_queue` persistent across restarts.
- `batch` processor — pipeline-level batching; see [Pipeline placement](../../SKILL.md#pipeline-placement).

## Details

- [Configuration](configuration.md) — keys, defaults, DSN merging, connection-pool sizing.
- [Verification](verification.md) — `telemetrygen` into local ClickHouse, proven with SQL.
- [Advanced use-cases](advanced.md) — schema DDL, clusters, rows per `INSERT`, async inserts.
- [Known quirks](quirks.md) — DDL failure stops startup, retried errors, duplicates, error→fix table.
