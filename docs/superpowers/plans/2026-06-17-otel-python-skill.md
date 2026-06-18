# otel-python Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a per-language `otel-python` skill (5 references, on par with `otel-go`) that enables declarative-config SDK setup and green-field instrumentation of logs/metrics/traces, plus the Python adjustments to the language-agnostic skills.

**Architecture:** Author a self-contained skill folder following the existing repo pattern (`SKILL.md` + `references/*.md`, "point at upstream Sources of Truth, don't embed"). A validation spike (throwaway FastAPI app under `local/`) is built **first** to ground the reference content in a working setup, since Python's declarative config is experimental and private. Then write the five references from verified facts, then adjust the agnostic skills, then re-run the spike as the acceptance test.

**Tech Stack:** Markdown skills; Python 3 / FastAPI / OpenTelemetry Python SDK + distro + contrib + OTLP/console exporter for the validation spike.

## Global Constraints

- Repo is **non-opinionated by design** — teach mechanics, never prescribe conventions. (copied verbatim from README)
- **Token-efficient retrieval**: small fetch tables, lookup indexes, scripts pointing at upstream sources of truth; do not copy docs into context.
- **Do not embed version numbers or exact `file_format` literals** in prose as ground truth — route them through Sources of Truth at use time.
- Skill folder name MUST match the `name:` field: `otel-python` (per agentskills.io spec / README).
- Branch: `otel-python-skill` (already created). Commit per task.
- Verified upstream facts (opentelemetry-python `1.42.1`/`0.63b1`):
  - Declarative file config is **private + experimental**: `opentelemetry.sdk._configuration.file`.
  - Public-from-that-module functions: `load_config_file(path) -> OpenTelemetryConfiguration`, `create_resource(config.resource) -> Resource`, `configure_propagator(config.propagator)`, `configure_tracer_provider(config.tracer_provider, resource)`, `configure_meter_provider(config.meter_provider, resource)`, `configure_logger_provider(config.logger_provider, resource)`. Each `configure_*` sets the global provider; `None` section ⇒ global left unset.
  - `file_format` current literal: `'1.0'`.
  - **No `OTEL_CONFIG_FILE`-style CLI activation exists** — activation is programmatic (import a bootstrap module before app code).
  - Zero-code path: `opentelemetry-distro`, `opentelemetry-bootstrap -a install`, `opentelemetry-instrument` CLI; ~50 contrib `opentelemetry-instrumentation-*` packages.

---

### Task 1: Skill scaffold + registration

**Files:**
- Create: `skills/otel-python/SKILL.md`
- Modify: `.claude-plugin/marketplace.json` (add `otel-python` entry to `plugins` array)
- Modify: `README.md` (add `otel-python` to the structure listing)

**Interfaces:**
- Produces: the `skills/otel-python/` folder and its `SKILL.md` References table (referenced by Tasks 3–7), and a registered plugin entry.

- [ ] **Step 1: Write `skills/otel-python/SKILL.md`**

Model it exactly on `skills/otel-go/SKILL.md`. Frontmatter `name: otel-python` and a `description:` covering: SDK setup, declarative config, zero-code (`opentelemetry-instrument`/distro), contrib auto-instrumentation, manual API, performance, breaking changes; with triggers ("setup otel in python", "python telemetry", "python tracing", "opentelemetry-instrument", "opentelemetry-distro", "TracerProvider python", "MeterProvider python", "FastAPI/Flask/Django otel", "python logging bridge"). Body sections:

```markdown
# OpenTelemetry in Python

Entry point for OpenTelemetry mechanics in Python services. Load a reference below based on
the task; each reference is self-contained.

## References

| File | Use when |
|---|---|
| [`references/declarative-setup.md`](references/declarative-setup.md) | Configuring the SDK via declarative YAML (`opentelemetry.sdk._configuration.file`): `load_config_file`, `configure_*`, `file_format`, env substitution, programmatic activation. |
| [`references/api.md`](references/api.md) | Import paths, global API access, tracer/meter/logger usage, attributes, propagation, the Python logging bridge (`LoggingHandler`). |
| [`references/instrumentation-libraries.md`](references/instrumentation-libraries.md) | Zero-code (`opentelemetry-distro`, `opentelemetry-bootstrap`, `opentelemetry-instrument`), the contrib catalog, and manual instrumentation following semconv. |
| [`references/performance.md`](references/performance.md) | Tuning sampling, batch span processor, periodic metric reader, views, asyncio context, exporters, graceful shutdown. |
| [`references/breaking-changes.md`](references/breaking-changes.md) | Auditing existing code for deprecated/renamed APIs and semconv changes across recent SDK/contrib releases. |

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config` skill.
For Python-specific facts:

