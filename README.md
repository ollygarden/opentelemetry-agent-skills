# OpenTelemetry Agent Skills

This repository contains reusable agent skills for OpenTelemetry.

The skills are **non-opinionated by design** — there are many valid ways to use OpenTelemetry, and prescribing conventions is out of scope.

They are designed for **token-efficient, agent-friendly retrieval**: small fetch tables, lookup indexes, and scripts that point at upstream sources of truth instead of copying docs into context. The agent finds what it missed during training; answers stay current as the project evolves.

## Installation

Install any skill using the skills CLI:

```bash
npx skills add ollygarden/opentelemetry-agent-skills
```

Install a specific skill from this repository:

```bash
npx skills add ollygarden/opentelemetry-agent-skills --skill otel-go
```

Replace the skill name with any entry from the table below.

## Repository Structure

Skills are grouped by audience. Each language has a single entry-point skill at the
language root, with task-specific references underneath:

```
skills/
  general/             # language-agnostic skills (otel-declarative-config, otel-sdk-versions, ...)
  go/
    SKILL.md           # name: otel-go
    references/        # declarative-setup, api, instrumentation-libraries, performance, breaking-changes
  java/
    SKILL.md           # name: otel-java
    references/
  js/
    SKILL.md           # name: otel-js
    references/
```

## Available Skills

All skill names carry the `otel-` prefix to declare the topic namespace. Folder names stay
descriptive (no prefix) for readability.

| Skill | Path | Use When |
| --- | --- | --- |
| `otel-declarative-config` | `skills/general/declarative-config/` | Configuring OpenTelemetry SDK providers via a single YAML file (`otelconf`, `OTEL_CONFIG_FILE`, `file_format`). Points at the upstream schema, env-var substitution rules, and configuration precedence. |
| `otel-sdk-versions` | `skills/general/sdk-versions/` | Choosing the latest compatible released OpenTelemetry SDK or package version for a language and finding setup docs or examples. |
| `otel-semantic-conventions` | `skills/general/semantic-conventions/` | Selecting released semantic convention groups, attributes, and span naming rules; checking compliance; looking up exact upstream entries via the bundled query script. |
| `otel-span-events-to-logs-migration` | `skills/general/span-events-to-logs-migration/` | Migrating instrumentation from the deprecated Span Event API (`AddEvent`, `RecordException`) to the Logs API following the OTEP 4430 deprecation plan. |
| `otel-telemetrygen` | `skills/general/telemetrygen/` | Constructing `telemetrygen` commands for generating synthetic traces, metrics, and logs; load-testing collectors; validating OTTL transforms, tail sampling, and filter rules. |
| `otel-weaver` | `skills/general/weaver/` | Authoring an OpenTelemetry Weaver registry, writing Jinja2 templates, generating language bindings, and wiring `weaver registry check`/`generate`/`diff` into CI. |
| `otel-go` | `skills/go/` | OpenTelemetry in Go: declarative SDK setup with `otelconf`, API surface, contrib instrumentation libraries (otelhttp, otelgrpc, etc.), performance tuning, and breaking-change audits. |
| `otel-java` | `skills/java/` | OpenTelemetry in Java: Javaagent zero-code instrumentation, Spring Boot Starter, manual autoconfigure SDK, declarative YAML configuration, and BOM dependency management. |
| `otel-js` | `skills/js/` | OpenTelemetry in Node.js / JavaScript / TypeScript: NodeSDK setup, declarative YAML configuration via `@opentelemetry/configuration`, auto-instrumentations, and ESM vs CJS import patterns. |

## Contributing

Contributions are welcome. When adding or updating skills, please follow these guidelines:

- Keep skills DRY. Prefer referencing official docs, examples, and source code that are already maintained instead of copying large amounts of additional knowledge into the skill. There will be exceptions, but the default should be to link or point to the maintained source of truth.
- Design skills to be token efficient. Avoid dumping large files or broad context into a skill when a targeted lookup, focused reference, or small generated artifact will do.
- Stay vendor neutral.
