---
name: otel-sdk-versions
description: OpenTelemetry release version lookup for SDKs across languages and for the Collector (core, contrib, distributions, builder/OCB), the declarative configuration schema, semantic conventions, and Weaver. Use when choosing the latest compatible released OpenTelemetry SDK, package, or component version, when asked what the latest release of one of these is, and when locating setup docs or examples. Not for Collector build manifests or upgrade-risk assessment.
---

# OpenTelemetry SDK Versions

## Workflow

1. Open `references/generated/otel-version-index.md`.
- the SDKs table has one primary SDK package per language; the Components table covers the Collector (core, contrib, distributions, builder), the configuration schema, semantic conventions, and Weaver
- use the bundled tables as the default source of truth for what they cover
- its "Versions last changed" date is the last day any row changed
- offline, give each version with that date: "`<version>` as of `<date>`; a newer release may exist". Never call it the current latest

2. Choose the version deliberately.
- prefer the latest released version when it is compatible with the project
- if the latest release is not compatible, choose the latest compatible version and state the compatibility reason explicitly
- reuse that decision for the rest of the task unless the language, package, or constraints change

3. Resolve companion packages from the release source.
- if the task requires additional packages from the same ecosystem (exporters, instrumentations, resource detectors, Collector components), look up their versions from the release source — do not infer or reuse the primary package version
- sub-packages within the same project often share a version line but not always; confirm each package individually
- Collector components in one build must match the core release line its builder uses, which can trail core; see `otel-collector-builder` for manifest pinning

4. Use the linked follow-up sources when needed.
- use the Release Source or Release column to confirm the package, repo, or release notes
- use the Setup Docs column for SDK setup guidance
- use the Examples column for implementation references when examples are available

5. Handle gaps explicitly.
- if the requested language, package, or component is not in the bundled index, say that the index does not cover it
- then fall back to the official release source and official docs for that package
- do not assume an unreleased, prerelease, or incompatible version is acceptable without saying so