| Fact | Fetch |
|---|---|
| Latest `opentelemetry-sdk` / `opentelemetry-api` | `WebFetch https://pypi.org/pypi/opentelemetry-sdk/json` (`.info.version`) |
| Latest `opentelemetry-distro` | `WebFetch https://pypi.org/pypi/opentelemetry-distro/json` |
| Latest `opentelemetry-instrumentation-<pkg>` (contrib) | `WebFetch https://pypi.org/pypi/opentelemetry-instrumentation-<pkg>/json` |
| Latest OTLP exporter | `WebFetch https://pypi.org/pypi/opentelemetry-exporter-otlp/json` |
| Declarative config support + vendored schema version | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/main/opentelemetry-sdk/src/opentelemetry/sdk/_configuration/README.md` |
| SDK CHANGELOG | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/main/CHANGELOG.md` |
| Contrib CHANGELOG | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python-contrib/main/CHANGELOG.md` |
| Python getting-started docs | `WebFetch https://opentelemetry.io/docs/languages/python/getting-started/` |

## Cross-References

- Schema-level facts: `otel-declarative-config` skill (language-agnostic YAML schema sources).
- SDK version selection across languages: `otel-sdk-versions` skill.
- Semantic conventions lookup: `otel-semantic-conventions` skill.
```

- [ ] **Step 2: Register the plugin in `.claude-plugin/marketplace.json`**

Append to the `plugins` array (mirror the `otel-go` entry style):

```json
    {
      "name": "otel-python",
      "source": "./skills/otel-python",
      "description": "OpenTelemetry in Python: declarative SDK setup (experimental file config), API surface and logging bridge, zero-code instrumentation (opentelemetry-distro / opentelemetry-instrument) and contrib catalog, performance tuning, breaking-change audits."
    }
```

- [ ] **Step 3: Add `otel-python` to the README structure listing**

In `README.md`, find the `skills/` tree block (the one listing `otel-go/`, `otel-java/`, `otel-js/`) and add an `otel-python/` entry in the same format right after `otel-js/`.

- [ ] **Step 4: Verify JSON + skill load**

Run: `python3 -c "import json,sys; json.load(open('.claude-plugin/marketplace.json')); print('marketplace OK')"`
Expected: `marketplace OK`
Run: `test -f skills/otel-python/SKILL.md && head -2 skills/otel-python/SKILL.md`
Expected: prints `---` then `name: otel-python`.

- [ ] **Step 5: Commit**

```bash
git add skills/otel-python/SKILL.md .claude-plugin/marketplace.json README.md
git commit -m "feat(otel-python): scaffold skill and register plugin"
```

---

### Task 2: Validation spike — working FastAPI app (grounds all reference content)

**Files:**
- Create: `local/otel-python-validation/.gitignore`
- Create: `local/otel-python-validation/otel.yaml`
- Create: `local/otel-python-validation/bootstrap.py`
- Create: `local/otel-python-validation/app.py`
- Create: `local/otel-python-validation/run.sh`
- Create: `local/otel-python-validation/RESULTS.md` (captured output — referenced by Tasks 3–5)

**Interfaces:**
- Produces: a verified-working declarative-config bootstrap, `otel.yaml`, and manual-signal code that Tasks 3 (declarative-setup), 4 (api), and 5 (instrumentation-libraries) copy verified snippets from. `RESULTS.md` records exact installed versions and observed all-three-signal output.

- [ ] **Step 1: Add gitignore so the throwaway app is never committed**

`local/otel-python-validation/.gitignore`:
```gitignore
*
```

- [ ] **Step 2: Create the venv and install pinned packages**

Resolve current versions first (use the `otel-sdk-versions` skill / PyPI JSON), then:
```bash
cd local/otel-python-validation
python3 -m venv .venv && . .venv/bin/activate
pip install fastapi uvicorn \
  "opentelemetry-sdk[file-configuration]" opentelemetry-api \
  opentelemetry-distro \
  opentelemetry-instrumentation-fastapi \
  opentelemetry-exporter-otlp
