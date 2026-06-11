# `transform`: advanced use-cases

## Multiple statement groups

Split work into ordered groups when statements need different contexts, conditions, or error modes. Groups run top-to-bottom; within a group, statements run top-to-bottom too.

```yaml
processors:
  transform:
    error_mode: ignore
    metric_statements:
      # Group 1 — metric context
      - statements:
          - convert_sum_to_gauge() where metric.name == "process.count"
          - set(metric.description, "process count gauge")
      # Group 2 — datapoint context (separate group: different context)
      - statements:
          - limit(datapoint.attributes, 50, ["host.name", "service.name"])
          - delete_matching_keys(datapoint.attributes, "(?i).*temp.*")
```

## Scoping statements with conditions

Group-level `conditions` pre-gate a whole group; they are OR'd, and AND'd with each statement's `where`.

```yaml
processors:
  transform:
    error_mode: ignore
    log_statements:
      # Only JSON-looking logs
      - conditions:
          - IsMatch(log.body, "^\\{")
        statements:
          - merge_maps(log.cache, ParseJSON(log.body), "upsert")
          - set(log.attributes["parsed"], true)
      # Only one service
      - conditions:
          - resource.attributes["service.name"] == "payment-service"
        statements:
          - delete_matching_keys(log.attributes, "(?i).*(card|ssn|password).*")
```

## Context selection trade-offs

Pick the **broadest** context that still reaches what you need: editing at `resource` runs once per resource, while `datapoint`/`spanevent` runs per item and is more expensive. Inference already prefers the most specific context that covers your paths — set `context:` explicitly only to override it, and prefer to express a transform at the highest level that the data allows. Path inventories per context live in the `otel-ottl` skill; the context table is in [configuration.md](configuration.md).

## Combining with `filter`

`transform` and `filter` share the OTTL language but do opposite jobs: `filter` drops, `transform` rewrites. A common pipeline normalizes then drops:

```yaml
service:
  pipelines:
    logs:
      processors: [memory_limiter, transform, filter]
```

Run `transform` first if `filter`'s conditions depend on attributes `transform` sets (or vice-versa) — ordering in the pipeline is load-bearing.

## error_mode for resilience

```yaml
# Production: a failing statement logs but doesn't drop the batch
processors:
  transform:
    error_mode: ignore

# Mixed: most groups tolerant, one group must fail loudly
processors:
  transform:
    error_mode: ignore
    metric_statements:
      - error_mode: propagate          # this group only
        statements:
          - set(metric.unit, "By") where metric.name == "system.memory.usage"
      - statements:                     # inherits ignore
          - set(metric.description, metric.cache["description"])
```

## Named instances

```yaml
processors:
  transform/redact:
    error_mode: ignore
    log_statements:
      - replace_pattern(log.body, "\\b\\d{16}\\b", "****")
  transform/enrich:
    error_mode: ignore
    log_statements:
      - set(log.attributes["env"], "prod")

service:
  pipelines:
    logs/redact:
      processors: [memory_limiter, transform/redact]
    logs/enrich:
      processors: [memory_limiter, transform/enrich]
```

## Common transforms

| Goal | Shape (OTTL — see `otel-ottl` skill for function details) |
|------|------------------------------------------------------------|
| Rename an attribute | `set(resource.attributes["new"], resource.attributes["old"])` then `delete_key(resource.attributes, "old")` — set before delete. |
| Redact a value | `replace_pattern(...)` / `delete_key(...)` / `delete_matching_keys(...)`. |
| Set a default | `set(log.severity_text, "INFO") where log.severity_text == nil or log.severity_text == ""`. |
| Limit / truncate attributes | `keep_keys(...)`, `limit(...)`, `truncate_all(...)`. |
| Convert metric types | `convert_sum_to_gauge()`, `convert_gauge_to_sum(...)` (metric context, with a `where` clause). |
| Reshape histograms | `extract_count_metric(...)`, `extract_sum_metric(...)`, `extract_percentile_metric(...)`. |
| Aggregate datapoints | `aggregate_on_attributes(...)`, `aggregate_on_attribute_value(...)`. |

These are processor-specific OTTL functions; their full signatures and the general OTTL converter set live in the `otel-ottl` skill.
