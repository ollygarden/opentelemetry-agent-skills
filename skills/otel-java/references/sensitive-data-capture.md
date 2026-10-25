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
| SQL query text / parameter values | `db.statement` by default; `db.query.text` with stable database semconv; `db.query.parameter.<key>` only when opted in | query text captured but sanitized; parameter values off |

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
  AWSAccessKeyId,Signature,X-Amz-Signature,X-Amz-Credential,\
  X-Amz-Security-Token,sig,X-Goog-Signature
```

- Type: list of case-sensitive parameter names. Setting it **replaces** the default list
  (full override, not additive) — re-list the credential defaults when extending it.
- In Javaagent and Spring Boot Starter **v2.33.0**, the equivalent declarative field is
  `instrumentation/development.general.sanitization.url.sensitive_query_parameters`. It also
  replaces the default list, so include every default that must remain active.
- History: replaces `otel.instrumentation.http.client.experimental.redact-query-parameters`
  (client-only; deprecated, then removed in 2026 releases —
  [#18229](https://github.com/open-telemetry/opentelemetry-java-instrumentation/pull/18229)).

**There is no property that drops the query string entirely.** To export URLs without query
strings, delete/rewrite the attributes in a Collector processor (`transform`/`redaction`), or
customize the HTTP instrumentation's attribute extraction. A `SpanProcessor` cannot reliably
remove them: its end callback receives a read-only span after HTTP attributes have been recorded.

## Opt-in capture knobs (off by default)

```properties
otel.instrumentation.http.server.capture-request-headers=<list>
otel.instrumentation.http.server.capture-response-headers=<list>
otel.instrumentation.http.client.capture-request-headers=<list>
otel.instrumentation.http.client.capture-response-headers=<list>
otel.instrumentation.servlet.experimental.request-parameters.included=<list-of-globs>
otel.instrumentation.servlet.experimental.request-parameters.excluded=<list-of-globs>
# High risk: captures JDBC parameter values and disables query sanitization
otel.instrumentation.jdbc.experimental.capture-query-parameters=true
```

Captured headers land as `http.request.header.<lowercase-name>` /
`http.response.header.<lowercase-name>` (list-valued); servlet parameters as
`servlet.request.parameter.<name>`. Enabling any of these captures the raw values — there
is no per-header/per-parameter value redaction. Servlet selectors are case-sensitive; exclusions
take precedence. With both lists empty, nothing is captured. An exclude-only selector captures
every available parameter not excluded, so prefer an explicit include list. The old
`otel.instrumentation.servlet.experimental.capture-request-parameters` include-only property is
deprecated (checked at 2.33.0) and does not support globs.

Those flat properties apply to normal property-based configuration. When declarative config is
active, flat instrumentation properties are not an overlay; use the corresponding YAML paths:

```yaml
instrumentation/development:
  general:
    http:
      client:
        request_captured_headers: [x-request-id]
        response_captured_headers: [x-request-id]
      server:
        request_captured_headers: [x-request-id]
        response_captured_headers: [x-request-id]
  java:
    servlet:
      request_parameters/development:
        included: [customer-*]
        excluded: [customer-password]
    jdbc:
      # High risk: captures values and disables query sanitization
      capture_query_parameters/development: true
```

## SQL sanitization

SQL literal/parameter sanitization (values → `?`) is on by default; the current toggle is
`java.common.db.query_sanitization.enabled` under `instrumentation/development` in declarative
config, or
`otel.instrumentation.common.db.query-sanitization.enabled` as a system property/environment
variable. Per-instrumentation toggles can take precedence. Older property spellings
(`otel.instrumentation.common.db-statement-sanitizer.enabled` and per-instrumentation
`*-statement-sanitizer.enabled` variants) are deprecated and, when
`instrumentation/development.java.common.v3_preview: true` in Javaagent/Starter 2.33.0, ignored.
Use the `db.query_sanitization.enabled` forms above. Do not turn sanitization off on request paths.

JDBC's parameter-capture switch shown above is a separate opt-in. It emits raw values as
`db.query.parameter.<key>` and disables sanitization even if the sanitization toggle is `true`.

## Sources of truth

| Fact | Fetch |
|---|---|
| HTTP capture properties (headers, servlet params, known-methods) | `WebFetch https://opentelemetry.io/docs/zero-code/java/agent/instrumentation/http/` |
| Current property names/defaults incl. `sensitive-query-parameters` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/instrumentation-docs/src/main/resources/shared-config-definitions.yaml`; per-instrumentation `metadata.yaml` files reference these shared definitions |
| Generated declarative names, defaults, and deprecations | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/docs/declarative-configuration-example.yaml` |
| Renames/removals of capture & sanitization properties | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/CHANGELOG.md` |
| Semconv redaction rules for `url.query`/`url.full` | `WebFetch https://opentelemetry.io/docs/specs/semconv/http/http-spans/` |