pip freeze | grep -E '^opentelemetry|^fastapi|^uvicorn' > installed-versions.txt
```
Expected: install succeeds; `installed-versions.txt` lists the opentelemetry packages.

- [ ] **Step 3: Write `otel.yaml` (declarative config, console exporters)**

Confirm the accepted `file_format` literal against the installed package
(`python -c "from opentelemetry.sdk._configuration.file import load_config_file"` must import).
Adapt the canonical example from the `otel-declarative-config` skill. Console-exporter skeleton:
```yaml
file_format: "1.0"
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-otel-python-validation}"
    - name: service.version
      value: "0.1.0"
tracer_provider:
  processors:
    - simple:
        exporter:
          console: {}
meter_provider:
  readers:
    - periodic:
        interval: 5000
        exporter:
          console: {}
logger_provider:
  processors:
    - simple:
        exporter:
          console: {}
```
If a field/exporter key is rejected at load time, fetch `examples/otel-sdk-config.yaml` from
the `otel-declarative-config` Sources of Truth and reconcile against the installed parser
(the parser wins). Record the final working YAML.

- [ ] **Step 4: Write `bootstrap.py` (programmatic activation)**

```python
"""Import this module before any application code to configure OTel from otel.yaml."""
import os
from opentelemetry.sdk._configuration.file import (
    load_config_file,
    create_resource,
    configure_propagator,
    configure_tracer_provider,
    configure_meter_provider,
    configure_logger_provider,
)

_config = load_config_file(os.environ.get("OTEL_PY_CONFIG", "otel.yaml"))
_resource = create_resource(_config.resource)
configure_propagator(_config.propagator)
configure_tracer_provider(_config.tracer_provider, _resource)
configure_meter_provider(_config.meter_provider, _resource)
configure_logger_provider(_config.logger_provider, _resource)
```
Run: `python -c "import bootstrap; print('bootstrap OK')"` (from the app dir, with `otel.yaml` present).
Expected: `bootstrap OK` and no exception. If `configure_*` argument names differ in the
installed version, read the installed `opentelemetry/sdk/_configuration/file/__init__.py`
`__all__` and the function signatures and adjust; record the working form in `RESULTS.md`.

- [ ] **Step 5: Write `app.py` (auto-instrumentation + one manual span/metric/log each)**

```python
import bootstrap  # noqa: F401  — must come first

import logging
from fastapi import FastAPI
from opentelemetry import trace, metrics
from opentelemetry._logs import get_logger_provider
from opentelemetry.sdk._logs import LoggingHandler
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

logging.getLogger().addHandler(LoggingHandler(logger_provider=get_logger_provider()))
logging.getLogger().setLevel(logging.INFO)
log = logging.getLogger("validation")

tracer = trace.get_tracer("validation")
meter = metrics.get_meter("validation")
hits = meter.create_counter("validation.hits")

app = FastAPI()
FastAPIInstrumentor.instrument_app(app)

@app.get("/work")
def work():
    with tracer.start_as_current_span("manual-work"):
        hits.add(1, {"route": "/work"})
        log.info("handled /work")
        return {"ok": True}
