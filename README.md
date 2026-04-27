# OpenTelemetry Agent Skills

This repository contains reusable agent skills for OpenTelemetry.

## Installation

Install any skill using the skills CLI:

```bash
npx skills add ollygarden/opentelemetry-agent-skills
```

Install a specific skill from this repository:

```bash
npx skills add ollygarden/opentelemetry-agent-skills --skill manual-instrumentation
```

Replace the skill name with any entry from the table below.

## Repository Structure

Skills are grouped by audience:

```
skills/
  general/   # language-agnostic skills
  go/        # Go-specific skills
```

Additional language folders (`java/`, `js/`, `python/`, …) will be added as skills land.

## Available Skills

| Skill | Path | Use When |
| --- | --- | --- |
| `manual-instrumentation` | `skills/general/manual-instrumentation/` | Planning, adding, or reviewing manual instrumentation; choosing runtime boundaries and signals; configuring SDK defaults; controlling cardinality; handling propagation; and performing final instrumentation review. |
| `semantic-conventions` | `skills/general/semantic-conventions/` | Selecting released semantic convention groups, attributes, and span naming rules; checking semantic convention compliance; and looking up exact upstream semantic convention entries. |
| `sdk-setup` | `skills/general/sdk-setup/` | Setting up or reviewing OpenTelemetry SDK initialization; choosing exporters, processors, propagators, and transport configuration; extending an existing SDK setup for new signals. |
| `sdk-versions` | `skills/general/sdk-versions/` | Choosing the latest compatible released OpenTelemetry SDK or package version for a language and finding setup docs or examples. |
| `span-events-to-logs-migration` | `skills/general/span-events-to-logs-migration/` | Migrating instrumentation from the deprecated Span Event API (AddEvent, RecordException) to the Logs API following the OTEP 4430 deprecation plan. |
| `telemetrygen` | `skills/general/telemetrygen/` | Constructing telemetrygen commands for generating synthetic traces, metrics, and logs; load testing collectors and backends; testing OTTL transforms, tail sampling, and filter rules; multi-tenant and correlated-signal scenarios. |
| `weaver` | `skills/general/weaver/` | Authoring or reviewing an OpenTelemetry Weaver registry; writing Jinja2 templates against the resolved schema; migrating hand-maintained telemetry constants to generated code; wiring `weaver registry check`/`generate`/`diff` into CI. |
| `go-sdk` | `skills/go/sdk/` | Writing, reviewing, or configuring OpenTelemetry instrumentation in Go; looking up current versions, import paths, API surface, contrib libraries, SDK setup, or performance tuning. |

## Contributing

Contributions are welcome. When adding or updating skills, please follow these guidelines:

- Keep skills DRY. Prefer referencing official docs, examples, and source code that are already maintained instead of copying large amounts of additional knowledge into the skill. There will be exceptions, but the default should be to link or point to the maintained source of truth.
- Design skills to be token efficient. Avoid dumping large files or broad context into a skill when a targeted lookup, focused reference, or small generated artifact will do.
- Stay vendor neutral.