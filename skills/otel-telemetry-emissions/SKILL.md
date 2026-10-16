---
name: otel-telemetry-emissions
description: Version-pinned inventory of the telemetry (spans, metrics, logs, attributes) emitted by OpenTelemetry collector components and SDK instrumentation packages. Use when working with a covered component — what it emits at a given version, or how emission changed across versions — and when upgrading a component or SDK version, to see the telemetry after the change.
---

# Telemetry Emissions

A registry of what telemetry OpenTelemetry components actually emit, pinned to
upstream versions.

> **Note:** The telemetry emissions in these files were detected by LLM
> analysis of the upstream source and may contain errors. Verify against the
> component's source or docs before relying on a row for critical decisions.

## Layout

```
<repo basename>/<path in repo>/v<version>.md
```

Example: `opentelemetry-collector-contrib/receiver/kafkareceiver/v0.158.0.md`

One file per scanned version. The files in a component directory are its
telemetry history; `index.md` lists every file — start there to see what is
covered.

## Looking up a component at a version

1. Find the component's directory (or its row in `index.md`).
2. Pick the file matching the target version exactly.
3. If there is no exact match, use the file with the highest version below the
   target and say the exact version is unverified.
4. If the component or version is not covered at all, say so — do not infer
   telemetry from semantic conventions.

## What a data file contains

Frontmatter (`repo`, `path`, `version`, `scope_name`, `commit_sha`,
`last_verified`), then per-signal tables:

- **Traces** — span name, kind, attributes, notes
- **Metrics** — metric name, instrument type, unit, attributes, notes
- **Logs** — log/event name, attributes, notes

`scope_name` is the component's instrumentation scope — the anchor for
matching runtime telemetry back to the component. When helper or pipeline
telemetry uses another scope, its rows name that scope in `Notes`.
`last_verified: null` means the historical verification date could not be
substantiated; recheck the pinned source before presenting its rows as verified.
Absent signals are omitted
entirely. `Notes` marks conditional emission (feature gates, opt-in config,
experimental semconv). `Sources` lists the upstream files the data came from.
