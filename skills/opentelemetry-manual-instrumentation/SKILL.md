---
name: opentelemetry-manual-instrumentation
description: OpenTelemetry best practices for manual instrumentation. Use when adding, changing, or reviewing OpenTelemetry instrumentation in code. Guidance to choose runtime boundaries, choose signals, apply semantic conventions, handle propagation, control cardinality, and verify the result.
---

# OpenTelemetry Manual Instrumentation

Use this skill for manual instrumentation design, implementation, and review.

Usage:
- load the bundled references only when they apply
- use `opentelemetry-sdk-versions` for version selection when available; otherwise use official release sources, setup docs, and source code
- use `opentelemetry-semantic-conventions` for semantic convention naming and attributes when available; otherwise use the upstream semantic convention docs via websearch

If a companion skill is unavailable:
- do not stop
- do not rely on memory alone when the guidance can be checked from official sources
- use the corresponding official OpenTelemetry release sources, documentation, examples, or source code
- state that the companion skill was unavailable
- cite the fallback source used
- leave any unverified item unresolved rather than guessing

## Non-Negotiable Rules

- Use the latest compatible OpenTelemetry SDK or package for the project language. If the latest release is not compatible, use the latest compatible version and state the reason. When SDK behavior, examples, or setup details are unclear, check official SDK docs and source code instead of relying on model memory.
- Instrument meaningful runtime boundaries only. Do not create spans for helpers, loop bodies, validation-only code, serialization glue, getters, or pure computation. See `references/boundaries.md`.
- Choose the signal before writing code. Use spans for request or operation flow, metrics for aggregate recurring questions, logs or events for discrete diagnostic facts, and nothing when telemetry would be noise. See `references/signal-selection.md`.
- For any known boundary type such as `http`, `db`, `messaging`, or `rpc`, check the released semantic conventions before choosing names or attributes. Derive semconv-governed span names directly from the released naming rule. Do not add custom prose, protocol labels, hostnames, product names, or business hints to semconv-governed span names. If no released key exists, use a stable custom namespace and keep values bounded.
- Keep values bounded. Prefer method, route template, status code, operation name, destination name, region, deployment environment, or customer tier. Avoid raw user IDs, full URLs, raw SQL in metrics, free-form messages, and timestamps as dimensions.
- Record failure on the span that owns the final failed outcome. Do not mark the final span as failed if retries succeed. Do not treat handled errors as terminal failures. See `references/boundaries.md`.
- Preserve trace context across network and async boundaries. Extract inbound context, inject outbound context, and use baggage only when it is intentional and bounded. See `references/propagation.md`.

## Workflow

1. Identify the task mode: planning, implementation, or review.
2. For each instrumentation point, decide:
   - the runtime boundary
   - the signal
   - the semantic convention group if one exists
   - the key attributes and cardinality risks
   - the propagation requirements
3. Implement or review the code.
4. Re-open the changed files and verify the result with evidence.

## SDK Defaults

Use these defaults unless the project already has an intentional compatible alternative:
- traces: OTLP exporter plus batch span processor
- metrics: OTLP exporter plus periodic exporting metric reader
- logs: OTLP exporter plus batching log record processor
- propagators: `tracecontext,baggage`
- protocol: prefer the SDK default transport; if choosing explicitly, prefer `http/protobuf` unless the SDK or project requires `grpc`

During implementation:
- if no SDK is set up yet, configure one for the signals in scope
- if an SDK is already present, reuse and extend it instead of replacing it
- preserve existing exporter, processor, and transport choices when they are already intentional and compatible
- do not add extra signals just because the SDK can support them

## References To Load On Demand

- boundary choice and error ownership: `references/boundaries.md`
- signal choice: `references/signal-selection.md`
- propagation and baggage: `references/propagation.md`
- semantic conventions naming and attributes skill: `opentelemetry-semantic-conventions`
- SDK versions and docs skill: `opentelemetry-sdk-versions`

## Verification Contract

If you changed code or performed an instrumentation review:
- re-open the changed files before finishing
- confirm each applicable item with codebase evidence
- mark non-applicable items explicitly
- do not mark an item complete based on intent alone

Report the final check with:
- `[x]` completed
- `[~]` not applicable, with a reason
- `[ ]` unresolved

Use these items:
- meaningful boundary
- latest compatible SDK or package decision
- SDK setup uses common defaults or preserves an intentional alternative
- released semantic conventions were checked for each known boundary type
- names and attributes match released semantic conventions, or a concrete compatibility limitation is stated
- signal choice is intentional
- cardinality is bounded
- error ownership matches the final failed outcome
- propagation is handled across relevant network or async boundaries, or is not applicable
- changed files were re-read
- remaining risks or gaps are stated
