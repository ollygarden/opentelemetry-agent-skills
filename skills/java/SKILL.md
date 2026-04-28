---
name: otel-java
description: OpenTelemetry in Java — Javaagent zero-code instrumentation, Spring Boot Starter, manual autoconfigure SDK, declarative YAML configuration, BOM dependency management. Use when adding, reviewing, or configuring OpenTelemetry in a Java service. Triggers on "setup otel in java", "java telemetry", "javaagent", "Spring Boot otel", "GlobalOpenTelemetry", "AutoConfiguredOpenTelemetrySdk", "TracerProvider java", or any Java-related OTel question.
---

# OpenTelemetry in Java

Entry point for OpenTelemetry mechanics in Java services. Load a reference below based on
the task; each reference is self-contained.

## References

| File | Use when |
|---|---|
| [`references/declarative-setup.md`](references/declarative-setup.md) | Configuring the SDK via declarative YAML: Javaagent activation, Spring Boot Starter, autoconfigure SDK, BOM, agent-only properties, manual instrumentation entry points. |

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config` skill.
For Java-specific facts:

| Fact | Fetch |
|---|---|
| Latest BOM (`opentelemetry-bom`) | `gh api repos/open-telemetry/opentelemetry-java/releases/latest -q '.tag_name'` |
| Latest Javaagent | `gh api repos/open-telemetry/opentelemetry-java-instrumentation/releases/latest -q '.tag_name'` |
| Javaagent declarative-config docs (current activation flag, supported `file_format`) | `WebFetch https://opentelemetry.io/docs/zero-code/java/agent/configuration/` |
| Javaagent CHANGELOG (when each schema rc landed) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/main/CHANGELOG.md` |
| Spring Boot starter docs | `WebFetch https://opentelemetry.io/docs/zero-code/java/spring-boot-starter/` |

## Cross-References

- Schema-level facts: `otel-declarative-config` skill (language-agnostic YAML schema sources).
- SDK version selection across languages: `otel-sdk-versions` skill.
- Semantic conventions lookup: `otel-semantic-conventions` skill.
