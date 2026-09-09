# Browser instrumentation catalog

Captured against `@opentelemetry/browser-instrumentation` 0.8.1 (2026-09). Verify current exports,
README options, and release notes before relying on experimental behavior.

## Contents

- [Event-based instrumentations](#event-based-instrumentations-opentelemetrybrowser-instrumentation)
- [Span-based instrumentations](#span-based-instrumentations)
- [Choosing what to enable](#choosing-what-to-enable)

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

### Network context correlation (captured through 0.8.1)

Release 0.6.0 added `ContextRegistry` and `NetworkContextRegistry` as a proposal for sharing
OpenTelemetry context between instrumentations that observe the same network operation from
different angles. The network registry indexes a completed span by URL and its `performance.now()`
window, then lets a consumer match that context to a `PerformanceResourceTiming` entry whose
`fetchStart` and `responseEnd` fall inside the window. This is intended to let resource-timing
telemetry retain the network span's trace context.

Release 0.8.0 added consolidated Fetch and XHR instrumentations. They register completed network
span context internally, and Resource Timing consumes matching context automatically. Activate this
correlation by using the 0.8.0 consolidated Fetch/XHR subpath exports together with Resource Timing;
the registry itself remains an internal API and is not an npm subpath export. Do not import its
internal source path, and do not claim this correlation for the separate opentelemetry-js Fetch/XHR
packages.

## Event-based instrumentations (`@opentelemetry/browser-instrumentation`)

Entry points are subpath exports under `./experimental/*`. Set the global `LoggerProvider` before
constructing them, because an enabled instrumentation can emit immediately; `registerInstrumentations`
also accepts `loggerProvider` and rebinds each instrumentation at registration. (Span-based
instrumentations are similarly rebound from `tracerProvider` or the then-current global provider;
see [setup-sdk.md](setup-sdk.md#register-the-instrumentations).)

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
Defers processing to main-thread idle time (`requestIdleCallback`, Safari `setTimeout` fallback),
works in batches, captures resources loaded before it was enabled (buffered), and flushes on
visibility change.

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

Semantic conventions v1.44.0 defines the released `browser.web_vital` event (development
stability; verify via the
`otel-semantic-conventions` skill, group `browser`, entry `event.browser.web_vital`) requires the
`browser.web_vital.name`, `.value`, `.delta`, and `.id` log attributes. The `.rating` and
`.navigation_type` attributes are recommended.

`@opentelemetry/browser-instrumentation` 0.8.1 matches that released shape. Its record body is unset
unless `includeRawAttribution` is enabled, which adds JSON-stringified attribution details outside
the semantic-convention fields. For hand-written reporting, put the web-vital fields in attributes,
not the record body.

| Option | Type | Default | Description |
|---|---|---|---|
| `includeRawAttribution` | `boolean` | `false` | Set the record body to the JSON-stringified `web-vitals` attribution object (which element/event caused the metric). |
| `applyCustomLogRecordData` | `(logRecord) => void` | — | Mutate the record before emit. |

INP and CLS finalize near the end of the page lifecycle — they depend on the SDK flushing on
`pagehide`/`visibilitychange`.

### Console (`browser.console`)

Captures console API calls (by default `log`, `warn`, `error`, `info`, `debug`); records carry
`browser.console.method`. The `messageSerializer` option controls how console arguments become the
record body. By default, objects are JSON-stringified when possible, other values use `String()`,
and the results are joined with spaces. Capturing `log`/`info`/`debug` in production is typically
noise and a PII risk — restrict it:

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

### User Action (`browser.user_action.click`)

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

User-action instrumentation also accepts `applyCustomLogRecordData`; review that hook as
untrusted-data handling and keep any added fields bounded and non-PII.

## Span-based instrumentations

These produce **spans**. The auto bundle contains fetch, XHR, document-load, and user-interaction;
the other packages listed below must be installed separately:

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
| `@opentelemetry/browser-instrumentation/experimental/fetch` | browser | Experimental consolidated Fetch spans; correlates matching Resource Timing events. |
| `@opentelemetry/browser-instrumentation/experimental/xhr` | browser | Experimental consolidated XHR spans; correlates matching Resource Timing events. |
| `instrumentation-fetch` | js | Spans for `fetch()`; injects `traceparent` (configure `propagateTraceHeaderCorsUrls`). |
| `instrumentation-xml-http-request` | js | Spans for `XMLHttpRequest`; same propagation knobs. |
| `instrumentation-document-load` | js-contrib | Spans for document load + navigation/resource timing (span flavor). |
| `instrumentation-user-interaction` | js-contrib | Spans for user interactions (clicks) with their async causal tree. |
| `instrumentation-long-task` | js-contrib | Spans for [Long Tasks](https://developer.mozilla.org/docs/Web/API/Long_Tasks_API) (>50 ms main-thread blocks). |
| `plugin-react-load` | js-contrib | React component mount/load performance; **unmaintained** upstream. |

The js-contrib `instrumentation-browser-navigation` and `instrumentation-web-exception` packages
are separately installed, event-based alternatives to the consolidated Navigation and Errors
instrumentations above; they are not span instrumentations or part of the auto bundle.

The released opentelemetry-js fetch/XHR instrumentations `0.222.0` and js-contrib document-load
`0.67.0` emit only the stable
HTTP semantic conventions. Their `semconvStabilityOptIn` migration option and legacy attributes
(`http.method`, `http.url`, `http.status_code`, …) are gone; query the stable names such as
`http.request.method`, `url.full`, and `http.response.status_code`.

### fetch / XHR cross-origin propagation

```typescript
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';

new FetchInstrumentation({
  // Required for traceparent to be sent to OTHER origins.
  propagateTraceHeaderCorsUrls: [/^https:\/\/api\.example\.com(?:\/|$)/],
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