```
If `LoggingHandler`'s import path or constructor differs in the installed version, read the
installed `opentelemetry/sdk/_logs/__init__.py` and adjust; record the working form.

- [ ] **Step 6: Write `run.sh` and run the app**

`run.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
. .venv/bin/activate
exec uvicorn app:app --port 8000
```
```bash
chmod +x run.sh && ./run.sh &   # then in another shell:
sleep 3 && curl -s localhost:8000/work
```
Expected: `{"ok":true}` from curl.

- [ ] **Step 7: Verify all three signals appear, capture results**

Inspect the uvicorn stdout. Expected console output to contain:
- a span named `manual-work` AND an auto span for `GET /work` (FastAPI instrumentation),
- a metric named `validation.hits`,
- a log record body `handled /work` carrying trace context.

Write `RESULTS.md` capturing: installed versions, the final `otel.yaml`, the exact
bootstrap/app snippets that worked, and a trimmed copy of the console output proving all
three signals. Stop the server.

- [ ] **Step 8: Confirm nothing under the spike is staged for commit**

Run: `git status --porcelain local/otel-python-validation`
Expected: empty output (covered by the `*` gitignore). No commit for this task — it is a spike whose deliverable is `RESULTS.md` consumed by later tasks.

---

### Task 3: `references/declarative-setup.md`

**Files:**
- Create: `skills/otel-python/references/declarative-setup.md`

**Interfaces:**
- Consumes: verified `otel.yaml` + `bootstrap.py` from Task 2 `RESULTS.md`.
- Produces: the headline setup reference.

- [ ] **Step 1: Write the reference**

Model structure on `skills/otel-go/references/declarative-setup.md`. Required sections:
- One-line intro + a prominent **experimental/private-API caveat**: declarative file config lives in `opentelemetry.sdk._configuration.file` (private module, may change between releases).
- **Sources of Truth** mini-table (reuse the Python rows from `SKILL.md`; defer YAML schema to `otel-declarative-config`).
- **Activation** — programmatic only; there is no `OTEL_CONFIG_FILE` CLI wiring. Show the verified `bootstrap.py` pattern from Task 2, and the rule: import the bootstrap module before any application import.
- **YAML config** — minimal skeleton (the verified `otel.yaml` from Task 2), noting `file_format` must be the literal the installed parser accepts; point at `otel-declarative-config` for the full schema.
- **Key facts** — `configure_*(section, resource)` sets globals; absent section ⇒ global left unset; env substitution handled per `otel-declarative-config`.

- [ ] **Step 2: Verify links and cross-references resolve**

Run: `grep -n "otel-declarative-config\|references/" skills/otel-python/references/declarative-setup.md`
Expected: cross-reference to `otel-declarative-config` present; no broken relative paths.

- [ ] **Step 3: Commit**

```bash
git add skills/otel-python/references/declarative-setup.md
git commit -m "docs(otel-python): add declarative-setup reference"
```

---

### Task 4: `references/api.md`

**Files:**
- Create: `skills/otel-python/references/api.md`

**Interfaces:**
- Consumes: verified manual-signal snippets from Task 2.
- Produces: the API-surface reference (incl. logging bridge).

- [ ] **Step 1: Write the reference**

Model on `skills/otel-go/references/api.md`. Sections: Import paths (`opentelemetry.api` vs `opentelemetry.sdk`); Global API access (`trace.get_tracer`, `metrics.get_meter`, `_logs.get_logger`); Tracing API (start_as_current_span, span attributes/status, context utilities); Metrics API (counter/histogram/up_down_counter/observable callbacks); Attributes; Propagation (`opentelemetry.propagate` inject/extract); **Logs / Python logging bridge** — wiring `LoggingHandler` to the stdlib `logging` module (the verified snippet from Task 2), and the relationship to auto `opentelemetry-instrumentation-logging`. Keep code minimal and correct; route versions through Sources of Truth.

- [ ] **Step 2: Verify**

Run: `grep -n "LoggingHandler\|get_tracer\|get_meter" skills/otel-python/references/api.md`
Expected: all three present.

- [ ] **Step 3: Commit**

```bash
git add skills/otel-python/references/api.md
git commit -m "docs(otel-python): add api reference"
```

---

### Task 5: `references/instrumentation-libraries.md`

**Files:**
- Create: `skills/otel-python/references/instrumentation-libraries.md`

**Interfaces:**
- Consumes: Task 2 install commands + auto-instrumentation behavior.
- Produces: zero-code + catalog + manual-pattern reference.

- [ ] **Step 1: Write the reference**

Model on `skills/otel-go/references/instrumentation-libraries.md`. Sections:
- **Detecting existing instrumentation** (search for `opentelemetry-instrumentation-*`, `opentelemetry-distro`, `opentelemetry-instrument` in run command / requirements).
- **Zero-code path**: `pip install opentelemetry-distro`, `opentelemetry-bootstrap -a install` (auto-detects installed libs and installs matching instrumentations), `opentelemetry-instrument <cmd>` wrapping; note it configures via env vars / distro defaults (distinct from the declarative path in Task 3).
- **Per-app instrumentation** (the `FastAPIInstrumentor.instrument_app(app)` form from Task 2) as the alternative to the CLI.
- **Contrib catalog** — table of common packages (fastapi, flask, django, requests, httpx, urllib3, psycopg/psycopg2, sqlalchemy, redis, pymongo, grpc, celery, kafka-python, logging, system-metrics) with the "fetch exact version from PyPI" rule; do not pin versions in prose. Point at the contrib repo `instrumentation/` dir + CHANGELOG as Sources of Truth.
- **Manual instrumentation patterns** following semconv (HTTP client, DB call, background job) — short, cross-reference `otel-semantic-conventions`.
- **Enriching auto-instrumented spans** (`trace.get_current_span().set_attribute(...)`).

- [ ] **Step 2: Verify**

Run: `grep -n "opentelemetry-bootstrap\|opentelemetry-instrument\|otel-semantic-conventions" skills/otel-python/references/instrumentation-libraries.md`
Expected: all present.

- [ ] **Step 3: Commit**

```bash
git add skills/otel-python/references/instrumentation-libraries.md
git commit -m "docs(otel-python): add instrumentation-libraries reference"
```

---

### Task 6: `references/performance.md`

**Files:**
- Create: `skills/otel-python/references/performance.md`

- [ ] **Step 1: Write the reference**

Model on `skills/otel-go/references/performance.md`, adapted to Python. Sections: Performance impact by signal; default config values (route exact numbers through CHANGELOG/source, don't hard-assert); Sampling (`TraceIdRatioBased`, `ParentBased`, `OTEL_TRACES_SAMPLER`); `BatchSpanProcessor` tuning vs `SimpleSpanProcessor`; `PeriodicExportingMetricReader`; **Views for cardinality control**; **asyncio context propagation** (contextvars-based context; pitfalls crossing threads/executors); Exporter choice (gRPC vs HTTP, compression, retry, timeout); log-handler cost; graceful shutdown / `force_flush` / `shutdown()` on providers.

- [ ] **Step 2: Verify**

Run: `grep -n "BatchSpanProcessor\|PeriodicExportingMetricReader\|asyncio\|force_flush" skills/otel-python/references/performance.md`
Expected: all present.

- [ ] **Step 3: Commit**

```bash
git add skills/otel-python/references/performance.md
git commit -m "docs(otel-python): add performance reference"
```

---

### Task 7: `references/breaking-changes.md`

**Files:**
- Create: `skills/otel-python/references/breaking-changes.md`

- [ ] **Step 1: Write the reference**

Model on `skills/otel-go/references/breaking-changes.md`. Frame as an **audit workflow** driven by CHANGELOG fetches (SDK + contrib Sources of Truth) rather than an embedded static list: how to find deprecated/renamed APIs, the stable-vs-`b`-suffixed (beta) versioning of `opentelemetry-api`/`-sdk` vs contrib (`0.x b`), semconv attribute renames (cross-ref `otel-semantic-conventions`), and the experimental status of the `_configuration` module. Include the two CHANGELOG fetch commands.

- [ ] **Step 2: Verify**

Run: `grep -n "CHANGELOG\|opentelemetry-python-contrib\|_configuration" skills/otel-python/references/breaking-changes.md`
Expected: all present.

- [ ] **Step 3: Commit**

```bash
git add skills/otel-python/references/breaking-changes.md
git commit -m "docs(otel-python): add breaking-changes reference"
```

---

### Task 8: Adjust language-agnostic skills

**Files:**
- Modify: `skills/otel-declarative-config/SKILL.md` (cross-refs line ~94, per-language pointer line ~54, add Python workflow note)
- Modify: `skills/otel-sdk-versions/references/generated/otel-version-index.md` (Python companion packages) — **only if** the index treats other languages with companion rows; otherwise leave the generated file and instead confirm SKILL.md guidance suffices
- Review: `skills/otel-semantic-conventions/SKILL.md` (edit only if a concrete Python gap is found)

**Interfaces:**
- Consumes: nothing from prior tasks beyond the skill name `otel-python`.

- [ ] **Step 1: Update `otel-declarative-config`**

In `skills/otel-declarative-config/SKILL.md`:
- Cross-References section: change `otel-go`, `otel-java`, `otel-js` to include `otel-python`.
- Per-language pointer sentence (currently "see the Sources of Truth section in each language's `otel-<lang>` skill (`otel-go`, `otel-java`, `otel-js`)"): add `otel-python`.
- Add a one-paragraph **Python note**: declarative config is supported via the experimental, private `opentelemetry.sdk._configuration.file` module; activation is programmatic (no `OTEL_CONFIG_FILE` CLI wiring); see the `otel-python` skill.

- [ ] **Step 2: Check whether the version index needs Python companion rows**

Run: `grep -n "Python\|distro\|instrumentation" skills/otel-sdk-versions/references/generated/otel-version-index.md`
Expected: shows the existing single Python `opentelemetry-sdk` row.
Decision: if the index has no companion-package rows for any language (it is one-primary-package-per-language by design — confirmed in `otel-sdk-versions/SKILL.md` step 3), make **no change** to the generated file; the SKILL's "resolve companion packages from the release source" guidance already covers distro/instrumentation/exporter. Note this decision in the commit message.

- [ ] **Step 3: Review semantic-conventions skill**

Read `skills/otel-semantic-conventions/SKILL.md`. It is language-agnostic; expected outcome is **no change**. Edit only if Task 4/5 surfaced a Python-specific naming gap (none expected).

- [ ] **Step 4: Verify**

Run: `grep -n "otel-python" skills/otel-declarative-config/SKILL.md`
Expected: appears in both the cross-references and the per-language pointer.

- [ ] **Step 5: Commit**

```bash
git add skills/otel-declarative-config/SKILL.md
git commit -m "docs(otel-declarative-config): add otel-python cross-references and Python note"
```

---

### Task 9: End-to-end acceptance — re-run the spike against the written skill

**Files:** none created (verification only).

- [ ] **Step 1: Dry-run the skill as an agent would**

Following only `skills/otel-python/SKILL.md` → `declarative-setup.md` + `api.md` +
`instrumentation-libraries.md`, reconstruct the FastAPI setup in a clean copy under
`local/otel-python-validation-2/` (gitignored). This checks the references alone are
sufficient to set up declarative config + manual all-three-signals.

- [ ] **Step 2: Run and confirm all three signals**

Run the app, `curl localhost:8001/work`, inspect console output for a manual span, an auto
FastAPI span, the `validation.hits` metric, and the log record with trace context.
Expected: all present (same as Task 2 Step 7).

- [ ] **Step 3: Confirm no scratch dirs are committed**

Run: `git status --porcelain local/`
Expected: empty.

- [ ] **Step 4: Final review + open PR (if requested)**

Confirm all reference files exist and link correctly:
```bash
ls skills/otel-python/references/
grep -L "otel-declarative-config\|Sources of Truth" skills/otel-python/references/*.md || true
```
Expected: five reference files present. Then, if the user wants a PR, push the
`otel-python-skill` branch and open one.

---

## Self-Review

**Spec coverage:**
- 5-reference skill mirroring Go → Tasks 1,3,4,5,6,7. ✓
- Declarative config headlined with experimental caveat → Task 3 (+ verified in Task 2). ✓
- Green-field logs/metrics/traces, zero-code + manual → Tasks 4,5 + validation Tasks 2,9. ✓
- FastAPI validation, throwaway under `local/`, console exporter acceptable → Tasks 2,9. ✓
- Adjust otel-declarative-config, otel-sdk-versions (verify-then-edit), otel-semantic-conventions (review-only), README + marketplace → Tasks 1,8. ✓
- Python logging bridge as idiomatic logs path → Tasks 2,4. ✓
- asyncio note → Task 6. ✓

**Placeholder scan:** Validation app code, otel.yaml, bootstrap, marketplace entry, and verify commands are all concrete. Reference-authoring steps specify exact sections + a model file + a grep verification (content authored from the Task 2 verified spike and upstream fetches, not from embedded guesses) — intentional given the "fetch, don't embed" repo philosophy.

**Type consistency:** `load_config_file`, `create_resource`, `configure_propagator/tracer_provider/meter_provider/logger_provider`, `LoggingHandler`, `FastAPIInstrumentor.instrument_app` used consistently across Tasks 2,3,4,5 and match the verified upstream signatures in Global Constraints.
