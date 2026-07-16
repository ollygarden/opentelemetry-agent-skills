# Browser instrumentation catalog

Choosing and configuring browser/RUM instrumentations. Browser telemetry uses **two signal shapes**;
knowing which is which tells you where each instrumentation lives and how to consume the data.

| Model | Signal | Lives in | Examples |
|---|---|---|---|
| **Event-based** | Logs API → `LogRecord` | `@opentelemetry/browser-instrumentation` + js-contrib | navigation, navigation timing, resource timing, web vitals, console, errors, user action |
| **Span-based** | Trace API → spans | `opentelemetry-js` + `opentelemetry-js-contrib` | fetch, XHR, document-load, long-task, user-interaction, react-load |

Many browser signals are *point-in-time facts* ("LCP was 2.1s", "a navigation happened", "an error
was thrown") rather than operations with a duration and children — modeling them as events is
cheaper than forcing a span around them. Spans remain the right model for network requests and work
with a real begin/end and parent/child relationship.

> Exact config options and captured attributes change while these packages are experimental. Confirm
> against the upstream READMEs and `package.json` `exports` (see
> [SKILL.md Sources of Truth](../SKILL.md#sources-of-truth)).

## Event-based instrumentations (`@opentelemetry/browser-instrumentation`)

Entry points are subpath exports under `./experimental/*`; they emit through the global
`LoggerProvider` (see [setup-sdk.md](setup-sdk.md#logger-provider-events)), so call
`logs.setGlobalLoggerProvider(...)` **before** `registerInstrumentations`. (Span-based
instrumentations resolve their tracer at registration time too — pass `tracerProvider` or register
the provider globally first; see [setup-sdk.md](setup-sdk.md#register-the-instrumentations).)

```typescript
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { WebVitalsInstrumentation } from '@opentelemetry/browser-instrumentation/experimental/web-vitals';
import { NavigationInstrumentation } from '@opentelemetry/browser-instrumentation/experimental/navigation';
import { ErrorsInstrumentation } from '@opentelemetry/browser-instrumentation/experimental/errors';
// …also: navigation-timing, resource-timing, user-action, console

registerInstrumentations({
  instrumentations: [
    new WebVitalsInstrumentation(),
    new NavigationInstrumentation(),
    new ErrorsInstrumentation(),
  ],
});
```

### Navigation (`browser.navigation`)

An event for the initial page load (hard navigation) and SPA route changes (soft navigations:
`history.pushState`/`replaceState`, `popstate`, hash changes). This is the reliable
**analytics / user-journey** signal — emitted early, unlike navigation *timing* which finalizes late.

| Option | Type | Default | Description |
|---|---|---|---|
| `useNavigationApiIfAvailable` | `boolean` | `false` | Use the [Navigation API](https://developer.mozilla.org/docs/Web/API/Navigation_API) instead of patching `history.*` (falls back when unavailable). |
| `sanitizeUrl` | `(url: string) => string` | — | Rewrite the URL before it is written to `url.full` — strip tokens, IDs, query params. |
| `applyCustomLogRecordData` | `(logRecord) => void` | — | Mutate the record before emit. Thrown errors are caught + diag-logged. |

Captured attributes include `url.full`, `browser.navigation.same_document` (false = full-page load,
true = SPA route change), `browser.navigation.hash_change`, and `browser.navigation.type`
(`push`/`replace`/`reload`/`traverse`). A `defaultSanitizeUrl` helper strips `user:password@`
credentials and common sensitive query params (`api_key`, `token`, `password`, …); compose your own
on top.

### Navigation Timing (`browser.navigation_timing`)

Detailed page-load milestones (DNS, TCP, TLS, request, response, DOM processing) from
[PerformanceNavigationTiming](https://developer.mozilla.org/docs/Web/API/PerformanceNavigationTiming).
Use for **performance** analysis; pair with the navigation *event* for reliable counts (timing can be
lost if `load` never fires).

### Resource Timing (`browser.resource_timing`)

One event per resource the page loads (scripts, CSS, images, fonts, XHR/fetch) from
[PerformanceResourceTiming](https://developer.mozilla.org/docs/Web/API/PerformanceResourceTiming).
Stays off the main thread (`requestIdleCallback`, Safari `setTimeout` fallback), processes in
batches, captures resources loaded before it was enabled (buffered), and flushes on visibility change.

| Option | Type | Default | Description |
|---|---|---|---|
| `batchSize` | `number` | `50` | Resources processed per batch. |
| `forceProcessingAfter` | `number` | `1000` | Max ms to wait for an idle callback before forcing. |
| `maxProcessingTime` | `number` | `50` | Max ms spent per idle callback. |
| `maxQueueSize` | `number` | `1000` | Queue size before forcing an immediate flush. |
| `initiatorTypes` | `string[]` | (all) | Restrict to specific initiator types, e.g. `['xmlhttprequest', 'fetch']`. |
| `ignoreUrls` | `(string \| RegExp)[]` | — | Drop resource entries whose URL matches. String matching is exact and case-sensitive; prefer non-stateful RegExp filters (no `g`/`y` flags) for robust endpoint filters. |

Captured data: URL, initiator type, total duration, timing phases, transfer/encoded/decoded sizes,
HTTP protocol (h1/h2/h3), redirect timing, service-worker start, render-blocking status (Chromium).
This is one of the **highest-volume** RUM signals — a content-heavy page can load hundreds of
resources. Restrict `initiatorTypes`, set `ignoreUrls` for known-noisy endpoints, or sample
aggressively (see
[performance.md](performance.md#telemetry-volume-and-cost)).

### Web Vitals (`browser.web_vital`)

[Core Web Vitals](https://web.dev/vitals/) via the Google
[`web-vitals`](https://github.com/GoogleChrome/web-vitals) library: **LCP** (loading), **INP**
(responsiveness; replaced FID), **CLS** (visual stability), plus TTFB and FCP. This event's semantic
conventions are **merged** (see the
[WebVital event](https://opentelemetry.io/docs/specs/semconv/browser/browser-events/#webvital-event)).

| Option | Type | Default | Description |
|---|---|---|---|
| `includeRawAttribution` | `boolean` | `false` | Set the record body to the JSON-stringified `web-vitals` attribution object (which element/event caused the metric). |
| `applyCustomLogRecordData` | `(logRecord) => void` | — | Mutate the record before emit. |

INP and CLS finalize near the end of the page lifecycle — they depend on the SDK flushing on
`pagehide`/`visibilitychange`.

### Console (`browser.console`)

Captures console API calls (by default `log`, `warn`, `error`, `info`, `debug`); records carry
`browser.console.method`. The `messageSerializer` option controls how console arguments become the
record body (default: join arguments as strings). Capturing `log`/`info`/`debug` in production is
typically noise and a PII risk — restrict it:

```typescript
new ConsoleInstrumentation({ logMethods: ['error', 'warn'] });
```

### Errors (`exception`)

An `exception` event for every uncaught error (`window` `error`) and unhandled promise rejection
(`unhandledrejection`), reusing the existing `exception` event. Records carry `exception.type`,
`exception.message`, `exception.stacktrace` (type/stacktrace omitted for non-`Error` throws).
When an `ErrorEvent` has no error object but has a non-empty message (as can happen for cross-origin
scripts), the message is still emitted; rejections with a null/undefined reason are dropped.
`applyCustomAttributes` can add fields (e.g. an app-level severity). Failures while extracting or
emitting an exception are contained and reported through SDK diagnostics rather than escaping the
global error handler.

### User Action (`browser.user_action`)

Captures user input events (by default `click`). Any `data-otel-*` attribute on the clicked element
is copied into the `browser.element.attributes` map with the prefix removed — a deliberate channel
for **non-PII** business context.

```typescript
new UserActionInstrumentation({ autoCapturedActions: ['click'] }); // default
```

```html
<button data-otel-feature="signup">Sign Up</button>
```

> `data-otel-*` values are exported verbatim — do not put PII (emails, names) in them.

## Span-based instrumentations (`opentelemetry-js` / `js-contrib`)

These produce **spans** and live outside `opentelemetry-browser`. The easiest on-ramp is the auto
bundle:

```typescript
import { getWebAutoInstrumentations } from '@opentelemetry/auto-instrumentations-web';
// Pass tracerProvider (or register the provider globally BEFORE this call); otherwise these
// span-based instrumentations resolve the no-op tracer at registration time and emit nothing.
registerInstrumentations({
  tracerProvider: provider,
  instrumentations: [getWebAutoInstrumentations()],
});
```

| Package | Repo | What it does |
|---|---|---|
| `instrumentation-fetch` | js | Spans for `fetch()`; injects `traceparent` (configure `propagateTraceHeaderCorsUrls`). |
| `instrumentation-xml-http-request` | js | Spans for `XMLHttpRequest`; same propagation knobs. |
| `instrumentation-document-load` | js-contrib | Spans for document load + navigation/resource timing (span flavor). |
| `instrumentation-user-interaction` | js-contrib | Spans for user interactions (clicks) with their async causal tree. |
| `instrumentation-long-task` | js-contrib | Spans for [Long Tasks](https://developer.mozilla.org/docs/Web/API/Long_Tasks_API) (>50 ms main-thread blocks). |
| `instrumentation-browser-navigation` | js-contrib | Event-based SPA navigation (alternative to the one above). |
| `instrumentation-web-exception` | js-contrib | Event-based unhandled exception capture. |
| `plugin-react-load` | js-contrib | React component mount/load performance; **unmaintained** upstream. |

### fetch / XHR cross-origin propagation

```typescript
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';

new FetchInstrumentation({
  // Required for traceparent to be sent to OTHER origins.
  propagateTraceHeaderCorsUrls: [/api\.example\.com/],
  // Avoid tracing telemetry export calls themselves (prevents feedback loops).
  ignoreUrls: [/\/v1\/(traces|logs)/],
});
```

The server must list `traceparent` (and `tracestate`/`baggage` if used) in
`Access-Control-Allow-Headers`, or the preflight fails. See
[setup-sdk.md](setup-sdk.md#connecting-frontend-to-backend-traces).

## Choosing what to enable

| Goal | Enable |
|---|---|
| Page performance | Web Vitals, Navigation Timing, Resource Timing (span: document-load, long-task) |
| User journeys / analytics | Navigation, User Action |
| Error tracking | Errors, Console (`error`/`warn` only) |
| Frontend↔backend tracing | fetch + XHR span instrumentation with CORS propagation |
| Minimal footprint | Web Vitals + Errors + Navigation (low volume, high value); add the rest deliberately |

## Anti-patterns

| Anti-pattern | Why it's wrong | Fix |
|---|---|---|
| Enabling every instrumentation by default | Resource-timing + console `log` create huge, low-value volume | Start minimal; add deliberately once you understand the volume |
| Capturing `console.log`/`debug` in production | Noise + PII leakage | `logMethods: ['error', 'warn']` |
| Putting PII in `data-otel-*` or URLs | Exported verbatim to your backend | Use `sanitizeUrl`; keep `data-otel-*` to non-PII business keys |
| Tracing your own OTLP export calls | Infinite telemetry-about-telemetry loop | `ignoreUrls` the `/v1/traces` and `/v1/logs` endpoints |
| Forcing spans around point-in-time facts | Wrong model; bloats traces | Use the event-based instrumentations for vitals/navigation/errors |
| Relying on navigation *timing* for page-view counts | Lost when `load` never fires / user leaves early | Use the navigation *event* for counts, timing for performance |
| Assuming an instrumentation works because registration threw no error | These packages are experimental; a misconfigured or no-op instrumentation emits neither errors nor telemetry | Verify each one reaches the Collector (diag logging + `debug` exporter) — see [setup-sdk.md](setup-sdk.md#verify-it-actually-emits) |
