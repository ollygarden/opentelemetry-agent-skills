# OllyGarden Agent Skills for OpenTelemetry

[![skills.sh](https://www.skills.sh/b/ollygarden/opentelemetry-agent-skills)](https://www.skills.sh/ollygarden/opentelemetry-agent-skills)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

OllyGarden Agent Skills for OpenTelemetry are open source skills that give coding agents such as Claude Code, Cursor, Codex, and GitHub Copilot accurate, current OpenTelemetry knowledge: SDK setup for Go, Java, JavaScript, Python, .NET, Ruby, and the browser; Collector configuration and OTTL; semantic conventions; and upgrades. They follow the [Agent Skills specification](https://agentskills.io/specification), so they work with any agent that supports it. The full catalog, and how they relate to OllyGarden's opinionated [`skills`](https://github.com/ollygarden/skills) package, is at [ollygarden.com/resources/agent-skills](https://ollygarden.com/resources/agent-skills).

The skills are **non-opinionated by design** — there are many valid ways to use OpenTelemetry, and prescribing conventions is out of scope.

They are designed for **token-efficient, agent-friendly retrieval**: small fetch tables, lookup indexes, and scripts that point at upstream sources of truth instead of copying docs into context. The agent finds what it missed during training; answers stay current as the project evolves.

## Available Skills

Language-agnostic skills:

| Skill | Path | Use When |
| --- | --- | --- |
| `otel-collector` | `skills/otel-collector/` | Configuring OpenTelemetry Collector components — config keys, defaults, validation, signal support, stability, and gotchas. Progressive disclosure via `components/<type>/README.md` plus on-demand detail files. |
| `otel-collector-builder` | `skills/otel-collector-builder/` | Building custom OpenTelemetry Collector distributions with OCB — authoring the builder manifest, aligning core/contrib/provider versions, local component development, CI/Docker/multi-arch builds, and build troubleshooting. |
| `otel-declarative-config` | `skills/otel-declarative-config/` | Configuring OpenTelemetry SDK providers via a single YAML file (`otelconf`, `OTEL_CONFIG_FILE`, `file_format`). Points at the upstream schema, env-var substitution rules, and configuration precedence. |
| `otel-ottl` | `skills/otel-ottl/` | Authoring or reviewing OTTL statements for `transform`, `filter`, `routing`, and `tail_sampling` processors; debugging OTTL syntax and semantics; transforming traces, metrics, logs, and profiles in the Collector. |
| `otel-profiles` | `skills/otel-profiles/` | Continuous profiling with the OpenTelemetry profiles signal: deploying the eBPF profiler (`otelcol-ebpf-profiler`) on Linux hosts and Kubernetes, including kind/k3d, exporting OTLP profiles to a backend, verifying data arrives, and troubleshooting by error message. |
| `otel-sdk-versions` | `skills/otel-sdk-versions/` | Choosing the latest compatible released OpenTelemetry SDK or package version for a language and finding setup docs or examples. |
| `otel-semantic-conventions` | `skills/otel-semantic-conventions/` | Selecting released semantic convention groups, attributes, and span naming rules; checking compliance; looking up exact upstream entries via the bundled query script. |
| `otel-span-events-to-logs-migration` | `skills/otel-span-events-to-logs-migration/` | Migrating instrumentation from the deprecated Span Event API (`AddEvent`, `RecordException`) to the Logs API following the OTEP 4430 deprecation plan. |
| `otel-telemetry-emissions` | `skills/otel-telemetry-emissions/` | Looking up which telemetry (spans, metrics, logs, attributes) a collector component or SDK instrumentation package emits at a specific version, and how emission changed across versions — e.g. when upgrading a component or SDK. |
| `otel-telemetrygen` | `skills/otel-telemetrygen/` | Constructing `telemetrygen` commands for generating synthetic traces, metrics, and logs; load-testing collectors; validating OTTL transforms, tail sampling, and filter rules. |
| `otel-upgrade` | `skills/otel-upgrade/` | Assessing OpenTelemetry package and Collector upgrades across ecosystems, including version selection, dependency compatibility, Collector distributions, configuration, builds, runtime behavior, telemetry changes, and rollout risk. |
| `otel-weaver` | `skills/otel-weaver/` | Authoring an OpenTelemetry Weaver registry, writing Jinja2 templates, generating language bindings, and wiring `weaver registry check`/`generate`/`diff` into CI. |

Language-specific skills:

| Skill | Path | Use When |
| --- | --- | --- |
| `otel-go` | `skills/otel-go/` | OpenTelemetry in Go: declarative SDK setup with `otelconf`, API surface, contrib instrumentation libraries (otelhttp, otelgrpc, etc.), compile-time zero-code instrumentation (`otelc`), performance tuning, and breaking-change audits. |
| `otel-java` | `skills/otel-java/` | OpenTelemetry in Java: Javaagent zero-code instrumentation, Spring Boot Starter, manual autoconfigure SDK, declarative YAML configuration, and BOM dependency management. |
| `otel-js` | `skills/otel-js/` | OpenTelemetry in Node.js / JavaScript / TypeScript: NodeSDK setup, declarative YAML configuration via `@opentelemetry/configuration`, auto-instrumentations, and ESM vs CJS import patterns. |
| `otel-python` | `skills/otel-python/` | OpenTelemetry in Python: declarative SDK setup via file config, API surface and logging bridge, zero-code instrumentation (opentelemetry-distro / opentelemetry-instrument) and contrib catalog, performance tuning, breaking-change audits. |
| `otel-dotnet` | `skills/otel-dotnet/` | OpenTelemetry in .NET: DI/builder SDK setup (`OpenTelemetry.Extensions.Hosting`), native BCL instrumentation (`ActivitySource`, `Meter`, `ILogger`), zero-code CLR-profiler agent and contrib instrumentation catalog, performance tuning, breaking-change audits. |
| `otel-ruby` | `skills/otel-ruby/` | OpenTelemetry in Ruby: SDK and Bundler setup, manual tracing, experimental metrics and logs, contrib instrumentation for Rails/Rack/Sinatra/Sidekiq and other gems, propagation, performance, and breaking-change audits. |
| `otel-browser` | `skills/otel-browser/` | OpenTelemetry in the browser (Real User Monitoring / RUM): web tracing SDK (`sdk-trace-web`, `context-zone`) and experimental `browser-sdk`, event- and span-based browser instrumentations (web vitals, navigation, errors, fetch/XHR), sessions, frontend→backend trace propagation, and bundle-size/cost/PII trade-offs. |

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

### Cursor, Codex, GitHub Copilot, and other agents

The `skills` CLI installs into the coding agents it detects. Pass `-a` to choose one, such as `cursor`, `codex`, or `github-copilot`:

```bash
npx skills add ollygarden/opentelemetry-agent-skills -a cursor
```

See the [skills CLI documentation](https://github.com/vercel-labs/skills#supported-agents) for every supported agent.

## Repository Structure

Each skill is a self-contained folder under `skills/`, named to match its `name:` field per
the [agentskills.io specification](https://agentskills.io/specification). Language-specific
skills bundle task-focused references; language-agnostic skills sit alongside them.

```
skills/
  otel-go/
    SKILL.md           # name: otel-go
    references/        # declarative-setup, api, instrumentation-libraries, performance, breaking-changes, compile-time-instrumentation
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
  otel-ruby/
    SKILL.md
    references/
  otel-browser/
    SKILL.md
    references/        # setup-sdk, instrumentation, performance
  otel-collector/
    SKILL.md
    components/        # one directory per Collector component (log_dedup, interval, …)
  otel-collector-builder/
    SKILL.md
    references/        # manifest, workflows, troubleshooting
  otel-declarative-config/
  otel-ottl/
  otel-profiles/
    SKILL.md
    references/        # receiver, kubernetes, nested-containers, backends, troubleshooting
  otel-sdk-versions/
  otel-semantic-conventions/
  otel-span-events-to-logs-migration/
  otel-telemetry-emissions/
    SKILL.md
    <repo>/<path>/     # generated telemetry inventories, one file per version
  otel-telemetrygen/
  otel-upgrade/
    SKILL.md
    references/        # dependency and Collector upgrade workflows
  otel-weaver/

bin/                     # the gates CI runs, runnable locally
  validate-skill.sh      # Agent Skills spec conformance + house rules
  check-skill-inventory.py
  skills-ref.requirement # pinned validator revision; the single source
docs/
  preferred-workflow.md  # how a change moves through this repository
tools/otel-agent-tools/  # Go CLI that generates bundled reference data
tools/otel-telemetry-emission-scan/  # Authoring-time telemetry emission scanner
```

## Contributing

Contributions are welcome — including pull requests authored and implemented by AI coding agents. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide. The essentials:

- Keep skills DRY. Prefer referencing official docs, examples, and source code that are already maintained instead of copying large amounts of additional knowledge into the skill. There will be exceptions, but the default should be to link or point to the maintained source of truth.
- Design skills to be token efficient. Avoid dumping large files or broad context into a skill when a targeted lookup, focused reference, or small generated artifact will do.
- Stay vendor neutral.
- Skills must conform to the [Agent Skills specification](https://agentskills.io/specification). Check yours with `./bin/validate-skill.sh` and `./bin/check-skill-inventory.py` — the same gates CI runs.
- PRs that add or substantively change a skill must include harness results: the same prompt run on a frontier model without and with the skill, showing the skill helps.
- [`docs/preferred-workflow.md`](docs/preferred-workflow.md) walks a change through the repository end to end, from branch to merge.
- Contributors sign the organization-wide
  [OllyGarden CLA](https://github.com/ollygarden/.github/blob/main/CLA.md) on their first pull request.

All participants must follow OllyGarden's
[Code of Conduct](https://github.com/ollygarden/.github/blob/main/CODE_OF_CONDUCT.md).
See [SUPPORT.md](SUPPORT.md) for help and issue routing, and
OllyGarden's organization-wide
[governance policy](https://github.com/ollygarden/.github/blob/main/GOVERNANCE.md) for project roles
and decision making. Report suspected vulnerabilities privately under the inherited
[security policy](https://github.com/ollygarden/opentelemetry-agent-skills/security/policy).
