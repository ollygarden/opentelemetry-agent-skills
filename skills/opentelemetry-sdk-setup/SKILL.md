---
name: opentelemetry-sdk-setup
description: OpenTelemetry SDK initialization and configuration. Use when setting up or reviewing TracerProvider, MeterProvider, or LoggerProvider; choosing exporters, processors, or propagators; configuring OTLP transport; or extending an existing SDK setup for new signals. Use this skill whenever the task involves wiring up the OpenTelemetry SDK, even if the user only mentions "add tracing" or "set up metrics" without saying "SDK."
---

# OpenTelemetry SDK Setup

Use this skill when configuring or reviewing OpenTelemetry SDK initialization.

Usage:
- use `opentelemetry-sdk-versions` for version selection when available; otherwise use official release sources, setup docs, and source code
- use `opentelemetry-manual-instrumentation` for what to instrument once the SDK is set up

If a companion skill is unavailable:
- do not stop
- do not rely on memory alone when the guidance can be checked from official sources
- use the corresponding official OpenTelemetry release sources, documentation, examples, or source code
- state that the companion skill was unavailable
- cite the fallback source used
- leave any unverified item unresolved rather than guessing

## Non-Negotiable Rules

- Use the latest compatible OpenTelemetry SDK or package for the project language. Delegate version choice to `opentelemetry-sdk-versions` or official release sources. When SDK behavior, setup details, or API surface is unclear, check official SDK docs and source code instead of relying on model memory.
- If no SDK is set up yet, configure one for the signals in scope. Do not add signals beyond what the current task requires — each signal adds export, batching, and lifecycle overhead that is wasted if nothing produces telemetry for it.
- If an SDK is already present, reuse and extend it instead of replacing it. Adding a new signal to an existing setup is almost always safer than rewriting initialization.
- Preserve existing exporter, processor, and transport choices when they are already intentional and compatible. Changing a working pipeline without a concrete reason risks breaking collection.

## Defaults

Use these unless the project already has an intentional compatible alternative:

- traces: OTLP exporter plus batch span processor
- metrics: OTLP exporter plus periodic exporting metric reader
- logs: OTLP exporter plus batching log record processor
- propagators: `tracecontext,baggage`
- protocol: prefer the SDK default transport; if choosing explicitly, prefer `http/protobuf` unless the SDK or project requires `grpc`

These defaults align with the OTLP-first direction of the OpenTelemetry project and work out of the box with any OTLP-compatible backend.

## Workflow

1. Detect whether the project already has an SDK setup (look for provider initialization, exporter registration, or a dedicated telemetry bootstrap file).
2. Determine which signals are in scope for the current task (traces, metrics, logs, or a combination).
3. If no setup exists: configure providers, exporters, and processors for in-scope signals using the defaults above.
4. If a setup exists: verify it covers in-scope signals. Extend if a needed signal is missing. Do not replace intentional choices.
5. Verify the configuration matches the defaults or has an intentional reason to differ.
6. Re-open the changed files and verify the result with evidence.

## Verification Contract

If you changed or reviewed SDK setup code:
- re-open the changed files before finishing
- confirm each applicable item with codebase evidence
- mark non-applicable items explicitly
- do not mark an item complete based on intent alone

Report the final check with:
- `[x]` completed
- `[~]` not applicable, with a reason
- `[ ]` unresolved

Use these items:
- providers configured for in-scope signals only
- exporters use OTLP or preserve an intentional existing alternative
- processors use batch or periodic, or preserve an intentional existing alternative
- propagators are `tracecontext,baggage` or preserve an intentional existing alternative
- transport uses SDK default or `http/protobuf`, or preserves an intentional existing alternative
- no extra signals added beyond what the task requires
- existing SDK setup reused and extended, not replaced (when one was already present)
- changed files were re-read
- remaining risks or gaps are stated
