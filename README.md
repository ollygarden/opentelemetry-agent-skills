# OpenTelemetry Agent Skills

This repository contains reusable agent skills for OpenTelemetry.

## Installation

Install any skill using the skills CLI:

```bash
npx skills add niwoerner/opentelemetry-agent-skills
```

Install a specific skill from this repository:

```bash
npx skills add niwoerner/opentelemetry-agent-skills --skill opentelemetry-manual-instrumentation
```

Replace the skill name with any entry from the table below.

## Available Skills

| Skill | Path | Use When |
| --- | --- | --- |
| `opentelemetry-manual-instrumentation` | `skills/opentelemetry-manual-instrumentation/` | Planning, adding, or reviewing manual instrumentation; choosing runtime boundaries and signals; configuring SDK defaults; controlling cardinality; handling propagation; and performing final instrumentation review. |
| `opentelemetry-semantic-conventions` | `skills/opentelemetry-semantic-conventions/` | Selecting released semantic convention groups, attributes, and span naming rules; checking semantic convention compliance; and looking up exact upstream semantic convention entries. |
| `opentelemetry-sdk-versions` | `skills/opentelemetry-sdk-versions/` | Choosing the latest compatible released OpenTelemetry SDK or package version for a language and finding setup docs or examples. |
| `span-events-to-logs-migration` | `skills/span-events-to-logs-migration/` | Migrating instrumentation from the deprecated Span Event API (AddEvent, RecordException) to the Logs API following the OTEP 4430 deprecation plan. |

## Contributing

Contributions are welcome. When adding or updating skills, please follow these guidelines:

- Keep skills DRY. Prefer referencing official docs, examples, and source code that are already maintained instead of copying large amounts of additional knowledge into the skill. There will be exceptions, but the default should be to link or point to the maintained source of truth.
- Design skills to be token efficient. Avoid dumping large files or broad context into a skill when a targeted lookup, focused reference, or small generated artifact will do.
- Stay vendor neutral.