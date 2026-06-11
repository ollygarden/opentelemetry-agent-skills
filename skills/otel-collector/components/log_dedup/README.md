# `log_dedup` processor

| | |
|-|-|
| Kind | processor |
| Type | `log_dedup` |
| Signals | logs |
| Stability | Alpha (processor); Development (telemetry metric `otelcol_dedup_processor_aggregated_logs`) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/logdedupprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/logdedupprocessor> |

**Rename in v0.151.0:** the type was renamed `logdedup` → `log_dedup` to match snake_case. The legacy name is a deprecated alias. New configs should use `log_dedup`; existing `logdedup` configs keep working.

## Description

Aggregates identical log records over a time window and emits a single log with a `log_count` attribute. Instead of forwarding every occurrence of a repeated message, the processor counts duplicates seen within an `interval` and emits one representative record per unique combination at the end of the window.

## Main use-cases

Use it when:
- The pipeline carries high-volume repetitive logs (health checks, polling, retry storms, connection errors).
- You want to reduce backend storage cost while preserving frequency information.
- You care about first/last observed timestamps of a recurring event, not every individual occurrence.

Avoid it when:
- Every entry must be preserved (audit, compliance, security logs).
- Sub-second precision per occurrence is required.
- The backend already deduplicates — you'll deduplicate twice.
- Each log carries a unique identifier or timestamp that makes every record unique (use `exclude_fields` to fix this, or skip the processor).

## Related components

- `groupbyattrs` — groups by attribute, does not deduplicate.
- `transform` — can rewrite/strip fields via OTTL, does not aggregate.
- `filter` — drops records, does not aggregate.

## Details

- [Configuration](configuration.md) — config keys, defaults, validation, and how records are matched/emitted.
- [Verification](verification.md) — telemetrygen recipe to confirm dedup works.
- [Advanced use-cases](advanced.md) — scoping with conditions, field whitelists, per-source instances.
- [Known quirks](quirks.md) — memory model, troubleshooting, anti-patterns.
