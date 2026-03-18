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

## Available Skills

| Skill | Path | Use When |
| --- | --- | --- |
| `opentelemetry-manual-instrumentation` | `skills/opentelemetry-manual-instrumentation/` | Planning, adding, or reviewing manual instrumentation; choosing signals and semantic conventions; naming spans, metrics, and attributes; controlling cardinality; handling propagation; and checking released SDK or semantic convention versions. |
| `span-events-to-logs-migration` | `skills/span-events-to-logs-migration/` | Migrating instrumentation from the deprecated Span Event API (AddEvent, RecordException) to the Logs API following the OTEP 4430 deprecation plan. |


## Contributing

Contributions are welcome. When adding or updating skills, try to follow these guidelines:

- Keep skills DRY. Prefer referencing official docs, examples, and source code that are already maintained instead of copying large amounts of additional knowledge into the skill. There will be exceptions, but the default should be to link or point to the maintained source of truth.
- Design skills to be token efficient. Avoid dumping large files or broad context into a skill when a targeted lookup, focused reference, or small generated artifact will do.
- Stay vendor neutral. Prefer guidance that applies across ecosystems unless a skill is intentionally focused on a specific vendor or tool.