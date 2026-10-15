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
      - uses: actions/checkout@v5

      - name: Validate the registry
        env:
          WORKSPACE: ${{ github.workspace }}
        run: |
          docker run --rm \
            -v "$WORKSPACE:/work" \
            otel/weaver:v0.26.1 \
            registry check \
              --v2 \
              --registry /work/telemetry/registry/

      - name: Generate and verify checked-in code is current
        env:
          WORKSPACE: ${{ github.workspace }}
        run: |
          docker run --rm \
            -v "$WORKSPACE:/work" \
            otel/weaver:v0.26.1 \
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
            otel/weaver:v0.26.1 \
            registry diff \
              --v2 \
              --baseline-registry /baseline/telemetry/registry/ \
              --registry /work/telemetry/registry/ \
              --format markdown
```

## Notes

- **Pin the Weaver version.** `otel/weaver:v0.26.1` (or the version your registry was authored against). Don't use `latest` — schema validation behavior changes between versions.
- **Format before diffing.** Without `gofmt -w` (or your language's formatter), Jinja whitespace produces multi-blank-line diffs that fail the `git diff --exit-code` gate spuriously.
- **`--future` is opt-in but still breaks custom `definition/2` registries.** It elevates the format's own instability warning to an error ([Version compatibility](#version-compatibility)). Re-enable it once the format goes stable.
- **Expected diagnostic noise.** `weaver registry check` emits `File format definition/2 is not yet stable` for each custom v2 definition file. Do not treat these warnings as a failure.
- **The diff job is informational.** It produces markdown breaking-change output; failing the build on it is too aggressive while a registry is young. Promote to a hard gate once the registry stabilizes.

## Version compatibility

### Verified 2026-10-15

Offline, cite this date, never call a value currently latest, and confirm it against the evidence before deploying.

| Fact | Value |
|---|---|
| This skill's examples | authored against `v0.26.1`; the next row re-checked at `v0.28.0` (released 2026-10-14) |
| `definition/2` status | not stable through `v0.28.0`: each file warns `File format definition/2 is not yet stable`; `--future` turns that warning into an error |
| Weaver pinned by semantic-conventions | `v1.43.0`: `v0.24.2`; `v1.44.0`: `v0.25.1`; `main`: `v0.28.0` |
| semconv `v1.44.0` model with Weaver `v0.28.0` | `weaver registry check -r model` exits 0; its `definition/2` files only warn |

Changelog-marked breaking changes after `v0.26.1` are in `registry live-check`, `registry infer`, and `registry mcp` (`v0.27.0`); `v0.28.0` warns when an entity `identity` reference sets `requirement_level`.

Evidence: Weaver source and `CHANGELOG.md` at `v0.28.0`; `dependencies.Dockerfile` at each semantic-conventions tag; a `v0.28.0` release-binary run on the `v1.44.0` tag.

## Local equivalent

```bash
docker run --rm -v "$PWD:/work" otel/weaver:v0.26.1 \
  registry check --v2 --registry /work/telemetry/registry/

docker run --rm -v "$PWD:/work" otel/weaver:v0.26.1 \
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
