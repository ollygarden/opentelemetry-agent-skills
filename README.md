# OpenTelemetry Agent Skills

This repository contains reusable agent skills for OpenTelemetry.

The skills are **non-opinionated by design** — there are many valid ways to use OpenTelemetry, and prescribing conventions is out of scope.

They are designed for **token-efficient, agent-friendly retrieval**: small fetch tables, lookup indexes, and scripts that point at upstream sources of truth instead of copying docs into context. The agent finds what it missed during training; answers stay current as the project evolves.

## Installation

### skills.sh

Install all skills:

```bash
npx skills add https://github.com/ollygarden/opentelemetry-agent-skills
```

Or install a single skill by pointing at its folder, e.g.:

```bash
npx skills add https://github.com/ollygarden/opentelemetry-agent-skills/tree/main/skills/otel-go
```

### Claude Code

1. Register the repository as a plugin marketplace:

   ```
   /plugin marketplace add ollygarden/opentelemetry-agent-skills
   ```

2. Install a skill:

   ```
   /plugin install <skill-name>@opentelemetry-agent-skills
   ```

   For example:

   ```
   /plugin install otel-go@opentelemetry-agent-skills
   ```

## Repository Structure

Each skill is a self-contained folder under `skills/`, named to match its `name:` field per
the [agentskills.io specification](https://agentskills.io/specification). Language-specific
skills bundle task-focused references; language-agnostic skills sit alongside them.

```
skills/
  otel-go/
    SKILL.md           # name: otel-go
    references/        # declarative-setup, api, instrumentation-libraries, performance, breaking-changes
  otel-java/
    SKILL.md
    references/
  otel-js/
    SKILL.md
    references/
  otel-python/
    SKILL.md
    references/
  otel-dotnet/
    SKILL.md
    references/
  otel-browser/
    SKILL.md
    references/        # setup-sdk, instrumentation, performance
  otel-collector/
    SKILL.md
    components/        # one directory per Collector component (log_dedup, interval, …)
  otel-declarative-config/
  otel-ottl/
  otel-sdk-versions/
  otel-semantic-conventions/
  otel-span-events-to-logs-migration/
  otel-telemetrygen/
  otel-weaver/
```

## Available Skills

Language-agnostic skills:

| Skill | Path | Use When |
| --- | --- | --- |
| `otel-collector` | `skills/otel-collector/` | Configuring OpenTelemetry Collector components — config keys, defaults, validation, signal support, stability, and gotchas. Progressive disclosure via `components/<type>/README.md` plus on-demand detail files. |
| `otel-declarative-config` | `skills/otel-declarative-config/` | Configuring OpenTelemetry SDK providers via a single YAML file (`otelconf`, `OTEL_CONFIG_FILE`, `file_format`). Points at the upstream schema, env-var substitution rules, and configuration precedence. |
| `otel-ottl` | `skills/otel-ottl/` | Authoring or reviewing OTTL statements for `transform`, `filter`, `routing`, and `tail_sampling` processors; debugging OTTL syntax and semantics; transforming traces, metrics, logs, and profiles in the Collector. |
| `otel-sdk-versions` | `skills/otel-sdk-versions/` | Choosing the latest compatible released OpenTelemetry SDK or package version for a language and finding setup docs or examples. |
| `otel-semantic-conventions` | `skills/otel-semantic-conventions/` | Selecting released semantic convention groups, attributes, and span naming rules; checking compliance; looking up exact upstream entries via the bundled query script. |
| `otel-span-events-to-logs-migration` | `skills/otel-span-events-to-logs-migration/` | Migrating instrumentation from the deprecated Span Event API (`AddEvent`, `RecordException`) to the Logs API following the OTEP 4430 deprecation plan. |
| `otel-telemetrygen` | `skills/otel-telemetrygen/` | Constructing `telemetrygen` commands for generating synthetic traces, metrics, and logs; load-testing collectors; validating OTTL transforms, tail sampling, and filter rules. |
| `otel-weaver` | `skills/otel-weaver/` | Authoring an OpenTelemetry Weaver registry, writing Jinja2 templates, generating language bindings, and wiring `weaver registry check`/`generate`/`diff` into CI. |

Language-specific skills:

| Skill | Path | Use When |
| --- | --- | --- |
| `otel-go` | `skills/otel-go/` | OpenTelemetry in Go: declarative SDK setup with `otelconf`, API surface, contrib instrumentation libraries (otelhttp, otelgrpc, etc.), performance tuning, and breaking-change audits. |
| `otel-java` | `skills/otel-java/` | OpenTelemetry in Java: Javaagent zero-code instrumentation, Spring Boot Starter, manual autoconfigure SDK, declarative YAML configuration, and BOM dependency management. |
| `otel-js` | `skills/otel-js/` | OpenTelemetry in Node.js / JavaScript / TypeScript: NodeSDK setup, declarative YAML configuration via `@opentelemetry/configuration`, auto-instrumentations, and ESM vs CJS import patterns. |
| `otel-python` | `skills/otel-python/` | OpenTelemetry in Python: declarative SDK setup via file config, API surface and logging bridge, zero-code instrumentation (opentelemetry-distro / opentelemetry-instrument) and contrib catalog, performance tuning, breaking-change audits. |
| `otel-dotnet` | `skills/otel-dotnet/` | OpenTelemetry in .NET: DI/builder SDK setup (`OpenTelemetry.Extensions.Hosting`), native BCL instrumentation (`ActivitySource`, `Meter`, `ILogger`), zero-code CLR-profiler agent and contrib instrumentation catalog, performance tuning, breaking-change audits. |
| `otel-browser` | `skills/otel-browser/` | OpenTelemetry in the browser (Real User Monitoring / RUM): web tracing SDK (`sdk-trace-web`, `context-zone`) and experimental `browser-sdk`, event- and span-based browser instrumentations (web vitals, navigation, errors, fetch/XHR), sessions, frontend→backend trace propagation, and bundle-size/cost/PII trade-offs. |

## Contributing

Contributions are welcome — including pull requests authored and implemented by AI coding agents. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide. The essentials:

- Keep skills DRY. Prefer referencing official docs, examples, and source code that are already maintained instead of copying large amounts of additional knowledge into the skill. There will be exceptions, but the default should be to link or point to the maintained source of truth.
- Design skills to be token efficient. Avoid dumping large files or broad context into a skill when a targeted lookup, focused reference, or small generated artifact will do.
- Stay vendor neutral.
- Skills must conform to the [Agent Skills specification](https://agentskills.io/specification).
- PRs that add or substantively change a skill must include harness results: the same prompt run on a frontier model without and with the skill, showing the skill helps.
- Contributors sign the [OllyGarden CLA](CLA.md) on their first pull request.

All participants must follow OllyGarden's
[Code of Conduct](https://github.com/ollygarden/.github/blob/main/CODE_OF_CONDUCT.md).
See [SUPPORT.md](SUPPORT.md) for help and issue routing, and
[GOVERNANCE.md](GOVERNANCE.md) for project roles and decision making. Report suspected
vulnerabilities privately under [SECURITY.md](SECURITY.md).
