# OCB Workflows

Development, CI, and packaging workflows for OCB-built distributions.

## Contents

- [Local component development](#local-component-development)
- [Two-stage CI builds](#two-stage-ci-builds)
- [Docker builds](#docker-builds)
- [Multi-platform builds](#multi-platform-builds)
- [Custom build metadata (ldflags)](#custom-build-metadata-ldflags)
- [Relationship to opentelemetry-collector-releases](#relationship-to-opentelemetry-collector-releases)

## Local component development

Point a component at a local checkout with `path:`; OCB emits a `replace` directive instead of downloading:

```yaml
receivers:
  - gomod: github.com/myorg/myreceiver v0.1.0   # version still required syntactically
    path: ../myreceiver
```

Requirements:

- The local directory must contain a `go.mod` whose module path matches the `gomod` path.
- Run OCB with a Go toolchain new enough for both OCB and the selected modules. OCB v0.156.0 declares Go 1.25, generates a `go 1.25` module, and runs `go mod tidy -compat=1.25`; a local component that requires newer Go needs that newer toolchain.
- Since v0.151.0 the generated `replace` uses a path relative to `dist.output_path`, so the generated tree can be committed or moved. Set `dist.use_absolute_replace_paths: true` if external tooling expects absolute paths.

This is also the fastest verification loop when authoring a new component: manifest with the local component + `otlpreceiver` + `debugexporter`, build, run, send data with `telemetrygen`.

## Two-stage CI builds

Split generation from compilation so CI can cache modules, run linters on the generated code, or build reproducibly:

```bash
# Stage 1 (once, or on manifest change): generate sources
ocb --config=builder.yaml --skip-compilation
git add dist/ && git commit -m "regenerate collector sources"

# Stage 2 (every build): compile without touching generated code
ocb --config=builder.yaml --skip-generate --skip-get-modules
```

Since v0.154.0, `--skip-get-modules` also skips regenerating `go.mod`, so manual edits to the generated `go.mod` survive. Before v0.154.0 it was regenerated from the template, silently reverting edits.

Because the generated tree is a plain Go module, stage 2 can also be a plain `go build` in `dist/` — useful when CI already has a Go build pipeline:

```bash
cd dist && go build -o otelcol-custom .
```

## Docker builds

Run OCB in a container (no local Go needed):

```bash
docker run --rm -v "$(pwd):/work" -w /work \
  otel/opentelemetry-collector-builder:0.156.0 \
  --config=/work/builder.yaml
```

The working directory matters when the manifest uses relative `path:` or `output_path` values. Without `-w /work`, the released image runs from `/home/ocb`, so `./dist` is written inside the disposable container instead of the mounted project directory.

Typical multi-stage Dockerfile for shipping the result:

```dockerfile
FROM golang:1.25 AS build
WORKDIR /build
COPY builder.yaml .
RUN go install go.opentelemetry.io/collector/cmd/builder@v0.156.0 \
 && builder --config=builder.yaml

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /build/dist/otelcol-custom /otelcol-custom
COPY config.yaml /etc/otelcol/config.yaml
ENTRYPOINT ["/otelcol-custom"]
CMD ["--config", "/etc/otelcol/config.yaml"]
EXPOSE 4317 4318
```

Distroless/static works because CGO is off by default. If `dist.cgo_enabled: true`, use a base image with the needed shared libraries instead.

## Multi-platform builds

OCB honors the standard Go cross-compilation environment:

```bash
GOOS=linux GOARCH=amd64 ocb --config=builder.yaml
GOOS=linux GOARCH=arm64 ocb --config=builder.yaml
GOOS=windows GOARCH=amd64 ocb --config=builder.yaml   # generated main_windows.go adds Windows service support
```

Use a distinct `dist.output_path` per platform (or move the binary between builds) — the binary name doesn't encode the platform. Cross-compiling with `cgo_enabled: true` additionally needs a target-platform C toolchain.

## Custom build metadata (ldflags)

Default builds strip symbols (`-s -w`). Override to inject build info:

```bash
ocb --config=builder.yaml \
  --ldflags="-s -w -X main.version=$(git describe --tags)"
```

Note `--ldflags` replaces the default entirely — re-add `-s -w` if you still want stripped binaries. `--gcflags` passes through similarly. For debugger-friendly builds prefer `dist.debug_compilation: true` over hand-rolled flags.

## Relationship to opentelemetry-collector-releases

[opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases) is where the official distributions (`otelcol`, `otelcol-contrib`, `otelcol-k8s`, `otelcol-otlp`) are defined and built — each is just an OCB manifest under `distributions/<name>/manifest.yaml` plus goreleaser packaging (archives, deb/rpm, container images, signing).

Use it two ways:

- **Starting point**: copy the manifest of the distribution closest to your needs and prune/add components, instead of writing a manifest from scratch. The `otelcol-contrib` manifest is the authoritative list of every component name + module path at a given version.
- **Packaging model**: if you need deb/rpm/systemd packaging or signed multi-arch images for your own distribution, its goreleaser configuration is the reference implementation to mirror.

OCB release binaries (`ocb`) are also published from that repository's releases page, tagged `cmd/builder/vX.Y.Z`.
