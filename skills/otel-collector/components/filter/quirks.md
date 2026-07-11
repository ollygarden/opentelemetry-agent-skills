# `filter`: known quirks

## Data is permanently dropped

`filter` removes data from the pipeline; there is no recovery downstream. Test conditions with `error_mode: propagate` (explicitly — the default is now `ignore`) and `debug` logging before rolling out.

## Dropping a metric's only datapoint removes the metric

Filtering at the `datapoint` level that empties a metric also removes the metric — if all datapoints of a metric match, the whole metric disappears. Likewise, if all telemetry for a resource is dropped, the resource is removed from the payload, and if all span events of a span are dropped the span itself survives (only the events go). Be deliberate about which level you filter at.

## Orphaned telemetry / broken trace completeness

- Dropping a **parent** span orphans its children — they remain in the trace but point at a span that no longer exists, which breaks waterfall views.
- Dropping spans that logs reference (via trace/span ID) breaks log↔trace correlation.
- Dropping arbitrary spans makes a trace incomplete. When you need a whole-trace keep/drop decision, use `tail_sampling`, not `filter`.

## error_mode behavior

- Default is `ignore` since v0.153.0 (the `processor.filter.defaultErrorModeIgnore` gate went **beta / enabled by default**): a failing condition is logged and skipped, valid data survives.
- Setting `error_mode: propagate` — or disabling the gate with `--feature-gates=-processor.filter.defaultErrorModeIgnore` — makes a single failing condition (e.g. a type mismatch or a nil dereference) **drop the entire batch**, not just the offending item. Older collectors (pre-v0.153.0) defaulted to `propagate`, so the same config behaves differently across versions unless `error_mode` is set explicitly.
- `silent` hides evaluation errors entirely; if a filter "isn't working," confirm it isn't set to `silent`.

## Stability caveats

All signals are **Alpha** (profiles are **Development**). Config keys can still change between releases, and profile filtering may break. Check the upstream README for the running version before treating any key as stable.

## Troubleshooting

**Nothing is dropped.**
- Missing/wrong path prefix — in `*_conditions`, paths must be prefixed (`span.name`, `log.body`, `metric.name`, `datapoint.value_int`); the context is inferred from those prefixes.
- Attribute doesn't exist — add a nil check: `log.attributes["x"] != nil and log.attributes["x"] == "y"`.
- `error_mode: silent` is swallowing evaluation errors — switch to `propagate` while debugging.

**Everything is dropped.**
- Overly broad condition (`IsMatch(span.name, ".*")` matches all). Tighten it and re-test with debug logging.

**Validation error: `configuring multiple configuration styles is not supported`.**
- You mixed basic (flat string list) and advanced (`context`/`conditions` objects) forms in one signal — use one style.

**Validation error: `cannot use context inferred ... conditions and the settings ... at the same time`.**
- You mixed `*_conditions` with a deprecated per-signal block or include/exclude for the same signal. Pick one.

**Verifying effectiveness.** The processor emits `processor_filter_spans.filtered`, `processor_filter_datapoints.filtered`, `processor_filter_logs.filtered` (and `processor_filter_profiles.filtered`, development) — watch these to confirm drops.

## Anti-patterns

**Datapoint-level filtering when a whole metric should go.**

```yaml
# Less efficient — inferred as datapoint context, evaluates every datapoint
metric_conditions:
  - 'datapoint.attributes["env"] == "test"'

# Prefer — resource condition drops the whole resource in one check
metric_conditions:
  - 'resource.attributes["env"] == "test"'
```

**Dropping parent/correlated spans.**

```yaml
# Risky — orphans children, breaks log correlation
trace_conditions:
  - 'span.kind == SPAN_KIND_SERVER'
```

Filter leaf/noise spans, or move whole-trace decisions to `tail_sampling`.

**Complex negative-lookahead regex.**

```yaml
# Rejected at config load — Go's RE2 engine has no lookahead
log_conditions:
  - 'IsMatch(log.body, "^(?!.*(error|warn|fail)).*$")'
```

`IsMatch` compiles with Go's RE2 engine, which does not support lookahead (`(?!`); the pattern above fails validation outright (`invalid or unsupported Perl syntax: `(?!``). Prefer explicit positive conditions (`log.severity_number < SEVERITY_NUMBER_WARN`).
