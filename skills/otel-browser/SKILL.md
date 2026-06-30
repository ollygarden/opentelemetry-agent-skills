---
name: otel-browser
description: OpenTelemetry in the browser (Real User Monitoring / RUM) — capturing page loads, Core Web Vitals, route changes, clicks, console output, and JavaScript errors, and connecting frontend telemetry to backend traces. Covers the web tracing SDK (sdk-trace-web, context-zone), the experimental browser-sdk, and event- and span-based browser instrumentations. Use when adding, reviewing, or configuring OpenTelemetry in a web app (SPA/MPA). Triggers on "browser otel", "RUM", "real user monitoring", "frontend observability", "web vitals", "core web vitals", "sdk-trace-web", "WebTracerProvider", "browser-sdk", "browser-instrumentation", "instrument the frontend", "page load tracing", "session", or any browser/web OTel question.
---

# OpenTelemetry in the Browser (RUM)

Entry point for OpenTelemetry mechanics in web apps. Load a reference below based on the task;
each reference is self-contained.

> **Stability**: Browser/RUM is one of the **newest and most experimental** areas of OpenTelemetry.
> Only the web *tracing* primitives (`@opentelemetry/sdk-trace-web`, `@opentelemetry/context-zone`)
> and the JS API are **stable** today. The Browser SDK (`@opentelemetry/browser-sdk`) and the
> event-based instrumentations are experimental and may break between minor versions — pin exact
> versions. Verify current status via the Sources of Truth below.

## References

| File | Use when |
|---|---|
| [`references/setup-sdk.md`](references/setup-sdk.md) | Wiring up the SDK: the direct providers (`WebTracerProvider` + `ZoneContextManager` for spans; `LoggerProvider` for events) vs the experimental `browser-sdk`, sessions, frontend→backend `traceparent`/CORS propagation, and why a Collector sits in front. |
| [`references/instrumentation.md`](references/instrumentation.md) | Choosing and configuring instrumentations: the event-based catalog (navigation, web vitals, console, errors, …), the span-based catalog (fetch, XHR, document-load, long-task, …), per-instrumentation options, and what each captures. |
| [`references/performance.md`](references/performance.md) | Keeping it cheap, fast, and private: bundle size, off-main-thread processing, page-lifecycle flushing, telemetry volume/cost, and PII vectors. |

## Two telemetry models — read first

Browser telemetry is modeled as **spans** and **events**; there is **no browser metrics story
yet** (metrics are out of scope for the Browser SDK). Picking the right model per signal is the
first decision:

| Model | Signal | For | Examples |
|---|---|---|---|
| **Events** | Logs API → `LogRecord` | point-in-time facts (no duration/children) | web vitals, navigation, console, errors, user action |
| **Spans** | Trace API | operations with a duration and parent/child | `fetch`, XHR, document load, long task |

## The browser package ecosystem — three repositories

Browser packages are spread across three upstream repos. The
[`opentelemetry-browser` README "Browser Packages" tables](https://github.com/open-telemetry/opentelemetry-browser#browser-packages)
are the authoritative, current map. Summary:

| Package | Repo | Model | Stability |
|---|---|---|---|
| `@opentelemetry/sdk-trace-web` | opentelemetry-js | SDK (spans) | **stable** |
| `@opentelemetry/context-zone` | opentelemetry-js | context | **stable** |
| `@opentelemetry/instrumentation-fetch` | opentelemetry-js | spans | experimental |
| `@opentelemetry/instrumentation-xml-http-request` | opentelemetry-js | spans | experimental |
| `@opentelemetry/browser-detector` | opentelemetry-js | resource | experimental |
| `@opentelemetry/browser-instrumentation` | opentelemetry-browser | events | experimental |
| `@opentelemetry/browser-sdk` | opentelemetry-browser | SDK | experimental (unreleased) |
| `@opentelemetry/auto-instrumentations-web` | opentelemetry-js-contrib | bundle | experimental |
| `instrumentation-document-load` / `-long-task` / `-user-interaction` | opentelemetry-js-contrib | spans | experimental |
| `instrumentation-browser-navigation` / `-web-exception` | opentelemetry-js-contrib | events | experimental |
| `plugin-react-load` | opentelemetry-js-contrib | spans | experimental |

`opentelemetry-browser` is the home of the event-based instrumentations and the future home of the
Browser SDK; the span-based packages still live in `opentelemetry-js` / `opentelemetry-js-contrib`.

## Why browser RUM is different

These constraints drive most design decisions (detailed in the references):

- **The process disappears** — no graceful shutdown. Flush on `visibilitychange`/`pagehide` with
  `keepalive`, not `unload`.
- **No gRPC** — export is **OTLP/HTTP** only (protobuf or JSON).
- **Cross-origin propagation is opt-in** — `traceparent` is not sent to other origins unless you set
  `propagateTraceHeaderCorsUrls` **and** the server allows the header via `Access-Control-Allow-Headers`.
- **The client is untrusted and chatty** — put a **Collector (or vendor edge)** between browsers and
  your backend for CORS termination, sampling, redaction, and rate limiting.
- **PII is everywhere** — URLs, console output, form fields, and click targets routinely carry it.

## Sources of Truth

Browser packages move fast while experimental — fetch current versions and status rather than
relying on these notes.

| Fact | Fetch |
|---|---|
| `opentelemetry-browser` package versions / status | `gh api repos/open-telemetry/opentelemetry-browser/releases -q '.[].tag_name'` |
| Latest `@opentelemetry/browser-instrumentation` | `npm view @opentelemetry/browser-instrumentation version` |
| Latest `@opentelemetry/browser-sdk` (may be unpublished) | `npm view @opentelemetry/browser-sdk version` |
| Latest `@opentelemetry/sdk-trace-web` | `npm view @opentelemetry/sdk-trace-web version` |
| Latest `@opentelemetry/auto-instrumentations-web` | `npm view @opentelemetry/auto-instrumentations-web version` |
| Authoritative browser package map | `WebFetch https://github.com/open-telemetry/opentelemetry-browser#browser-packages` |
| `browser-instrumentation` README / config | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-browser/main/packages/instrumentation/README.md` |
| `browser.*` event semantic-convention status | `WebFetch https://opentelemetry.io/docs/specs/semconv/browser/` |

## Cross-References

- Shared JS API and Node.js SDK (the browser builds on the same API): `otel-js` skill.
- Schema-level facts for declarative YAML config: `otel-declarative-config` skill.
- Semantic conventions lookup (`browser.*`, `session.*`, `exception`): `otel-semantic-conventions` skill.
- Edge sampling / redaction / rate limiting in front of browsers: `otel-collector` skill.
- SDK version selection across languages: `otel-sdk-versions` skill.
