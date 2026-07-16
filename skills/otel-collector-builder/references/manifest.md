# Builder Manifest Reference

Every key OCB accepts in `builder.yaml`, verified against the `Config` struct in [cmd/builder/internal/builder/config.go](https://github.com/open-telemetry/opentelemetry-collector/blob/main/cmd/builder/internal/builder/config.go).

## Contents

- [Top-level structure](#top-level-structure)
- [dist: distribution settings](#dist-distribution-settings)
- [Component specification](#component-specification)
- [providers and converters](#providers-and-converters)
- [conf_resolver](#conf_resolver)
- [telemetry](#telemetry)
- [replaces and excludes](#replaces-and-excludes)
- [Name collisions](#name-collisions)
- [Full example](#full-example)

## Top-level structure

```yaml
dist: {}          # distribution settings (see below)
receivers: []     # component lists — all five use the same module spec
processors: []
exporters: []
extensions: []
connectors: []
providers: []     # confmap providers (file, env, http, https, yaml)
converters: []    # confmap converters (rarely needed; no in-tree modules currently)
telemetry: {}     # single module (not a list): custom telemetry provider, advanced
conf_resolver: {} # default_uri_scheme
replaces: []      # go.mod replace directives
excludes: []      # go.mod exclude directives
```

## dist: distribution settings

| Key | Default | Purpose |
|-----|---------|---------|
| `module` | `go.opentelemetry.io/collector/cmd/builder` | Go module path of the generated distribution. Set it to your own path (`github.com/myorg/otelcol-custom`). |
| `name` | `otelcol-custom` | Binary name. |
| `description` | `Custom OpenTelemetry Collector distribution` | Shown in `--version`/components output. |
| `output_path` | current directory | Where sources and the binary are written. |
| `version` | `1.0.0` | Your distribution's version, shown in `--version`. Independent of the Collector version. |
| `go` | `go` from PATH | Path to the Go binary used for the build. |
| `debug_compilation` | `false` | `true` keeps symbols and disables optimizations (`gcflags all=-N -l`) for Delve. Default builds strip symbols (`-s -w`). |
| `cgo_enabled` | `false` | Sets `CGO_ENABLED=1` for components needing native libraries. |
| `build_tags` | empty | Comma-separated Go build tags. |
| `use_absolute_replace_paths` | `false` | See below. |

**`use_absolute_replace_paths` (v0.151.0 breaking change):** when a component uses `path:` or a `replaces` entry points at a local directory, OCB writes a `replace` directive into the generated `go.mod`. Since v0.151.0 those paths are **relative** to `output_path` by default, making the generated tree portable (committable, buildable on another machine). Set `true` to restore the pre-v0.151.0 absolute-path behavior.

## Component specification

`receivers`, `processors`, `exporters`, `extensions`, and `connectors` all take the same module spec:

```yaml
receivers:
  - gomod: github.com/myorg/myreceiver v0.1.0   # required: "<module-path> <version>"
    import: github.com/myorg/myreceiver          # optional: import path if != module path
    name: myreceiver                             # optional: package alias in generated code
    path: ../myreceiver                          # optional: local dir; emits a replace directive
```

- `gomod` is required; version needs the `v` prefix.
- `import` defaults to the `gomod` path — needed only when the factory package lives in a subpackage of the module.
- `path` makes OCB build against a local checkout instead of downloading; the `gomod` version is still required syntactically.

## providers and converters

Providers implement config URI schemes. The built binary can only load config through schemes whose provider is compiled in:

```yaml
providers:
  - gomod: go.opentelemetry.io/collector/confmap/provider/fileprovider v1.62.0   # file: and bare paths
  - gomod: go.opentelemetry.io/collector/confmap/provider/envprovider v1.62.0    # env: (${env:VAR})
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpprovider v1.62.0   # http://
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpsprovider v1.62.0  # https://
  - gomod: go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.62.0   # yaml: (inline YAML)
```

Providers version on the stable `v1.y.0` stream (see the version-alignment section in SKILL.md for the pairing rule). Minimum sensible set: `fileprovider` + `envprovider`.

`converters:` accepts the same module spec and hooks confmap converters into config resolution. Core and contrib currently ship no standalone converter modules (`expandconverter` was removed after env-expansion moved into confmap core); the key is only useful for custom converters.

## conf_resolver

```yaml
conf_resolver:
  default_uri_scheme: env   # scheme applied to bare ${VAR} references
```

When unset, the generated collector falls back to `env` at runtime, so `${VAR}` behaves as `${env:VAR}`. Setting any scheme here requires the matching provider to be in `providers:` — the binary errors at startup otherwise.

## telemetry

Advanced, single module (not a list). Overrides the collector's internal-telemetry SDK provider — only needed for unusual targets (e.g. Wasm). If omitted, `otelconftelemetry` from `go.opentelemetry.io/collector/service` is used:

```yaml
telemetry:
  gomod: go.opentelemetry.io/collector/service v0.156.0
  import: go.opentelemetry.io/collector/service/telemetry/otelconftelemetry
```

## replaces and excludes

Passed through to the generated `go.mod`:

```yaml
replaces:
  - github.com/old/module => github.com/new/module v2.0.0     # swap a module
  - github.com/pinned/module => github.com/pinned/module v1.5.0  # pin a version
  - github.com/myorg/internal => ../internal                   # local path

excludes:
  - github.com/broken/module v0.5.0
```

Local paths in `replaces` follow the same relative/absolute encoding as `path:` (see `use_absolute_replace_paths`).

## Name collisions

Two components whose modules end in the same package name collide in the generated imports; OCB auto-suffixes (`processor`, `processor2`). Set explicit `name:` values instead so `components.go` stays readable:

```yaml
processors:
  - gomod: github.com/org-a/processor v1.0.0
    name: orgaprocessor
  - gomod: github.com/org-b/processor v1.0.0
    name: orgbprocessor
```

## Full example

```yaml
dist:
  module: github.com/acme/otelcol-acme
  name: otelcol-acme
  description: ACME production Collector
  output_path: ./dist
  version: 2.1.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.156.0
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver v0.156.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.156.0
  - gomod: go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.156.0
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor v0.156.0

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.156.0
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.156.0

extensions:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckv2extension v0.156.0
  - gomod: go.opentelemetry.io/collector/extension/zpagesextension v0.156.0

connectors:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector v0.156.0

providers:
  - gomod: go.opentelemetry.io/collector/confmap/provider/fileprovider v1.62.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/envprovider v1.62.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpsprovider v1.62.0

replaces:
  - github.com/acme/private-exporter => ../private-exporter
```
