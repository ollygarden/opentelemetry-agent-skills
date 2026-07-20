# Python OpenTelemetry Breaking Changes — Audit Workflow

Use this reference when upgrading existing Python OpenTelemetry code. It teaches how to find
deprecated or renamed APIs from upstream CHANGELOGs rather than embedding a static list that
rots with each release.

## Step 1: Fetch the CHANGELOGs

Fetch both released upstream sources before reviewing any code. As of
2026-07-20, the released ceiling is core **1.44.0** and contrib **0.65b0**:

```
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/v1.44.0/CHANGELOG.md
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python-contrib/v0.65b0/CHANGELOG.md
```

Scan for entries marked **Deprecated**, **Removed**, or **Breaking** in the version range you are
crossing. Contrib and SDK carry independent version numbers but are released in lock-step (see
Step 2); check both even for SDK-only upgrades, because contrib instrumentation libraries depend
on the SDK ABI.

## Step 2: Understand the Dual Versioning Scheme

Python OpenTelemetry uses two independent version tracks:

| Track | Packages | Example |
|---|---|---|
| Stable (SemVer 1.x) | `opentelemetry-api`, `opentelemetry-sdk` | `1.44.0` |
| Beta (`0.Yb`-suffix) | `opentelemetry-instrumentation-*`, contrib packages, and experimental packages such as `opentelemetry-configuration` (OTLP exporters track stable 1.x) | `0.65b0` |

The two tracks move in lock-step: SDK `1.44.x` pairs with contrib `0.65bx`. When the CHANGELOG
entry says "version 0.65b0", look up the paired SDK version before concluding which SDK release
introduced the change.

The definitive indicator of which track a given package follows is its version string, not its
name: a `b` suffix (e.g. `0.63b0`) means beta track; a plain SemVer (e.g. `1.42.0`) means stable.
Check with `pip show <pkg>` or the PyPI `.info.version` field — some SDK sub-packages (such as
`opentelemetry-sdk-extension-*`) follow the beta track despite the `opentelemetry-sdk` prefix.

Fetch current versions from PyPI to orient yourself:

```
WebFetch https://pypi.org/pypi/opentelemetry-sdk/json          # .info.version → stable track
WebFetch https://pypi.org/pypi/opentelemetry-distro/json       # .info.version → beta track
```

## Step 3: Find Deprecated and Renamed APIs

In the CHANGELOG output, search for:

- `Deprecated` — the old symbol still exists but will be removed in a future release.
- `Removed` — the symbol is gone; code referencing it raises `ImportError` at startup.
- `Renamed` — import path or class name changed.

**Example deprecation an audit would catch:** The SDK's `opentelemetry.sdk._logs.LoggingHandler`
is deprecated; the class emits a `DeprecationWarning` at construction reading "`LoggingHandler` in
`opentelemetry-sdk` is deprecated. Use the handler from `opentelemetry-instrumentation-logging`
instead." The replacement is the `LoggingHandler` class from the contrib
`opentelemetry-instrumentation-logging` package, imported as a drop-in replacement:
```python
from opentelemetry.instrumentation.logging.handler import LoggingHandler
```
Code using the old SDK handler keeps working until removal, but the deprecation entry marks the
migration window. Add the contrib package to `requirements.txt`. See `references/api.md` for wiring
details. (Note: `LoggingInstrumentor` can install the contrib handler during instrumentation and can
also inject trace context into text logs; check the current `opentelemetry-instrumentation-logging`
behavior before treating it as only a formatting helper.)

For each finding, grep the codebase for the old symbol before concluding impact:

```bash
grep -r "LoggingHandler" .  # adjust pattern per finding
```

### Released 1.44.0 / 0.65b0 audit points

When crossing this release boundary, explicitly check for:

- Imports from `opentelemetry.sdk._configuration.file`; declarative setup moved
  to the public, experimental `opentelemetry.configuration` namespace. The old
  SDK extra remains only as an install alias.
- The removed Events API/SDK; emit a `LogRecord` with `event_name` instead.
- Custom environment-carrier code: `EnvironmentGetter.get` now requires the
  source mapping as its first argument rather than using a cached environment
  snapshot.
- Custom log exporters that read `ReadableLogRecord.context`; records buffered
  by `BatchLogRecordProcessor` now carry an empty `Context()` to reduce retained
  memory.
- `ProcessResourceDetector` consumers expecting `process.command_args` or
  `process.command_line`; these privacy-sensitive attributes are now omitted by
  default and require `include_command_args=True`.
- In-place mutation of `opentelemetry.context.Context`; inherited `dict`
  mutation methods no longer modify it. Use the context API to create updated
  contexts.
- Histogram configurations with non-finite or non-increasing explicit bucket
  boundaries; creating the metric stream's aggregation now raises `ValueError`.
  NaN and Inf measurements are dropped at the instrument boundary.
- `opentelemetry-instrumentation-elasticsearch`; contrib removed the package in
  0.65b0 because supported Elasticsearch clients provide native OTel
  instrumentation.

## Step 4: Audit Semantic Convention Renames

Instrumentation libraries adopt updated semconv attribute names on their own schedule; the SDK
CHANGELOG and contrib CHANGELOG both carry these entries. Common patterns include:

- HTTP: `http.method` → `http.request.method`, `http.url` → `url.full`
- DB: `db.sql.table` → `db.collection.name`
- RPC: `rpc.system` values and metric names shift with each stable semconv release

For the canonical attribute-level mapping, cross-reference the **`otel-semantic-conventions`**
skill, which fetches the upstream semconv CHANGELOG and spec. Do not rely on memory for attribute
names — fetch the spec.

Core 1.44.0 updates `opentelemetry-semantic-conventions` to the 1.43.0
semantic-conventions release. Re-run this audit even when your application API
imports did not change.

If your code or dashboards query raw attribute names (e.g. Prometheus label names, Datadog facets,
log filters), a semconv rename is a silent breaking change: telemetry keeps flowing but queries
return no data. Treat semconv entries in the CHANGELOG with the same urgency as API removals.

## Step 5: Check Declarative Configuration

In 1.44.0 / 0.65b0, declarative configuration moved from the private
`opentelemetry.sdk._configuration.file` module to the separate
`opentelemetry-configuration` package and public `opentelemetry.configuration`
namespace. The new package remains experimental, so audit it on each upgrade.

Before upgrading, fetch the README/schema for the release you are upgrading to. As of the
latest released SDK (1.44.0), the declarative configuration README and schema are:

```text
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/v1.44.0/opentelemetry-configuration/README.rst
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/v1.44.0/opentelemetry-configuration/src/opentelemetry/configuration/schema.json
```

Install `opentelemetry-configuration` directly. The old
`opentelemetry-sdk[file-configuration]` extra is a deprecated alias that still
installs it for compatibility.

Cross-reference the **`references/declarative-setup.md`** reference in this skill for the current
YAML schema and activation API. If a release changes the package, that reference is the
authoritative source of the updated usage pattern.

## Step 6: Verify Installed Versions Match

After identifying changes, confirm the running environment matches the target version:

```bash
pip show opentelemetry-sdk opentelemetry-api opentelemetry-configuration \
  opentelemetry-distro opentelemetry-instrumentation
```

A mismatch between `opentelemetry-api` and `opentelemetry-sdk` versions is a common source of
subtle breakage; they must be pinned to the same stable-track version.

## Cross-References

- Attribute renames and semconv mapping: **`otel-semantic-conventions`** skill.
- Declarative config schema and activation: **`references/declarative-setup.md`** in this skill.
- Version selection across languages: **`otel-sdk-versions`** skill.
