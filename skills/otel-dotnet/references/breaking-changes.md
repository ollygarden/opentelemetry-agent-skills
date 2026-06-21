# .NET OpenTelemetry Breaking Changes — Audit Workflow

Use this reference to audit existing .NET OpenTelemetry code for deprecated or renamed APIs,
semconv renames, and experimental feature flags. This is a **workflow document** — fetch the
upstream CHANGELOGs rather than relying on any embedded snapshot here.

---

## 1. Understand the Versioning Model First

OpenTelemetry .NET ships two kinds of releases:

**Core packages** (`OpenTelemetry`, `OpenTelemetry.Api`, `OpenTelemetry.Exporter.*`,
`OpenTelemetry.Extensions.Hosting`, etc.) are released together under a single umbrella
`core-x.y.z` tag in the `opentelemetry-dotnet` repository. All packages under that tag share the
same version number.

**Instrumentation and contrib packages** (`OpenTelemetry.Instrumentation.AspNetCore`,
`OpenTelemetry.Instrumentation.Http`, `OpenTelemetry.Instrumentation.GrpcNetClient`, etc.) live in
the `opentelemetry-dotnet-contrib` repository and follow **their own independent versioning
cadence**. A given contrib package may be at a completely different version than the core release
released at the same time (for example, `OpenTelemetry.Instrumentation.AspNetCore` may be at
`1.15.x` while the core is at `1.16.x`).

**Stable vs pre-release packaging:** Core APIs (`OpenTelemetry.Api`) are stable. Instrumentation
packages are often released as pre-release (e.g. `1.12.0-beta.1`) until the underlying semantic
conventions they implement are finalized. Check the NuGet stability label before pinning a version:

```bash
WebFetch https://www.nuget.org/packages/<PackageId>
```

Always fetch the latest `core-*` tag and per-package NuGet version as described in the Sources of
Truth table in `SKILL.md` — do not rely on version numbers embedded in any skill file.

---

## 2. Fetch the CHANGELOGs

Breaking changes, deprecations, and renames are documented per package in each repository.

### 2a. Core packages (`opentelemetry-dotnet`)

```bash
# Discover current core release tag
gh api repos/open-telemetry/opentelemetry-dotnet/releases/latest -q '.tag_name'

# Fetch CHANGELOG for a specific core package — replace <Package> with the directory name
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet/main/src/<Package>/CHANGELOG.md
```

Common core package names (match the directory under `src/`):

- `OpenTelemetry`
- `OpenTelemetry.Api`
- `OpenTelemetry.Exporter.OpenTelemetryProtocol`
- `OpenTelemetry.Exporter.Console`
- `OpenTelemetry.Extensions.Hosting`

### 2b. Contrib / instrumentation packages (`opentelemetry-dotnet-contrib`)

```bash
# Fetch CHANGELOG for a contrib package — replace <Package> with the directory name
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet-contrib/main/src/<Package>/CHANGELOG.md
```

Common contrib package names (match the directory under `src/`):

- `OpenTelemetry.Instrumentation.AspNetCore`
- `OpenTelemetry.Instrumentation.Http`
- `OpenTelemetry.Instrumentation.GrpcNetClient`
- `OpenTelemetry.Instrumentation.SqlClient`
- `OpenTelemetry.Instrumentation.Runtime`
- `OpenTelemetry.Instrumentation.Process`

Each CHANGELOG lists breaking changes, deprecations, and removals in reverse-chronological order.
Look for `### Breaking changes`, `### Deprecated`, and `### Removed` sections.

---

## 3. Audit Protocol

Run this sequence when upgrading any package:

1. **Identify the package type** (core or contrib) and its repository.
2. **Fetch the CHANGELOG** (Step 2 above) for the exact package being upgraded.
3. **Scan from the previous pinned version to the target version** — read every release section in
   that range for breaking changes and deprecations.
4. **Cross-reference semconv renames** — if the CHANGELOG mentions attribute renames, fetch the
   current semantic convention definition via the `otel-semantic-conventions` skill.
