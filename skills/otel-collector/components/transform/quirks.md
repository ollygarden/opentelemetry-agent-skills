# `transform`: known quirks

## Statement order matters

Statements run top-to-bottom and later ones see earlier effects. A classic bug is deleting an attribute before reading it:

```yaml
# BAD — delete runs first, set reads nil
- delete_key(resource.attributes, "k8s.namespace.name")
- set(resource.attributes["namespace"], resource.attributes["k8s.namespace.name"])

# GOOD — set first, then delete
- set(resource.attributes["namespace"], resource.attributes["k8s.namespace.name"])
- delete_key(resource.attributes, "k8s.namespace.name")
```

## Mutating shared resource/scope affects everything under it

A `resource`-context statement runs once per resource and changes are seen by **all** spans/metrics/logs sharing that resource. Setting a resource attribute from per-item data (e.g. a single log's attribute) leaks that value across the whole group. When per-item resource edits are genuinely needed for logs, enable `flatten_data` (behind the `transform.flatten.logs` feature gate) so each record gets its own resource copy — at the cost of copying and hashing every record.

## `error_mode: propagate` drops the whole batch on error

Under `propagate`, the first statement that errors (a type mismatch, a nil dereference like `log.cache["x"]["y"]` when `x` is nil) **drops the entire payload**, not just the offending item, and the remaining statements never run. The default is now `ignore` (since v0.153.0), but any config or group that still sets `propagate` behaves this way. Keep `error_mode: ignore` (or `silent`) in production, and nil-check before accessing nested fields:

```yaml
- set(log.attributes["uid"], log.cache["user"]["id"]) where log.cache["user"] != nil
```

`silent` hides evaluation errors entirely — if a transform "isn't working," confirm it isn't set to `silent`. The default was flipped from `propagate` to `ignore` by the `processor.transform.defaultErrorModeIgnore` feature gate, which reached beta / on-by-default in v0.153.0; on pre-v0.153.0 collectors the default is still `propagate`, and the gate can be disabled (`--feature-gates=-processor.transform.defaultErrorModeIgnore`) to restore it. Don't assume the default without knowing the running version.

## Context performance

Statements evaluate per item at their inferred context. A `datapoint`- or `spanevent`-context group runs for every datapoint/event, which is far more work than a `resource`- or `metric`-context group. Express each transform at the broadest context that reaches the data. Mixing incompatible paths/functions in one group makes context inference fail — split them:

```yaml
# FAILS — convert_sum_to_gauge is metric-context-only; limit needs datapoint context
metric_statements:
  - convert_sum_to_gauge() where metric.name == "process.count"
  - limit(datapoint.attributes, 100, ["host.name"])

# WORKS — one group per context
metric_statements:
  - statements:
      - convert_sum_to_gauge() where metric.name == "process.count"
  - statements:
      - limit(datapoint.attributes, 100, ["host.name"])
```

## OTTL version drift

`transform` is a thin shell around OTTL, which evolves quickly: function names, signatures, path syntax, and enums change between collector releases (e.g. the path-prefix style `log.body` vs older bare `body`; `set_semconv_span_name` semconv versions; `extract_percentile_metric` added in v0.151.0). Validate statements against the running version — see the `otel-ottl` skill — rather than assuming a snippet from another release still parses.

## Don't change identity carelessly

- Setting `span.trace_id`/`span.span_id` orphans spans and breaks trace/log correlation.
- Renaming a metric or stripping all its datapoint attributes (`set(metric.name, ...)`, `delete_matching_keys(datapoint.attributes, ".*")`) collapses distinct series into one — an identity conflict.
- `copy_metric` without a `where` clause that excludes the copy creates an infinite loop. Always gate it.

## Stability caveats

Traces, metrics, and logs are **Beta**; profiles are **Development**. Beta means breaking changes are rare but config keys can still move; profile support may break. Check the upstream README for the running version before treating any key as stable.

## Troubleshooting

**Statement does nothing.**
- A `where` clause or group `conditions` didn't match.
- Wrong context inferred — set `context:` explicitly, or split the group.
- The field is nil; guard with a nil check.
- Enable `service.telemetry.logs.level: debug` to see the `TransformContext` before/after each statement and whether conditions matched.

**Errors logged but telemetry still flows.** Expected under `error_mode: ignore`; use `silent` to quiet the logs once you've accepted the errors.

**Whole payloads disappear.** `error_mode: propagate` plus an erroring statement. Switch to `ignore` or fix the statement.

**"unexpected token" / "unknown context" on startup.** Incompatible paths/functions in one group (split them), a syntax error, an unknown/renamed function, or mixing the flat and explicit-group styles in one signal's list.

## Anti-patterns

**Using `transform` to filter.** Setting a flag so a later `filter` drops the item is a smell — drop directly with `filter`.

**Hardcoding values that should come from detection.** `set(resource.attributes["deployment.environment.name"], "production")` belongs in resource detection or the SDK, not a static transform, unless you genuinely have no better source.

**`propagate` in production.** Any single error drops the batch; use `ignore`/`silent` and nil-guard nested access.

**Over-transformation.** Long chains of rewrites per item add latency and CPU; do at the source what you can, and transform only what must change downstream.
