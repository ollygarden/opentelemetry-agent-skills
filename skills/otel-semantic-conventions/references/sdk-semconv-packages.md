# SDK semantic-convention packages

Which semantic-conventions release each language's semconv package exposes. Packages trail the
spec, so pick the version your SDK can emit, not the newest spec release.

## Verified 2026-10-15

Offline, cite this date, never call a value currently latest, and confirm it against the evidence before deploying. Online, re-check the package registry first.

Latest spec release: the `otel-sdk-versions` index if it lists semantic-conventions, else the GitHub releases.

| Language | Package and version | Newest semconv exposed |
|---|---|---|
| Go | `go.opentelemetry.io/otel` v1.47.0, packages `go.opentelemetry.io/otel/semconv/vX` | `v1.43.0` (added in v1.45.0; no `v1.44.0` package) |
| Java | `io.opentelemetry.semconv:opentelemetry-semconv` 1.44.0 (`-incubating` 1.44.0-alpha) | 1.44.0 |
| JavaScript | `@opentelemetry/semantic-conventions` 1.43.0 | 1.43.0 |
| Python | `opentelemetry-semantic-conventions` 0.66b1 | 1.44.0 |
| Ruby | `opentelemetry-semantic_conventions` 1.44.0 | 1.44.0 |
| PHP | `open-telemetry/sem-conv` 1.44.0 | 1.44.0 |
| C++ | `opentelemetry-cpp` v1.29.0 headers | 1.44.0 |
| Rust | `opentelemetry-semantic-conventions` 0.33.1 | 1.42.0 |
| .NET | `OpenTelemetry.SemanticConventions`: only the unlisted 1.0.0-rc9.9 on NuGet; unreleased dotnet-contrib source tracks 1.44.0 | - (no current release) |

Evidence, at each release tag or in the published artifact:

- Go: `semconv/` directories at tag `v1.47.0`; CHANGELOG `1.45.0` adds `semconv/v1.43.0`.
- Java: the jar's `SchemaUrls` highest constant is `schemas/1.44.0`.
- JavaScript: package `VERSION = '1.43.0'`; its CHANGELOG `1.43.0` cites Semantic Conventions v1.43.0.
- Python: the wheel's `opentelemetry/semconv/schemas.py` highest member is `V1_44_0`.
- Ruby: the gem's `semantic_conventions/version.rb` `SPEC_VERSION = '1.44.0'`.
- PHP: `Version.php` at tag `1.44.0` defines `VERSION_1_44_0`.
- C++: `api/include/opentelemetry/semconv/schema_url.h` at `v1.29.0`.
- Rust: the crate's `SCHEMA_URL` is `schemas/1.42.0`.
- .NET: NuGet registration (`listed: false`); dotnet-contrib `src/OpenTelemetry.SemanticConventions/CHANGELOG.md` Unreleased (#5121).

For services in several languages, the shared version is the lowest "Newest semconv exposed" value
among the languages in use (1.43.0 for Go plus JavaScript, 1.42.0 with Rust). .NET has no value, so
it cannot join a shared version.
