# `cardinality_guardian`: known quirks

## Single-writer violation in enforcement mode (`tag_only: false`)

This is the headline gotcha. In enforcement mode, stripping a label can collapse multiple data points onto the **same series identity**. Cumulative backends (Prometheus and similar) interpret the overlapping cumulative values as **counter resets**, silently corrupting `rate()` and `increase()` for affected Sum and Histogram metrics — the numbers look plausible but are wrong, with no error surfaced anywhere.

**Fix / recommendation:** run `tag_only: true` plus a routing connector (see [Advanced use-cases](advanced.md)) in production until a downstream spatial-reaggregation processor exists. Reserve `tag_only: false` for gauges or for pipelines you have verified are safe.

Add the labels your dashboards group by (`service.name`, `environment`, etc.) to `never_drop_labels` so enforcement can never break them.

## Nothing gets stripped despite an obvious explosion

The check is on per-epoch *delta*, not absolute cardinality. If the explosion happened before the processor started (steady state), no delta is observed. Lower `epoch_duration_seconds` or restart to re-baseline. Also confirm the label is not in `never_drop_labels`.

## Development stability

Introduced at **Development** stability in contrib v0.152.0. Configuration and behavior may change without a deprecation period — pin the collector version and re-check defaults on upgrade.

## Not in any distribution

Not bundled in the `contrib` or `k8s` distributions. You must build a custom collector that includes `processor/cardinalityguardianprocessor` via the OpenTelemetry Collector Builder (OCB).

## Mutates data

`MutatesData: true` — the processor takes ownership of the incoming `pmetric.Metrics`.
