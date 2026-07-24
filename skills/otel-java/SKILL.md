---
name: otel-java
description: OpenTelemetry in Java — Javaagent zero-code instrumentation, Spring Boot Starter, manual autoconfigure SDK, declarative YAML configuration, BOM dependency management, sensitive-data capture and redaction (url.query, headers, request parameters, SQL sanitization). Use when adding, reviewing, or configuring OpenTelemetry in a Java service. Triggers on "setup otel in java", "java telemetry", "javaagent", "Spring Boot otel", "GlobalOpenTelemetry", "AutoConfiguredOpenTelemetrySdk", "TracerProvider java", "url.query redaction", "capture request headers", or any Java-related OTel question.
---

# OpenTelemetry in Java

Entry point for OpenTelemetry mechanics in Java services. Load a reference below based on
the task; each reference is self-contained.

## References

| File | Use when |
|---|---|
| [`references/declarative-setup.md`](references/declarative-setup.md) | Configuring the SDK via declarative YAML: Javaagent activation, Spring Boot Starter, autoconfigure SDK, BOM, agent-only properties, manual instrumentation entry points. |
| [`references/sensitive-data-capture.md`](references/sensitive-data-capture.md) | What HTTP instrumentation captures by default (query strings ON, headers/params OFF), query-parameter redaction (`sensitive-query-parameters`), header/servlet-parameter capture knobs, SQL sanitization. |

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config` skill.
For Java-specific facts:

| Fact | Fetch |
|---|---|
| Latest BOM (`opentelemetry-bom`) | `gh api repos/open-telemetry/opentelemetry-java/releases/latest -q '.tag_name'` |
| Latest Javaagent | `gh api repos/open-telemetry/opentelemetry-java-instrumentation/releases/latest -q '.tag_name'` |
| SDK declarative-config accepted and preferred `file_format` for a selected BOM tag | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java/<selected-sdk-tag>/sdk-extensions/declarative-config/src/main/java/io/opentelemetry/sdk/autoconfigure/declarativeconfig/OpenTelemetryConfigurationFactory.java` |
| Javaagent declarative-config docs (current activation flag, supported `file_format`) | `WebFetch https://opentelemetry.io/docs/zero-code/java/agent/declarative-configuration/` |
| Javaagent declarative-config smoke fixture (parser truth for selected agent tag) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/smoke-tests/src/test/resources/declarative-config.yaml` |
| Javaagent CHANGELOG (when each schema rc landed) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/CHANGELOG.md` |
| Spring Boot Starter declarative-config fixture (selected starter tag) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/smoke-tests-otel-starter/spring-boot-2/src/testDeclarativeConfig/resources/application.yaml` |
| Spring Boot starter docs | `WebFetch https://opentelemetry.io/docs/zero-code/java/spring-boot-starter/` |
| Per-instrumentation telemetry & config (resolved spans/attributes, metrics, config options, target versions) | `WebFetch https://explorer.opentelemetry.io/data/javaagent/instrumentations/<id>/<id>-<hash>.json` — get `<id>` and the latest `<hash>` from the index (see [below](#what-telemetry-does-an-instrumentation-emit)) |

## What telemetry does an instrumentation emit?

For *"what does the agent produce for library X"* — which spans, attributes, metrics, or config
knobs — use the [OpenTelemetry Ecosystem Explorer](https://explorer.opentelemetry.io/), which fully
maps the Java agent and exposes an agent-friendly surface (Markdown indexes and resolved JSON, no
scraping). Do **not** answer from model memory.

Navigation:

1. `WebFetch https://explorer.opentelemetry.io/agent/javaagent/index.md` — table mapping display
   name → `id` → the instrumentation's JSON data URL (the URL embeds the content `<hash>`).
2. `WebFetch` that JSON URL — one self-contained record (a few KB): resolved `configurations`,
   `telemetry` (spans with `span_kind` + typed attributes, and `metrics`),
   `javaagent_target_versions`, `semantic_conventions`, and `scope`.

Version-specific or *"what changed between releases"*:
`https://explorer.opentelemetry.io/agent/javaagent/versions.md` lists versions and marks the latest;
`https://explorer.opentelemetry.io/data/javaagent/versions/<version>-index.json` gives the
`id`→`hash` map for a version — a differing hash for the same `id` across two versions means that
instrumentation changed. Schema:
`https://explorer.opentelemetry.io/schemas/javaagent-instrumentation.schema.json`; use
`/llms.txt` for the agent-oriented index and `/llms-full.txt` for the full documentation.

Prefer this Explorer data over the raw `ecosystem-registry` YAML on GitHub: the Explorer applies
upstream metadata corrections that the raw registry does not.

## Cross-References

- Schema-level facts: `otel-declarative-config` skill (language-agnostic YAML schema sources).
- SDK version selection across languages: `otel-sdk-versions` skill.
- Semantic conventions lookup: `otel-semantic-conventions` skill.
