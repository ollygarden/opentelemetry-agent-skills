# CI Integration

Three gates protect the telemetry contract:

1. **Schema check** — registry parses and validates.
2. **Codegen freshness** — checked-in generated files match what `weaver` produces today.
3. **Breaking-change diff** — surfaces removed/modified entries against the base branch.

## GitHub Actions example

```yaml
name: telemetry

on:
  pull_request:
    paths:
      - "telemetry/**"
      - "internal/telemetry/**_gen.go"
      - ".github/workflows/telemetry.yml"

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - name: Validate the registry
        env:
          WORKSPACE: ${{ github.workspace }}
        run: |
          docker run --rm \
            -v "$WORKSPACE:/work" \
            otel/weaver:v0.28.0 \
            registry check \
              --v2 \
              --registry /work/telemetry/registry/

      - name: Generate and verify checked-in code is current
        env:
          WORKSPACE: ${{ github.workspace }}
        run: |
          docker run --rm \
            --user "$(id -u):$(id -g)" \
            -v "$WORKSPACE:/work" \
            otel/weaver:v0.28.0 \
            registry generate \
              --v2 \
              --registry /work/telemetry/registry/ \
              --templates /work/telemetry/templates/ \
              go \
              /work/internal/telemetry/
          gofmt -w internal/telemetry/
          git diff --exit-code internal/telemetry/

      - name: Diff against base branch
        if: github.event_name == 'pull_request'
        env:
          WORKSPACE: ${{ github.workspace }}
          BASE_REF: ${{ github.base_ref }}
        run: |
          git fetch origin "$BASE_REF":__base
          git worktree add /tmp/base __base
          docker run --rm \
            -v "$WORKSPACE:/work" \
            -v /tmp/base:/baseline \
            otel/weaver:v0.28.0 \
            registry diff \
              --v2 \
              --baseline-registry /baseline/telemetry/registry/ \
              --registry /work/telemetry/registry/ \
              --format markdown
```

## Notes

- **Pin the Weaver version.** `otel/weaver:v0.28.0` (or the version your registry was authored against). Don't use `latest` — schema validation behavior changes between versions.
- **Run `generate` as the checkout's owner.** The image's default user (`weaver`) need not match the runner's uid; without `--user "$(id -u):$(id -g)"`, writing into the mounted checkout can fail with `Permission denied`.
- **Format before diffing.** Without `gofmt -w` (or your language's formatter), Jinja whitespace produces multi-blank-line diffs that fail the `git diff --exit-code` gate spuriously.
- **`--future` is opt-in but still breaks custom `definition/2` registries.** It elevates the format's own instability warning to an error ([Version compatibility](#version-compatibility)). Re-enable it once the format goes stable.
- **Expected diagnostic noise.** `weaver registry check` emits `File format definition/2 is not yet stable` for each custom v2 definition file. Do not treat these warnings as a failure. A missing manifest `file_format` warning is not noise: add `file_format: definition_manifest/2.0`.
- **Policy loading.** `check` and `generate` load every `*.rego` in the registry directory plus each `--policy` path. Weaver's own Rego is only the `defaults/rego/semconv.rego` helper library and built-in `live_check_advice` rules.
- **The diff job is informational.** It produces markdown breaking-change output; failing the build on it is too aggressive while a registry is young. Promote to a hard gate once the registry stabilizes.

## Version compatibility

### Verified 2026-11-22

Offline, cite this date, never call a value currently latest, and confirm it against the evidence before deploying.

| Fact | Value |
|---|---|
| This skill's examples | `check`, `generate`, and `diff` run clean (warnings only) on `v0.28.0` (released 2026-10-14); `otel/weaver:v0.28.0` image exists |
| Manifest `file_format: definition_manifest/2.0` | accepted from `v0.27.0`; `v0.26.1` rejects it; missing it warns on `v0.27.0`+ and errors with `--future` |
| Manifest fields and presence (`v0.28.0`) | `manifest.yaml` rejects the legacy top-level `semconv_version`/`schema_base_url` (a legacy-named `registry_manifest.yaml` still accepts them with a deprecation warning); top-level `name` is not a v2 manifest field; a missing manifest only logs `No registry manifest found` |
| `definition/2` status | not stable through `v0.28.0`: each file warns `File format definition/2 is not yet stable`; `--future` turns that warning into an error |
| `--registry .` or `--registry ..` | `v0.28.0` loads the manifest but no definition files: `check` passes vacuously (registry-dir policies never fire) and `generate` writes nothing. `./` or any other spelling works |
| Weaver pinned by semantic-conventions | `v1.43.0`: `v0.24.2`; `v1.44.0`: `v0.25.1`; `v1.45.0`: `v0.28.0` |
| semconv `v1.44.0` and `v1.45.0` models with Weaver `v0.28.0` | `weaver registry check -r model` (with or without `--v2`) exits 0; warnings only: its `definition/2` files, its manifest (no `file_format`), and in `v1.45.0` signals without `requirement_level`; `--future` exits 1 |

Changelog-marked breaking changes after `v0.26.1` are in `registry live-check`, `registry infer`, and `registry mcp` (`v0.27.0`); `v0.28.0` warns when an entity `identity` reference sets `requirement_level`. When moving a pin up from `v0.26.x`, add the manifest `file_format` in the same change, not before (`v0.26.x` rejects it).

Evidence: Weaver source at `v0.26.1` and `v0.28.0`, `CHANGELOG.md`, and `schemas/definition-manifest.v2.json` at `v0.28.0`; `dependencies.Dockerfile` at each semantic-conventions tag; `otel/weaver` `v0.26.0`, `v0.27.0`, and `v0.28.0` runs on this skill's examples; `v0.28.0` binary runs on the semconv `v1.44.0` and `v1.45.0` models.

## Local equivalent

```bash
docker run --rm -v "$PWD:/work" otel/weaver:v0.28.0 \
  registry check --v2 --registry /work/telemetry/registry/

docker run --rm --user "$(id -u):$(id -g)" -v "$PWD:/work" otel/weaver:v0.28.0 \
  registry generate \
    --v2 \
    --registry /work/telemetry/registry/ \
    --templates /work/telemetry/templates/ \
    go \
    /work/internal/telemetry/

gofmt -w internal/telemetry/
git diff -- internal/telemetry/
```

A pinned local binary from the [Weaver releases page](https://github.com/open-telemetry/weaver/releases) avoids the Docker startup cost while developing.
