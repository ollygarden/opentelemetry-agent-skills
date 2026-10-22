# OpenTelemetry Version Index

Versions last changed: 2026-10-22 (UTC). Each version below was the latest release on that date; newer releases may exist.

## SDKs

| Language | Package/Repo | Latest | Release Source | Setup Docs | Examples |
|---|---|---|---|---|---|
| .NET | `OpenTelemetry` | `1.20.1` | [NuGet](https://www.nuget.org/packages/OpenTelemetry) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/dotnet/instrumentation.md) | [Examples](https://github.com/open-telemetry/opentelemetry-dotnet/tree/main/examples) |
| C++ | `open-telemetry/opentelemetry-cpp` | `1.29.0` | [GitHub releases](https://github.com/open-telemetry/opentelemetry-cpp/releases) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/cpp/instrumentation.md) | [Examples](https://github.com/open-telemetry/opentelemetry-cpp/tree/main/examples) |
| Erlang/Elixir | `opentelemetry` | `1.7.0` | [Hex.pm](https://hex.pm/packages/opentelemetry) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/erlang/instrumentation.md) | - |
| Go | `go.opentelemetry.io/otel/sdk` | `1.47.0` | [Go proxy](https://pkg.go.dev/go.opentelemetry.io/otel/sdk) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/go/instrumentation.md) | [Examples](https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/examples) |
| Java | `io.opentelemetry:opentelemetry-sdk` | `1.67.0` | [Maven Central](https://search.maven.org/artifact/io.opentelemetry/opentelemetry-sdk) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/java/sdk.md) | [Examples](https://github.com/open-telemetry/opentelemetry-java-examples#java-opentelemetry-examples) |
| JavaScript | `@opentelemetry/sdk-node` | `0.300.0` | [npm](https://www.npmjs.com/package/@opentelemetry/sdk-node) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/js/instrumentation.md) | [Examples](https://github.com/open-telemetry/opentelemetry-js/tree/main/examples) |
| Kotlin | `open-telemetry/opentelemetry-kotlin` | `0.9.0` | [GitHub releases](https://github.com/open-telemetry/opentelemetry-kotlin/releases) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry-kotlin/refs/heads/main/README.md) | [Examples](https://github.com/open-telemetry/opentelemetry-kotlin/tree/main/examples) |
| PHP | `open-telemetry/sdk` | `1.15.0` | [Packagist](https://packagist.org/packages/open-telemetry/sdk) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/php/instrumentation.md) | [Examples](https://github.com/open-telemetry/opentelemetry-php/tree/main/examples) |
| Python | `opentelemetry-sdk` | `1.45.1` | [PyPI](https://pypi.org/project/opentelemetry-sdk/) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/python/instrumentation.md) | - |
| Ruby | `opentelemetry-sdk` | `1.13.1` | [RubyGems](https://rubygems.org/gems/opentelemetry-sdk) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/ruby/instrumentation.md) | [Examples](https://github.com/open-telemetry/opentelemetry-ruby/tree/main/examples) |
| Rust | `opentelemetry_sdk` | `0.33.1` | [crates.io](https://crates.io/crates/opentelemetry_sdk) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/rust/getting-started.md) | [Examples](https://github.com/open-telemetry/opentelemetry-rust/tree/main/examples) |
| Swift | `open-telemetry/opentelemetry-swift` | `2.6.0` | [GitHub releases](https://github.com/open-telemetry/opentelemetry-swift/releases) | [Docs](https://raw.githubusercontent.com/open-telemetry/opentelemetry.io/refs/heads/main/content/en/docs/languages/swift/instrumentation.md) | [Examples](https://github.com/open-telemetry/opentelemetry-swift/tree/main/Examples) |

## Components

| Component | Latest | Release | Notes |
|---|---|---|---|
| Collector core | `1.69.0` / `0.163.0` | [Notes](https://github.com/open-telemetry/opentelemetry-collector/releases/tag/v0.163.0) | Stable `1.x` modules / other `0.x` modules |
| Collector contrib | `1.2.0` / `0.163.0` | [Notes](https://github.com/open-telemetry/opentelemetry-collector-contrib/releases/tag/v0.163.0) | Stable `1.x` modules / other `0.x` modules |
| Collector distributions | `0.163.2` | [Notes](https://github.com/open-telemetry/opentelemetry-collector-releases/releases/tag/v0.163.2) | Binaries and images (`otelcol`, `otelcol-contrib`, `otelcol-k8s`, ...); can trail core |
| Collector builder (OCB) | `0.163.2` | [Notes](https://github.com/open-telemetry/opentelemetry-collector-releases/releases/tag/cmd/builder/v0.163.2) | `ocb` binary; can trail core |
| Collector Helm chart | `0.176.1` | [Notes](https://github.com/open-telemetry/opentelemetry-helm-charts/releases/tag/opentelemetry-collector-0.176.1) | `opentelemetry-collector` chart; chart version is independent of the Collector |
| Operator | `0.161.0` | [Notes](https://github.com/open-telemetry/opentelemetry-operator/releases/tag/v0.161.0) | Kubernetes Operator; version is independent of the Collector |
| Java agent | `2.32.0` | [Notes](https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/tag/v2.32.0) | `opentelemetry-javaagent` 2.x; embeds its own SDK version |
| Configuration schema | `1.2.0` | [Notes](https://github.com/open-telemetry/opentelemetry-configuration/releases/tag/v1.2.0) | Declarative config schema; SDK parsers may target older releases |
| Semantic conventions | `1.44.0` | [Notes](https://github.com/open-telemetry/semantic-conventions/releases/tag/v1.44.0) | Specification release; language semconv packages may lag |
| Specification | `1.61.0` | [Notes](https://github.com/open-telemetry/opentelemetry-specification/releases/tag/v1.61.0) | OpenTelemetry specification release |
| Weaver | `0.28.0` | [Notes](https://github.com/open-telemetry/weaver/releases/tag/v0.28.0) | Semantic-convention registry CLI |
