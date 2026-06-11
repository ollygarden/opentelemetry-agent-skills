# `filter`: known quirks

## Data is permanently dropped

`filter` removes data from the pipeline; there is no recovery downstream. Test conditions with `error_mode: propagate` and `debug` logging before rolling out.

## Dropping a metric's only datapoint removes the metric

Filtering at the `datapoint` level that empties a metric also removes the metric — if all datapoints of a metric match, the whole metric disappears. Likewise, if all telemetry for a resource is dropped, the resource is removed from the payload, and if all span events of a span are dropped the span itself survives (only the events go). Be deliberate about which level you filter at.

## Orphaned telemetry / broken trace completeness

- Dropping a **parent** span orphans its children — they remain in the trace but point at a span that no longer exists, which breaks waterfall views.
- Dropping spans that logs reference (via trace/span ID) breaks log↔trace correlation.
- Dropping arbitrary spans makes a trace incomplete. When you need a whole-trace keep/drop decision, use `tail_sampling`, not `filter`.

## error_mode behavior

- Default is `propagate`: a single failing condition (e.g. a type mismatch or a nil dereference) **drops the entire batch**, not just the offending item. This surprises people in production — set `error_mode: ignore` there.
- `silent` hides evaluation errors entirely; if a filter "isn't working," confirm it isn't set to `silent`.
- The `processor.filter.defaultErrorModeIgnore` feature gate can flip the default to `ignore`; don't assume the default without checking whether the gate is enabled.

## Stability caveats

All signals are **Alpha** (profiles are **Development**). Config keys can still change between releases, and profile filtering may break. Check the upstream README for the running version before treating any key as stable.

## Troubleshooting

**Nothing is dropped.**
- Wrong context — span fields used in a `spanevent` block, or metric fields used directly in `datapoint` (use `metric.name`).
- Attribute doesn't exist — add a nil check: `attributes["x"] != nil and attributes["x"] == "y"`.
- `error_mode: silent` is swallowing evaluation errors — switch to `propagate` while debugging.

**Everything is dropped.**
- Overly broad condition (`IsMatch(name, ".*")` matches all). Tighten it and re-test with debug logging.

**Validation error: `cannot use ottl conditions and include/exclude ... at the same time`.**
- You mixed OTTL conditions and legacy include/exclude for the same signal. Pick one.

**Verifying effectiveness.** The processor emits `processor_filter_spans.filtered`, `processor_filter_datapoints.filtered`, `processor_filter_logs.filtered` (and `processor_filter_profiles.filtered`, development) — watch these to confirm drops.

## Anti-patterns

**Datapoint-level filtering when a whole metric should go.**

```yaml
# Less efficient — evaluates every datapoint
metrics:
  datapoint:
    - 'resource.attributes["env"] == "test"'

# Prefer — drops the whole metric in one check
metrics:
  metric:
    - 'resource.attributes["env"] == "test"'
```

**Dropping parent/correlated spans.**

```yaml
# Risky — orphans children, breaks log correlation
traces:
  span:
    - 'kind == SPAN_KIND_SERVER'
```

Filter leaf/noise spans, or move whole-trace decisions to `tail_sampling`.

**Complex negative-lookahead regex.**

```yaml
# Hard to maintain, error-prone
logs:
  log_record:
    - 'IsMatch(body, "^(?!.*(error|warn|fail)).*$")'
```

Prefer explicit positive conditions (`severity_number < SEVERITY_NUMBER_WARN`).
