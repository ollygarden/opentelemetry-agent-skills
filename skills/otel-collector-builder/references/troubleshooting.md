# OCB Troubleshooting

Common OCB build and runtime failures, most frequent first.

## Contents

- [Version mismatches](#version-mismatches)
- [Missing providers at runtime](#missing-providers-at-runtime)
- [Module resolution failures](#module-resolution-failures)
- [Local path issues](#local-path-issues)
- [Compilation failures](#compilation-failures)
- [Runtime failures](#runtime-failures)
- [Debugging techniques](#debugging-techniques)

## Version mismatches

Symptoms: `go mod tidy` errors during the "get modules" step, `there is a mismatch in go.mod and builder configuration versions`, or components silently resolved to unexpected versions.

Fix: align every core and contrib component on the same `v0.x.0` matching the OCB version, and providers on the paired `v1.y.0` (pairing rule in SKILL.md). Common mistakes:

- One component pinned at an older `v0.x.0` than the rest — Go resolves the highest requested version and the manifest no longer matches reality.
- Version written without the `v` prefix (`0.156.0`) — module query fails outright.
- OCB binary older than the component versions in the manifest — upgrade OCB first; the generated templates encode assumptions about the core APIs of the matching release.

Strict checking is off by default; run with `--skip-strict-versioning=false` to make OCB verify resolved versions against the manifest instead of failing later in `go build`.

## Missing providers at runtime

```
cannot load configuration: no provider found for scheme "file"
```

The binary was built without the provider for that config URI scheme. This only happens when the manifest sets `providers:` explicitly — the key **replaces** OCB's default set (env, file, http, https, yaml), so listing only `fileprovider` removes `${env:VAR}` expansion. Add the missing provider to `providers:` (or drop the key to restore all defaults) and rebuild.

## Module resolution failures

- `no matching versions for query` / `404 Not Found`: module path typo, version doesn't exist (check the module's tags), or a private module — for private modules set `GOPRIVATE` and credentials, or use `path:`.
- `requires <mod> ... not required by ...` conflicts between components' transitive deps: add a `replaces:` entry pinning the conflicting module.
- Inconsistent state after repeated builds: `go clean -modcache`, delete `output_path`, rebuild.

## Local path issues

- `reading <path>/go.mod: no such file or directory`: `path:` doesn't point at a module root, or a relative path resolves differently than expected. OCB resolves the input `path:` from its process working directory, not from the manifest's directory. Since v0.151.0, the emitted `replace` directive is then rewritten relative to `output_path`.
- Generated tree breaks after being moved/committed: built with pre-v0.151.0 absolute replace paths — upgrade OCB, or set `dist.use_absolute_replace_paths: true` if something depends on the old layout.
- Module path in the local `go.mod` must exactly match the `gomod` path in the manifest.

## Compilation failures

- `cgo: C compiler "gcc" not found`: a component needs CGO. Either install a C toolchain and set `dist.cgo_enabled: true`, or drop the component. Default is CGO off.
- `build constraints exclude all Go files`: `dist.build_tags` or `GOOS`/`GOARCH` excludes everything — clear the tags or fix the target platform.
- OOM on small CI runners: the collector dependency graph is large. `GOMAXPROCS=2 ocb --config=builder.yaml`, or split into generate + `go build` stages.
- `exec: "go": executable file not found`: install Go or set `dist.go` to its path. OCB requires a current Go toolchain (it invokes `go mod tidy` with a recent `-compat` version).

## Runtime failures

- `unknown type: "<component>"` when starting the built binary: the collector config references a component that isn't in the manifest — the whole point of a custom distribution is that only manifest components exist. Add it and rebuild, and confirm with `./dist/<name> components`.
- Validate config against the actual binary, not otelcol-contrib: `./dist/<name> validate --config=config.yaml`.
- Crash loops: rebuild with `dist.debug_compilation: true` and run under Delve (`dlv exec ./dist/<name> -- --config=config.yaml`), or raise `service.telemetry.logs.level: debug` in the collector config.

## Debugging techniques

```bash
ocb --config=builder.yaml --verbose            # show the underlying go commands
ocb --config=builder.yaml --skip-compilation   # inspect generated code without compiling
cat dist/components.go                          # exactly which factories got registered
./dist/<name> components                        # component inventory of the built binary
cd dist && go list -m all                       # resolved module versions
```
