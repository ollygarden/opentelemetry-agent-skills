# `drain` processor

| | |
|-|-|
| Kind | processor |
| Type | `drain` |
| Signals | logs |
| Stability | Alpha (processor); Development (telemetry metrics `otelcol_processor_drain_*`) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/drainprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/drainprocessor> |

Logs support is Alpha since v0.151.0; configuration keys and behavior may change between releases.

## Description

Applies the [Drain](https://pinjiahe.github.io/papers/ICWS17.pdf) log-clustering algorithm to each log record, derives a **template string** (e.g. `user <*> logged in from <*>`), and writes it to a configurable attribute (`log.record.template` by default). The template becomes a stable key for grouping, filtering, routing, or sampling whole classes of logs.

Drain tokenizes each log line and walks a **fixed-depth parse tree** (depth set by `tree_depth`). Lines with similar token structure land in the same cluster, and each cluster carries a template whose varying tokens become `<*>` wildcards while stable tokens are kept verbatim. The processor **annotates only** — it does not drop, aggregate, reorder, or otherwise modify the flow of logs.

## Main use-cases

Use it when:
- You want to group high-volume, semi-structured logs into a small number of stable patterns.
- You need a stable key for downstream filtering, routing, or sampling of log classes.
- Your log bodies are free-form strings (or a structured body with a clear message field).
- You want to identify the noisiest log patterns before deciding what to drop.

Avoid it when:
- Your logs are already fully structured with a reliable event identifier — use that attribute directly.
- You need exact, hand-authored templates with no early-training variance — seed them, or template upstream.
- Bodies are large JSON blobs with no single message field — templating the serialized map produces low-value templates (set `body_field`, or promote the message field upstream).

## Related components

- `filter` — drop or keep logs by matching the derived template.
- `routing` (connector) — route log classes to different pipelines by template.
- `log_dedup` — aggregate identical logs with a count; complements template-based grouping.
- `transform` — OTTL-based per-record transformations.
- Storage extensions (e.g. `file_storage`) — back `storage` for snapshot persistence.

## Details

- [Configuration](configuration.md) — config keys, defaults, validation rules, and structured-body behavior. Open when wiring up `drain` or tuning what gets templated.
- [Verification](verification.md) — telemetrygen recipe to confirm templates are written. Open when checking the processor is bundled and working end-to-end.
- [Advanced use-cases](advanced.md) — persistence, granularity tuning, warmup, delimiters, and combining with downstream components. Open when going beyond a single stock instance.
- [Known quirks](quirks.md) — unbounded memory, warmup withholding, restart resets, map bodies, Alpha stability, and the "does not reduce volume" trap. Open when something behaves unexpectedly.
