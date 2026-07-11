# Python OpenTelemetry Breaking Changes — Audit Workflow

Use this reference when upgrading existing Python OpenTelemetry code. It teaches how to find
deprecated or renamed APIs from upstream CHANGELOGs rather than embedding a static list that
rots with each release.

## Step 1: Fetch the CHANGELOGs

Fetch both upstream sources before reviewing any code:

```
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/main/CHANGELOG.md
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python-contrib/main/CHANGELOG.md
```

Scan for entries marked **Deprecated**, **Removed**, or **Breaking** in the version range you are
crossing. Contrib and SDK carry independent version numbers but are released in lock-step (see
Step 2); check both even for SDK-only upgrades, because contrib instrumentation libraries depend
on the SDK ABI.

## Step 2: Understand the Dual Versioning Scheme

Python OpenTelemetry uses two independent version tracks:

| Track | Packages | Example |
|---|---|---|
| Stable (SemVer 1.x) | `opentelemetry-api`, `opentelemetry-sdk` | `1.43.0` |
| Beta (`0.Yb`-suffix) | `opentelemetry-instrumentation-*`, contrib packages, some SDK extras (OTLP exporters track stable 1.x) | `0.64b0` |

The two tracks move in lock-step: SDK `1.43.x` pairs with contrib `0.64bx`. When the CHANGELOG
entry says "version 0.64b0", look up the paired SDK version before concluding which SDK release
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
details. (Note: `LoggingInstrumentor` is a separate feature for injecting trace context into log
text formatting; it is not the deprecation replacement.)

For each finding, grep the codebase for the old symbol before concluding impact:

```bash
grep -r "LoggingHandler" .  # adjust pattern per finding
```

## Step 4: Audit Semantic Convention Renames

Instrumentation libraries adopt updated semconv attribute names on their own schedule; the SDK
CHANGELOG and contrib CHANGELOG both carry these entries. Common patterns include:

- HTTP: `http.method` → `http.request.method`, `http.url` → `url.full`
- DB: `db.sql.table` → `db.collection.name`
- RPC: `rpc.system` values and metric names shift with each stable semconv release

For the canonical attribute-level mapping, cross-reference the **`otel-semantic-conventions`**
skill, which fetches the upstream semconv CHANGELOG and spec. Do not rely on memory for attribute
names — fetch the spec.

If your code or dashboards query raw attribute names (e.g. Prometheus label names, Datadog facets,
log filters), a semconv rename is a silent breaking change: telemetry keeps flowing but queries
return no data. Treat semconv entries in the CHANGELOG with the same urgency as API removals.

## Step 5: Check the `_configuration` Module

The `opentelemetry.sdk._configuration` module (declarative YAML config) is **private and
experimental**. The leading underscore is intentional: the public API is not guaranteed. Any
release may rename the entry-point, change the schema, or remove the module without a deprecation
period.

Before upgrading, fetch the current README for this module:

```
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/main/opentelemetry-sdk/src/opentelemetry/sdk/_configuration/README.md
```

Cross-reference the **`references/declarative-setup.md`** reference in this skill for the current
YAML schema and activation API. If a release changes `_configuration`, that reference is the
authoritative source of the updated usage pattern.

## Step 6: Verify Installed Versions Match

After identifying changes, confirm the running environment matches the target version:

```bash
pip show opentelemetry-sdk opentelemetry-api opentelemetry-distro opentelemetry-instrumentation
```

A mismatch between `opentelemetry-api` and `opentelemetry-sdk` versions is a common source of
subtle breakage; they must be pinned to the same stable-track version.

## Cross-References

- Attribute renames and semconv mapping: **`otel-semantic-conventions`** skill.
- Declarative config schema and activation: **`references/declarative-setup.md`** in this skill.
- Version selection across languages: **`otel-sdk-versions`** skill.
