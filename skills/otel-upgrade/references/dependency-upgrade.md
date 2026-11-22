# Dependency Upgrade

Use this workflow for OpenTelemetry APIs, SDKs, instrumentation libraries, exporters, plugins, and related packages in any ecosystem.

## Assess the dependency change

1. Inspect relevant manifests, lockfiles, imports, toolchain files, CI, configuration, generated code, deployment files, and repository status.
2. Identify the direct dependencies involved, relevant transitive constraints, actual usage, wrappers, plugins, independently versioned packages, and packages that must move together. Do not assume related packages share a version.
3. Query the authoritative package registry and primary upstream release sources for each relevant package. When `otel-sdk-versions` is available, use it as an initial lookup index, then confirm package-specific versions at their release sources.
4. Resolve the candidate dependency family in an isolated environment. Inspect material changes to the resolved graph and generated lockfiles rather than treating successful resolution as sufficient proof.
5. Run the available project-supported checks relevant to the change, such as dependency resolution, lockfile generation, compilation or type checking, tests, configuration validation, and integration checks.
6. Review release notes and migration guidance for changes connected to APIs, features, configuration, protocols, deployment modes, or externally consumed behavior used by the repository. Inspect the corresponding upstream source diff when risk or incomplete documentation warrants it.
7. Check available downstream contracts, including APIs, schemas, persisted data, events, telemetry, dashboards, and alerts, and identify any required rollout ordering.
8. When SDK instrumentation is involved and `otel-telemetry-emissions` is available, compare exact current and candidate telemetry. Report missing component or version coverage as unverified; do not infer emissions from semantic conventions.

## Resolve versions and package families

Use the ecosystem's registry as the source for a published package version and the tagged upstream source for its contents and release notes. Prefer machine-readable registry commands such as `go list -m -versions <module>`, `mvn dependency:get -Dartifact=<group>:<artifact>:<version>`, `npm view <package> versions --json`, `python -m pip index versions --pre <distribution>`, `dotnet package search <package> --exact-match --prerelease`, and `gem list --remote --all --exact --prerelease <gem>`. Keep the prerelease flags on those three: without them, a package line that publishes only prerelease identifiers (for example Python contrib `b` releases) looks unpublished. Apply the repository's normal resolver afterward; a registry listing does not establish compatibility.

Determine coordination from the release's actual package set and dependency constraints:

- A repository or language may publish stable, experimental, instrumentation, exporter, semantic-convention, or other package lines at different versions and stability levels. Do not derive one line's target from another line's number.
- For monorepos with independently versioned modules or gems, query every directly used package and inspect the candidate tag or release commit. A repository-wide latest tag is not necessarily the package's latest release.
- Keep BOMs, dependency-management packages, metapackages, and lockfiles consistent with their documented role. Do not force equal versions when the upstream release does not.
- Treat declarative configuration schemas, generated semantic-convention artifacts, and code generators as separate dependencies. Check schema/tool compatibility and regenerate checked-in output with the repository's pinned tool rather than editing it.

## Interpret the evidence

- Dependency resolution shows that the resolver found a graph; it does not prove source or behavioral compatibility.
- Successful compilation covers only the configurations and code paths compiled.
- Passing tests increase confidence only for the behavior and environments they exercise.
- An untested version is a candidate, not the highest validated compatible version.
