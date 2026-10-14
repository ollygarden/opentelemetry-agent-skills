---
name: otel-component-telemetry
description: Scan one OpenTelemetry component at pinned upstream versions and record the telemetry it emits (spans, metrics, logs) as change-point files under skills/otel-telemetry-emissions/. Invoked headless by scripts/scan.sh, one agent per component from config.json. Authoring-time only — not shipped.
---

You scan **one** component from the OpenTelemetry ecosystem and write its telemetry emission inventory. 

Repo: ${Repo}
Path: ${Path}
Versions to scan: ${Version}
Tag prefix: ${TagPrefix}
Force: ${Force}
Output root: ${OutDir}

## Procedure

1. **Resolve versions.** The repo is already cloned locally at `/tmp/otel-component-telemetry/<repo basename>` — do not clone it yourself. `Versions to scan` is either a count of recent releases or one exact tag. For a count, list that many released versions from `git tag`. For an exact tag, scan only that tag. When `Tag prefix` is non-empty, consider only tags beginning with that literal prefix, check out the full tag, and strip the prefix to obtain the version used in frontmatter and the output filename. For independently versioned packages, confirm the package version in its manifest or `version.py`, even when a repository release tag exists (e.g. Python repo tag `v1.16.0` contains instrumentation `0.37b0`). Use that package version for the filename and `version`, and `git rev-parse '<tag>^{commit}'` for the full `commit_sha`.

   For Python count requests, select recent instrumentation beta tags (`v<major>.<minor>b<revision>`); the historical SDK tag `v1.16.0` is supported only as an exact-tag request. Normalize the filename and `version` to exactly one leading `v`.

2. **Get source.** Check out each resolved tag into its own temp directory: `git worktree add "$(mktemp -d)" <tag>`. Never scan a branch — only pinned tags.

3. **Ground truth in the code.** Follow instrument/span/log construction through shared helpers and generated code. Record literal names after prefixes and namespaces are applied, exact units, and attributes from recording sites; semantic-convention definitions alone do not prove emission. Distinguish generated pipeline telemetry from the component's internal telemetry. Read the actual meter/tracer scope creation; when rows use different scopes, name them in `Notes` rather than assigning the component's path to all rows.

4. **Record conditionality.** If a signal, metric, or attribute is gated (feature gate, opt-in config, experimental semconv env var, framework/runtime version), it still gets a row — with the gate and default named in `Notes`. Check both branches and client/server recording sites separately. If emission is delegated to a runtime/library outside the pinned repository, identify that boundary and source instead of claiming its telemetry was verified here.
5. **Recheck before writing.** Compare each row to its construction and recording sites, specifically checking prefixes/separators, scopes, units, attribute sets, and disabled/default branches. List exact source paths present at the pinned commit; do not cite guessed paths or abbreviate them with `...`. Reconcile an existing component/version file rather than creating another path. Set `last_verified` to today's date only after this source check, never copy a date from another version. Run `python3 tools/otel-telemetry-emission-scan/scripts/validate.py --component '${Repo}/${Path}'` from the repository root. This checks provenance and source paths, not semantic completeness; fix failures before accepting output.

## Updates

- A version whose file already exists with a non-null verification date is **skipped**. A `last_verified: null` file needs a source recheck and reconciliation even without force.
- If your prompt says `force: true`, scan it anyway and **reconcile**: compare our findings with the existing file and edit only rows that are wrong or missing. Do not rewrite the whole file.

Never modify files of other components.

## Output

Output one file per **scanned version** under the output root given above, mirroring repo + path:

```
${OutDir}/<repo basename>/<path>/v<version>.md
```

Example: `${OutDir}/opentelemetry-collector-contrib/receiver/kafkareceiver/v0.158.0.md`

Follow the below format strictly:

````md
---
repo: $repo
path: $path
version: v$version
scope_name: $scope_name
commit_sha: $full_commit_sha
last_verified: $YYYY-MM-DD
---

# $component

## Traces

| Span name | Kind | Attributes | Notes |
|---|---|---|---|

## Metrics

| Metric name | Type | Unit | Attributes | Notes |
|---|---|---|---|---|

## Logs

| Log/event name | Attributes | Notes |
|---|---|---|

## Sources

- $files_consulted
````

Omit any signal section the component does not emit — never write empty tables. `Notes` names the gate when emission is conditional, `-` otherwise.
