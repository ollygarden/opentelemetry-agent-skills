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
            otel/weaver:v0.22.1 \
            registry check \
              --registry /work/telemetry/registry/

      - name: Generate and verify checked-in code is current
        env:
          WORKSPACE: ${{ github.workspace }}
        run: |
          docker run --rm \
            -v "$WORKSPACE:/work" \
            otel/weaver:v0.22.1 \
            registry generate \
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
            otel/weaver:v0.22.1 \
            registry diff \
              --baseline-registry /baseline/telemetry/registry/ \
              --registry /work/telemetry/registry/ \
              --diff-format markdown
```

## Notes

- **Pin the Weaver version.** `otel/weaver:v0.22.1` (or the version your registry was authored against). Don't use `latest` — schema validation behavior changes between versions.
- **Format before diffing.** Without `gofmt -w` (or your language's formatter), Jinja whitespace produces multi-blank-line diffs that fail the `git diff --exit-code` gate spuriously.
- **`--future` is opt-in but breaks today.** It enables upcoming validation rules — useful for catching tightening rules early — but at 0.22.1 it errors on `definition/2` itself. Re-enable once the format goes stable.
- **Expected stderr noise.** `weaver registry check` emits `File format definition/2 is not yet stable`. Do not treat it as a failure.
- **The diff job is informational.** It produces markdown breaking-change output; failing the build on it is too aggressive while a registry is young. Promote to a hard gate once the registry stabilizes.

## Local equivalent

```bash
docker run --rm -v "$PWD:/work" otel/weaver:v0.22.1 \
  registry check --registry /work/telemetry/registry/

docker run --rm -v "$PWD:/work" otel/weaver:v0.22.1 \
  registry generate \
    --registry /work/telemetry/registry/ \
    --templates /work/telemetry/templates/ \
    go \
    /work/internal/telemetry/

gofmt -w internal/telemetry/
git diff -- internal/telemetry/
```

A pinned local binary from the [Weaver releases page](https://github.com/open-telemetry/weaver/releases) avoids the Docker startup cost while developing.
