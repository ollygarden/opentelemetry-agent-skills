---
name: span-events-to-logs-migration
description: Migrate OpenTelemetry Span Events (AddEvent, RecordException) to the Logs API following the OTEP 4430 deprecation plan. Use when migrating instrumentation from span events to log-based events, reviewing code that still uses AddEvent or RecordException, or planning a migration across a codebase.
---

# Span Events to Logs Migration

Use this skill to migrate instrumentation from the deprecated Span Event API (`AddEvent`, `RecordException`) to the Logs API, following the [OTEP 4430 deprecation plan](https://github.com/open-telemetry/opentelemetry-specification/blob/main/oteps/4430-span-event-api-deprecation-plan.md).

## Background

The OpenTelemetry project is deprecating `Span.AddEvent` and `Span.RecordException` in favor of emitting events and exceptions through the Logs API. The Span Event API is deprecated, but Span Events as a concept remain valid -- they are now emitted via logs that correlate to the active span.

See `references/deprecation-plan.md` for the full context.

## Workflow

0. Prepare before migrating.
- check the project's OpenTelemetry SDK version supports log-based events (check the `opentelemetry-manual-instrumentation` skill's version index if available)
- identify whether the project has a LoggerProvider configured; if not, one must be set up
- determine if downstream consumers (backends, dashboards, alerts) depend on span events appearing in the span proto envelope

1. Scan the codebase for span event usage.
- search for `AddEvent`, `add_event`, `addEvent`, `RecordException`, `record_exception`, `recordException`, and language-specific variants
- categorize each call site: general event, exception recording, or informational annotation
- note the span context, attributes, and timestamp usage at each site

2. Classify each call site using the decision tree.
- see `references/decision-tree.md` for the full classification logic
- the three outcomes are: migrate to log-based event, convert to span attributes, or remove

3. Apply the migration for each call site.
- see `references/migration-patterns.md` for language-specific before/after patterns
- ensure the replacement log record carries the correct span context, event name, attributes, and timestamp
- for exceptions, ensure `exception.type`, `exception.message`, and `exception.stacktrace` attributes are preserved

4. If backward compatibility is needed, configure the SDK bridge.
- see `references/backward-compat.md`
- this is an SDK-level log processor that converts log-based events back to span events
- only needed when downstream systems require span events in the same proto envelope as the span

5. Verify the migration.

## Required Completion Loop

Follow this loop every time:
1. scan and classify all span event call sites
2. migrate each call site following the decision tree and patterns
3. review the changed code against the checklist below
4. re-open the changed files and confirm each checklist item with codebase evidence
5. if any item is unresolved, patch the code or mark it not applicable with a reason, then repeat the review
6. do not finish until every checklist item is completed or explicitly marked not applicable

Do not mark a checklist item complete based on intent alone. Mark it complete only after confirming it in the current codebase.

## Migration Checklist

For every item, report one of these statuses in the final answer:
- `[x]` completed
- `[~]` not applicable, with a reason
- `[ ]` unresolved

Include file references as evidence for every completed item.

- `[ ]` All `AddEvent` / `add_event` / `addEvent` call sites identified and classified.
- `[ ]` All `RecordException` / `record_exception` / `recordException` call sites identified and classified.
- `[ ]` A LoggerProvider is configured in the SDK setup (or already existed).
- `[ ]` Each migrated event uses the Logs API with the correct event name and attributes.
- `[ ]` Each migrated exception preserves `exception.type`, `exception.message`, and `exception.stacktrace`.
- `[ ]` Migrated log records carry the active span context for trace correlation.
- `[ ]` Call sites classified as "convert to span attributes" now use span attributes instead.
- `[ ]` Call sites classified as "remove" have been removed with justification.
- `[ ]` Backward compatibility bridge is configured if downstream systems require span events in the span envelope.
- `[ ]` No remaining references to the deprecated `AddEvent` or `RecordException` APIs unless intentionally kept for the current major version.
- `[ ]` The changed files were re-read after implementation to verify the final state.
- `[ ]` The final answer includes this checklist, file evidence, and any remaining risks or gaps.

## Final Review Format

In the final answer, include the checklist in this format:
- `[x]` LoggerProvider configured. Evidence: `src/telemetry/setup.go:42` -- added OTLP log exporter with batch processor.
- `[~]` Backward compatibility bridge. Reason: no downstream systems depend on span events in the proto envelope.
- `[ ]` Exception migration. Missing evidence; re-check required.
