# Sensitive-data capture in Java HTTP instrumentation

What the Javaagent and Spring Boot Starter capture from HTTP traffic by default, and every
knob that changes it. Applies to both distribution channels — they share the same
instrumentation config properties (env-var form: upper-case, dots/dashes → underscores).

## What is captured by default

| Data | Attribute | Default |
|---|---|---|
| URL path | `url.path` (server) / inside `url.full` (client) | **captured** |
| URL query string | `url.query` (server) / inside `url.full` (client) | **captured**, with only the sensitive-parameter list below redacted |
| Request/response headers | `http.request.header.<name>` / `http.response.header.<name>` | not captured (opt-in) |
| Servlet request parameters (form/query) | `servlet.request.parameter.<name>` | not captured (opt-in, experimental) |
| SQL bound values | `db.query.text` | not captured (statements sanitized to `?` by default) |

The asymmetry matters: headers, request parameters, and SQL values are **off** by default,
but the raw query string is **on** — any user data in a query string
(`GET /owners?lastName=Smith`, search terms, tokens in links) is exported verbatim unless
its parameter name is in the redaction list.

## Query-parameter redaction

Values of listed parameter names are replaced with `REDACTED` in `url.query` and `url.full`
(parameter names themselves are preserved, per semconv):

```properties
# default list — credential parameters only
otel.instrumentation.sanitization.url.experimental.sensitive-query-parameters=\
  AWSAccessKeyId,Signature,sig,X-Goog-Signature
```

- Type: list of case-sensitive parameter names. Setting it **replaces** the default list
  (full override, not additive) — re-list the credential defaults when extending it.
- Declarative config: `general.sanitization.url.sensitive_query_parameters` under
  `instrumentation/development`.
- History: replaces `otel.instrumentation.http.client.experimental.redact-query-parameters`
  (client-only; deprecated, then removed in 2026 releases —
  [#18229](https://github.com/open-telemetry/opentelemetry-java-instrumentation/pull/18229)).

**There is no property that drops the query string entirely.** To export URLs without query
strings, post-process: overwrite `url.query`/`url.full` in a `SpanProcessor` (registered via
the autoconfigure SPI, or an agent extension jar for the Javaagent), or delete/rewrite the
attributes in a Collector processor (`transform`/`redaction`).

## Opt-in capture knobs (off by default)

```properties
otel.instrumentation.http.server.capture-request-headers=<list>
otel.instrumentation.http.server.capture-response-headers=<list>
otel.instrumentation.http.client.capture-request-headers=<list>
otel.instrumentation.http.client.capture-response-headers=<list>
otel.instrumentation.servlet.experimental.capture-request-parameters=<list>
```

Captured headers land as `http.request.header.<lowercase-name>` /
`http.response.header.<lowercase-name>` (list-valued); servlet parameters as
`servlet.request.parameter.<name>`. Enabling any of these captures the raw values — there
is no per-header/per-parameter redaction.

## SQL sanitization

Statement sanitization (bound values → `?`) is on by default; the current toggle is
`db.query_sanitization.enabled` in declarative config. Older property spellings
(`otel.instrumentation.common.db-statement-sanitizer.enabled` and per-instrumentation
`*-statement-sanitizer.enabled` variants) are deprecated. Do not turn it off on request
paths.

## Sources of truth

| Fact | Fetch |
|---|---|
| HTTP capture properties (headers, servlet params, known-methods) | `WebFetch https://opentelemetry.io/docs/zero-code/java/agent/instrumentation/http/` |
| Current property names/defaults incl. `sensitive-query-parameters` | any instrumentation `metadata.yaml`, e.g. `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/main/instrumentation/jodd-http-4.2/metadata.yaml` |
| Renames/removals of capture & sanitization properties | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/main/CHANGELOG.md` |
| Semconv redaction rules for `url.query`/`url.full` | `WebFetch https://opentelemetry.io/docs/specs/semconv/http/http-spans/` |
