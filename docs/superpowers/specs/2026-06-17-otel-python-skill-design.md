# Design: `otel-python` skill + Python guidance adjustments

**Date:** 2026-06-17
**Status:** Approved (design); pending implementation plan

## Goal

Add a per-language `otel-python` skill, on par with the existing `otel-go` skill,
that lets an agent:

1. Set up the OpenTelemetry Python SDK via **declarative YAML configuration** (the
   headline path).
2. Properly instrument a **green-field** Python application for **logs, metrics, and
   traces** — both zero-code/auto and manual.

Validate the skill end-to-end by instrumenting a throwaway example application, then
adjust the language-agnostic skills that need Python coverage.

## Context (verified findings)

- **Existing structure.** `otel-go` is the rich reference (5 files: `declarative-setup`,
  `api`, `instrumentation-libraries`, `performance`, `breaking-changes`). `otel-java` and
  `otel-js` are lean (`declarative-setup` only). All follow a "point at upstream sources of
  truth, don't embed docs" philosophy with small fetch tables. The repo is
  **non-opinionated by design** and optimized for token-efficient retrieval.
- **Python declarative config is real but experimental and private.** It lives at
  `opentelemetry.sdk._configuration.file` (in `opentelemetry-python`, release `1.42.1` /
  `0.63b1` at time of writing). It exposes `load_config_file()` plus
  `configure_tracer_provider()` / `configure_meter_provider()` /
  `configure_logger_provider()` / `configure_propagator()` and a vendored `schema.json`.
  `file_format` is `'1.0'`.
- **No env-var CLI activation yet.** Unlike Go/JS (`OTEL_CONFIG_FILE`), no
  `*_CONFIG_FILE` env var is wired into the `opentelemetry-instrument` CLI (grep over
  `opentelemetry-python-contrib/opentelemetry-instrumentation` found none). Activation is
  **programmatic**: a bootstrap module that calls `load_config_file()` + the `configure_*`
  functions, imported before application code.
- **Python's distinctive zero-code path.** `opentelemetry-distro` +
  `opentelemetry-bootstrap -a install` + `opentelemetry-instrument` CLI, backed by ~50
  contrib auto-instrumentation packages (FastAPI, Flask, Django, requests, httpx, psycopg,
  SQLAlchemy, redis, grpc, logging, …). Closer to the Java agent than to Go.
- **Version index already covers Python** (`opentelemetry-sdk 1.42.1`) but only the primary
  SDK package row.
- **Plugin registration.** Each skill is registered in `.claude-plugin/marketplace.json`.
  `otel-python` must be added there.
- **`local/` is untracked scratch** (no `.gitignore` in the repo; `git ls-files local/`
  returns nothing). Safe place for the throwaway validation app; add a defensive
  `local/otel-python-validation/.gitignore` (or top-level ignore entry) so it never gets
  committed.

## Design

### 1. New skill `skills/otel-python/` — mirrors `otel-go` (5 references)

**`SKILL.md`** — entry point. Sections:
- Short intro framing it as the entry point for OTel mechanics in Python services.
- **References** table (one row per reference, "Use when …").
- **Sources of Truth** fetch table, Python-specific:
  | Fact | Fetch |
  |---|---|
  | Latest `opentelemetry-sdk` / `opentelemetry-api` | `pip index versions opentelemetry-sdk` (or PyPI JSON) |
  | Latest `opentelemetry-distro` | PyPI |
  | Latest `opentelemetry-instrumentation-<pkg>` (contrib) | PyPI per package |
  | Latest OTLP exporter (`opentelemetry-exporter-otlp`) | PyPI |
  | Declarative config support + vendored schema version | `opentelemetry-python` `opentelemetry-sdk/src/opentelemetry/sdk/_configuration` (README + `schema.json`) |
  | SDK CHANGELOG | `opentelemetry-python/CHANGELOG.md` |
  | Contrib CHANGELOG | `opentelemetry-python-contrib/CHANGELOG.md` |
  | Python getting-started docs | opentelemetry.io Python docs |
- **Cross-References** to `otel-declarative-config`, `otel-sdk-versions`,
  `otel-semantic-conventions`.

**References (each self-contained):**

