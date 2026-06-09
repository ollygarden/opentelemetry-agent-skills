# `drain` processor

| | |
|-|-|
| Kind | processor |
| Signals | logs |
| Stability | Alpha (processor); Development (telemetry metrics `otelcol_processor_drain_*`) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/drainprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/drainprocessor> |

The `type` is `drain`. Logs support is Alpha since v0.151.0; configuration keys and behavior may change between releases.

## Description

Applies the [Drain](https://pinjiahe.github.io/papers/ICWS17.pdf) log-clustering algorithm to each log record, derives a **template string** (e.g. `user <*> logged in from <*>`), and writes it to a configurable attribute (`log.record.template` by default). The template becomes a stable key for grouping, filtering, routing, or sampling whole classes of logs.

The processor **annotates only** — it does not drop, aggregate, reorder, or otherwise modify the flow of logs. To act on the derived template (for example, dropping a noisy log class), pair it with a `filter` processor or `routing` connector downstream.

### How the parse-tree works

Drain tokenizes each log line and walks a **fixed-depth parse tree** (depth set by `tree_depth`). Lines with similar token structure land in the same **cluster**; each cluster carries a template. As more logs arrive, the template is refined — tokens that vary across a cluster's members become `<*>` wildcards while stable tokens are kept verbatim. The algorithm is deterministic: given the same configuration and a representative sample of log forms, the same token structure always yields the same template. Because of this, independent collector instances converge on identical templates once each has seen enough lines.

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

## Typical config

```yaml
processors:
  drain:
    template_attribute: log.record.template

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, drain]
      exporters: [otlphttp]
```

### Configuration reference

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `tree_depth` | int | `4` | Max depth of the Drain parse tree (`depth` in the paper). Higher = more specific templates. **Minimum: 3.** |
| `merge_threshold` | float | `0.4` | Minimum fraction of tokens that must match an existing cluster for a line to merge into it rather than form a new cluster (`st` in the paper). Range `[0.0, 1.0]`. |
| `max_node_children` | int | `100` | Maximum children per internal parse-tree node (`maxChild` in the paper). Bounds memory on high-cardinality token positions. |
| `max_clusters` | int | `0` | Maximum clusters tracked; when exceeded, the least-recently-used cluster is evicted. `0` = unlimited. |
| `extra_delimiters` | []string | `[]` | Additional token delimiters beyond whitespace (e.g. `[",", ":"]`). |
| `body_field` | string | `""` | If set and the body is a map, the value of this **single top-level key** is templated instead of the full body. Full OTTL paths are not supported. `""` = use the full body string. |
| `template_attribute` | string | `"log.record.template"` | Attribute key the derived template string is written to. |
| `seed_templates` | []string | `[]` | Template strings pre-loaded at startup. Empty/whitespace-only entries are skipped. |
| `seed_logs` | []string | `[]` | Raw example log lines trained on at startup. Empty/whitespace-only entries are skipped. |
| `warmup_min_clusters` | int | `0` | Distinct clusters that must be observed before annotation starts. `0` disables warmup. |
| `storage` | component ID | `""` | ID of a storage extension used to persist the tree across restarts. `""` = disabled. |
| `save_interval` | duration | `0s` | Interval between periodic snapshot saves. `0s` = save on shutdown only. Requires `storage`. |

### Validation rules

- `tree_depth` must be `>= 3`.
- `merge_threshold` must be in `[0.0, 1.0]`.
- `warmup_min_clusters` must be `>= 0`.
- `save_interval` must be `>= 0`.
- `save_interval > 0` requires `storage` to be set.

## Verification

Confirm `drain` is in your distribution (check the component's metadata); build via OCB if it is not yet bundled.

Config (`drain-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  drain:
    template_attribute: log.record.template
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [drain]
      exporters: [debug]
```

Send log records that share structure but differ in values (see the `otel-telemetrygen` skill):

```bash
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 100 --body "user alice logged in from 10.0.0.1" --duration 5s
```

telemetrygen sends a **constant** body per invocation (`--body` defaults to `"the message"`), so a single run produces one template with no wildcards. To watch several distinct values collapse to one template (e.g. `user <*> logged in from <*>`), run the command a few times with different `--body` values — for example `"user bob logged in from 192.168.1.1"` and `"user carol logged in from 172.16.0.9"` — or point the pipeline at a real log source.

**What proves it worked:** the `debug` exporter shows each record annotated with a `log.record.template` attribute holding the clustered template string (e.g. `user <*> logged in from <*>`).

## Advanced use-cases

### Persist the tree across restarts

Without persistence the parse tree is rebuilt from scratch on every restart, so templates reset and warm up again. Back the processor with a storage extension and snapshot periodically:

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/drain

processors:
  drain:
    storage: file_storage
    save_interval: 5m

service:
  extensions: [file_storage]
```

On startup a valid snapshot is loaded (skipping `seed_templates`/`seed_logs`, and warmup if the loaded tree already satisfies `warmup_min_clusters`); `save_interval > 0` snapshots periodically; a final snapshot is always saved on shutdown. With a shared backend (Redis, a database) new instances inherit a tree trained by running instances, avoiding cold starts.

### Tune cluster granularity

```yaml
processors:
  drain:
    tree_depth: 5          # higher → more specific templates
    merge_threshold: 0.5   # higher → more specific templates
```

Raise `tree_depth`/`merge_threshold` for **more specific** templates (less over-merging); lower them for **more general** ones. Add `extra_delimiters` when tokens you want split are joined by non-whitespace characters such as `:` or `,`.

### Suppress warmup before annotating

```yaml
processors:
  drain:
    warmup_min_clusters: 20
```

With `warmup_min_clusters > 0`, the processor trains on every record but does **not** write the template attribute until that many distinct clusters exist. Records still pass through immediately — no buffering, no added latency — they simply arrive unannotated during the warmup window (and are not retroactively annotated). This keeps unstable, half-trained templates from reaching downstream `filter`/`routing` rules.

### Split structured tokens with `extra_delimiters`

```yaml
processors:
  drain:
    extra_delimiters: [":", ","]
```

Useful when values you want abstracted are glued to neighboring tokens by punctuation rather than whitespace.

### Combine templates with downstream components

Feed the derived template into a `filter` to drop a noisy class, a `routing` connector to fan log classes into separate pipelines, or a `log_dedup` processor to aggregate identical records with a count:

```yaml
processors:
  drain:
  filter/drop_noisy:
    error_mode: ignore
    logs:
      log_record:
        - attributes["log.record.template"] == "heartbeat ping <*>"
        - attributes["log.record.template"] == "connected to <*>"

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [drain, filter/drop_noisy]
      exporters: [otlp]
```

## Known quirks

### Unbounded memory with `max_clusters: 0`

The default `max_clusters: 0` means *unlimited* — active clusters and tree nodes grow with unique log patterns. On high-diversity streams this can grow without bound. Bound it with `max_clusters` (enables LRU eviction) and cap fan-out with `max_node_children`. Track `otelcol_processor_drain_clusters_active` for growth.

### Warmup withholds templates

When `warmup_min_clusters > 0`, the template attribute does not appear until that many clusters have been observed. If the attribute "never shows up", check `otelcol_processor_drain_clusters_active` against the threshold, or lower `warmup_min_clusters`. The warmup window is observable as `otelcol_processor_incoming_items - otelcol_processor_drain_log_records_annotated`.

### Templates reset on restart without persistence

The tree lives in memory. Without a `storage` extension, templates are rebuilt (and re-warmed) on every restart, and independent instances may disagree on a template during early training on low-volume or highly variable streams. Mitigate with `seed_templates`/`seed_logs`, `warmup_min_clusters`, or shared `storage`.

### Structured (map) bodies

If the body is a map, the full serialized form is templated by default, which yields low-value templates. Set `body_field` to the **single top-level** message key (full OTTL paths like `body["event"]["message"]` are not supported), or promote that field to a plain string body upstream (e.g. a `move` operator in the filelog receiver). The full body is used unchanged if the key is absent or the body is not a map.

### Alpha stability and provisional attribute name

The processor is **Alpha** — keys and behavior may change. The default `log.record.template` tracks a *proposed* (not yet adopted) semantic convention ([semantic-conventions#1283](https://github.com/open-telemetry/semantic-conventions/issues/1283), [#2064](https://github.com/open-telemetry/semantic-conventions/issues/2064)); pin `template_attribute` explicitly if you depend on the value. The internal metrics `otelcol_processor_drain_clusters_active` and `otelcol_processor_drain_log_records_annotated` are Development stability and may change or be removed.

### Drain does not reduce volume

It only annotates. Expecting `drain` alone to drop or aggregate logs is a common mistake — always add a `filter`/`routing` (or `log_dedup`) component downstream to act on the template.

## Related components

- `filter` — drop or keep logs by matching the derived template.
- `routing` (connector) — route log classes to different pipelines by template.
- `log_dedup` — aggregate identical logs with a count; complements template-based grouping.
- `transform` — OTTL-based per-record transformations.
- Storage extensions (e.g. `file_storage`) — back `storage` for snapshot persistence.