5. **Check experimental flags** (Section 4 below).
6. **Repeat** for every upgraded package — core and contrib have independent changelogs.

For a full service upgrade (e.g. upgrading all `OpenTelemetry.*` packages):

```bash
# Step 1: pin down the current core tag
gh api repos/open-telemetry/opentelemetry-dotnet/releases/latest -q '.tag_name'

# Step 2: fetch changelogs for every package in use
# Core examples:
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet/main/src/OpenTelemetry/CHANGELOG.md
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet/main/src/OpenTelemetry.Exporter.OpenTelemetryProtocol/CHANGELOG.md
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet/main/src/OpenTelemetry.Extensions.Hosting/CHANGELOG.md

# Contrib examples:
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet-contrib/main/src/OpenTelemetry.Instrumentation.AspNetCore/CHANGELOG.md
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet-contrib/main/src/OpenTelemetry.Instrumentation.Http/CHANGELOG.md

# Zero-code / CLR-profiler agent (single-repo product — CHANGELOG is at the repo root, not per-package):
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet-instrumentation/main/CHANGELOG.md
```

---

## 4. Experimental Feature Flags

The .NET SDK exposes opt-in experimental behaviors via environment variables prefixed with
`OTEL_DOTNET_EXPERIMENTAL_`. These are not stable API surface and may change or be removed
without a major version bump.

To discover which flags are in effect in a given release, search the CHANGELOG and the source for
`OTEL_DOTNET_EXPERIMENTAL_` after fetching:

```bash
WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet/main/src/OpenTelemetry/CHANGELOG.md
```

Then search the result for `OTEL_DOTNET_EXPERIMENTAL_` to enumerate flags introduced or changed
in that release. Common patterns include flags that enable pre-stable metric views, proto format
choices, or alternative propagation behaviors.

Similarly, check the contrib CHANGELOG of any instrumentation package you use — instrumentation
packages sometimes expose their own experimental flags gated on `OTEL_DOTNET_EXPERIMENTAL_*`.

> Treat any `OTEL_DOTNET_EXPERIMENTAL_*` flag as unstable: re-audit it on every upgrade.

---

## 5. Semantic Convention Attribute Renames

Instrumentation packages in `opentelemetry-dotnet-contrib` track upstream semconv versions.
When semconv renames an attribute (e.g. HTTP, RPC, or database attributes), contrib packages
follow, sometimes in pre-release versions first.

**Audit workflow for semconv renames:**

1. Fetch the CHANGELOG for the affected instrumentation package (Section 2b).
2. Identify any attribute rename noted in the `### Breaking changes` section.
3. Cross-reference the current definition in the `otel-semantic-conventions` skill to confirm the
   canonical name and any opt-in migration flags the package may provide during a transition
   period.
4. Update your custom attribute keys, dashboards, and alert queries to match the new names.

Packages will sometimes emit **both** old and new attribute names for one or more releases behind
a feature flag (e.g. `OTEL_DOTNET_EXPERIMENTAL_*`) to allow migration without a hard cutover —
check the CHANGELOG section for that detail.

---

## 6. Finding Deprecated or Removed APIs

Search the fetched CHANGELOG text for:

- `[Obsolete]` — C# obsolete annotations that become compile-time warnings
- `Deprecated` — noted in prose, may not yet be `[Obsolete]`
- `Removed` — already gone; a compile error if still referenced
- `renamed` / `moved to` — indicates the replacement symbol

For each deprecated or removed API, the CHANGELOG typically names the replacement. If not,
fetch the current API docs:

```bash
# Per-package CHANGELOG is the authoritative source — fetch it as shown in Section 2.
# If you need a general API reference, search the .NET API browser:
WebFetch https://learn.microsoft.com/en-us/dotnet/api/?term=OpenTelemetry
```

or the in-repo migration guide linked from the CHANGELOG entry.

---

## Cross-References

- Semantic convention definitions: `otel-semantic-conventions` skill.
- SDK version selection across languages: `otel-sdk-versions` skill.
- SDK setup and exporter configuration: `references/setup.md`.
- Instrumentation package catalog: `references/instrumentation-libraries.md`.