| File | Contents |
|---|---|
| `declarative-setup.md` | **Headline.** `opentelemetry.sdk._configuration.file`: `load_config_file()` + `configure_*()`, `file_format: '1.0'`, env substitution. Programmatic activation pattern (bootstrap imported before app code) since there is no `OTEL_CONFIG_FILE` CLI wiring. Explicit experimental/private-API caveat. Defers YAML schema details to `otel-declarative-config`. Includes a minimal `otel.yaml` skeleton with the parser-accepted `file_format` literal verified against the installed package. |
| `api.md` | Packages (`opentelemetry-api`, `opentelemetry-sdk`); trace/metrics/logs API surface; getting a tracer/meter/logger; attributes; context & propagation; the **Python logging bridge** (`LoggingHandler`) as the idiomatic logs path; resource construction. |
| `instrumentation-libraries.md` | **Zero-code path**: `opentelemetry-distro`, `opentelemetry-bootstrap -a install`, `opentelemetry-instrument` CLI. Detecting existing instrumentation. Contrib catalog (web frameworks, HTTP clients, DBs, messaging, logging). Manual instrumentation patterns following semconv. Enriching auto-instrumented spans. |
| `performance.md` | Sampling; `BatchSpanProcessor` tuning; `PeriodicExportingMetricReader`; views for cardinality control; **asyncio context propagation**; exporter choice (gRPC vs HTTP, compression, retry, timeouts); log-handler cost; graceful shutdown / force-flush. |
| `breaking-changes.md` | Deprecated/renamed APIs across recent SDK & contrib releases; semconv renames; how to audit existing Python code. Driven by CHANGELOG fetches rather than embedded lists. |

Content rule: keep the "fetch, don't embed" discipline. Version numbers and exact literals
come from Sources of Truth at use time; the references teach mechanics and patterns.

### 2. Validation (throwaway under `local/`, not committed)

Minimal **FastAPI** app exercising the full path:

1. Create a venv under `local/otel-python-validation/`.
2. Install `opentelemetry-sdk`, `opentelemetry-distro`,
   `opentelemetry-instrumentation-fastapi`, and an OTLP/console exporter (versions resolved
   via `otel-sdk-versions`).
3. Write an `otel.yaml` declarative config (resource + tracer/meter/logger providers +
   console or OTLP exporter).
4. Programmatic bootstrap module that calls `load_config_file()` + `configure_*()`,
   imported before the app.
5. On top of auto-instrumentation, add **one manual span + one counter + one log record**
   (via the logging bridge) in a route handler.
6. Run the app, hit the route, and confirm **all three signals** are emitted
   (console exporter is an acceptable target; a real collector is optional).

Acceptance: traces (auto span + manual child span), at least one metric data point, and a
log record with trace context all appear. Capture the commands and observed output in the
implementation notes; discard the app afterward.

### 3. Existing skill adjustments (verify-then-edit; touch only what needs it)

- **`otel-declarative-config`** — add `otel-python` to the Cross-References and to the
  per-language pointer line (currently `otel-go`, `otel-java`, `otel-js`). Add a short
  Python workflow note: declarative config is experimental and lives in a private module;
  activation is programmatic, not via `OTEL_CONFIG_FILE`.
- **`otel-sdk-versions`** — Python primary row exists. Add/confirm companion-package
  guidance for Python (distro, instrumentation packages, OTLP exporter, contrib version
  line) consistent with how the index treats other ecosystems.
- **`README.md`** — add `otel-python` to the repository structure listing.
- **`.claude-plugin/marketplace.json`** — register the `otel-python` plugin entry
  (name, source `./skills/otel-python`, description matching the others' style).
- **`otel-semantic-conventions`** — review only; edit solely if the Python work surfaces a
  concrete gap. Expected outcome: no change (language-agnostic skill).

## Out of scope (YAGNI)

- Committing the example app or adding an `examples/` directory to the skill.
- Covering every contrib instrumentation package exhaustively — the reference teaches the
  pattern and points at the contrib catalog/CHANGELOG.
- Django/Flask-specific deep dives beyond what the instrumentation-libraries catalog needs.
- Any opinionated convention (the repo is non-opinionated by design).

## Risks / open points

- The declarative-config module is private (`_configuration`) and experimental; its API
  surface or activation may change. The skill mitigates by flagging this and routing
  literals/versions through Sources of Truth.
- If `local/` turns out to be picked up by `git status` during validation, add an explicit
  ignore so the throwaway app is never committed.
