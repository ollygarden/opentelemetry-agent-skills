# `drain`: advanced use-cases

## Persist the tree across restarts

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

## Tune cluster granularity

```yaml
processors:
  drain:
    tree_depth: 5          # higher → more specific templates
    merge_threshold: 0.5   # higher → more specific templates
```

Raise `tree_depth`/`merge_threshold` for **more specific** templates (less over-merging); lower them for **more general** ones. Add `extra_delimiters` when tokens you want split are joined by non-whitespace characters such as `:` or `,`.

## Suppress warmup before annotating

```yaml
processors:
  drain:
    warmup_min_clusters: 20
```

With `warmup_min_clusters > 0`, the processor trains on every record but does **not** write the template attribute until that many distinct clusters exist. Records still pass through immediately — no buffering, no added latency — they simply arrive unannotated during the warmup window (and are not retroactively annotated). This keeps unstable, half-trained templates from reaching downstream `filter`/`routing` rules.

## Split structured tokens with `extra_delimiters`

```yaml
processors:
  drain:
    extra_delimiters: [":", ","]
```

Useful when values you want abstracted are glued to neighboring tokens by punctuation rather than whitespace.

## Combine templates with downstream components

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
