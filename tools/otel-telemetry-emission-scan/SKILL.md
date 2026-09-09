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

1. **Resolve versions.** The repo is already cloned locally at `/tmp/otel-component-telemetry/<repo basename>` — do not clone it yourself. `Versions to scan` is either a count of recent releases or one exact tag. For a count, list that many released versions from `git tag`. For an exact tag, scan only that tag. When `Tag prefix` is non-empty, consider only tags beginning with that literal prefix, check out the full tag, and strip the prefix to obtain the version used in frontmatter and the output filename. In monorepos where packages version independently of repo tags and no tag prefix is provided (e.g. opentelemetry-js experimental packages), use the package's own version from its manifest at each release tag, and name files after the package version.

2. **Get source.** Check out each resolved tag into its own temp directory: `git worktree add "$(mktemp -d)" <tag>`. Never scan a branch — only pinned tags.

3. **Ground truth in the code.** Analyse the instrumentation inside a codebase deeply and identify all emitted instrumentation data points required to fill the table format below.

4. **Record conditionality.** If a signal, metric, or attribute is gated (feature gate, opt-in config, experimental semconv env var), it still gets a row — with the gate named in `Notes`.
5. **Apply the skip/force rule**, run the self-check, write the file(s).

## Updates

- A version whose file already exists is **skipped** — existence means scanned.
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
commit_sha: $tag_or_sha
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
