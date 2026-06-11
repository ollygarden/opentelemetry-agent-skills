# `drain`: known quirks

## Unbounded memory with `max_clusters: 0`

The default `max_clusters: 0` means *unlimited* — active clusters and tree nodes grow with unique log patterns. On high-diversity streams this can grow without bound. Bound it with `max_clusters` (enables LRU eviction) and cap fan-out with `max_node_children`. Track `otelcol_processor_drain_clusters_active` for growth.

## Warmup withholds templates

When `warmup_min_clusters > 0`, the template attribute does not appear until that many clusters have been observed. If the attribute "never shows up", check `otelcol_processor_drain_clusters_active` against the threshold, or lower `warmup_min_clusters`. The warmup window is observable as `otelcol_processor_incoming_items - otelcol_processor_drain_log_records_annotated`.

## Templates reset on restart without persistence

The tree lives in memory. Without a `storage` extension, templates are rebuilt (and re-warmed) on every restart, and independent instances may disagree on a template during early training on low-volume or highly variable streams. Mitigate with `seed_templates`/`seed_logs`, `warmup_min_clusters`, or shared `storage` (see [Advanced use-cases](advanced.md)).

## Structured (map) bodies

If the body is a map, the full serialized form is templated by default, which yields low-value templates. Set `body_field` to the **single top-level** message key (full OTTL paths like `body["event"]["message"]` are not supported), or promote that field to a plain string body upstream (e.g. a `move` operator in the filelog receiver). The full body is used unchanged if the key is absent or the body is not a map.

## Alpha stability and provisional attribute name

The processor is **Alpha** — keys and behavior may change. The default `log.record.template` tracks a *proposed* (not yet adopted) semantic convention ([semantic-conventions#1283](https://github.com/open-telemetry/semantic-conventions/issues/1283), [#2064](https://github.com/open-telemetry/semantic-conventions/issues/2064)); pin `template_attribute` explicitly if you depend on the value. The internal metrics `otelcol_processor_drain_clusters_active` and `otelcol_processor_drain_log_records_annotated` are Development stability and may change or be removed.

## Drain does not reduce volume

It only annotates. Expecting `drain` alone to drop or aggregate logs is a common mistake — always add a `filter`/`routing` (or `log_dedup`) component downstream to act on the template.
