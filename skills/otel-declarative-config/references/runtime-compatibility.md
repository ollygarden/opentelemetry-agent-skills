# Runtime compatibility

Which configuration-schema release each released parser targets, and the `file_format` literal to
write for it. Offline, report a row with its verification date and never call it currently latest.
Online, look up the selected package's latest version: if it matches the row, the row stands;
otherwise re-verify from its evidence before generating deployable YAML.

## Verified 2026-10-15

The schema repository's latest release is `v1.2.0`; parsers trail it.

| Runtime | Package and version | Schema | `file_format` | Validation |
|---|---|---|---|---|
| Go | `go.opentelemetry.io/contrib/otelconf` v0.27.0 | v1.0.0-rc.3 | `"1.0-rc.3"` | presence only |
| Java SDK | `opentelemetry-sdk-extension-declarative-config` 1.67.0-alpha (BOM 1.67.0) | v1.2.0 | `"1.2"` | `0.4`/`1.x` accepted; anything but `"1.2"` warns |
| Java agent, Spring Boot starter | 2.32.0 (embeds SDK 1.66.0) | v1.1.0 | `"1.1"` | `0.4`/`1.x` accepted; prefers `"1.1"` |
| JavaScript | `@opentelemetry/configuration` 0.223.0 | v1.1.0 | `"1.1"` | major must be 1; newer minor warns |
| Python | `opentelemetry-configuration` 0.66b1 (SDK 1.45.1) | v1.1.0 | `"1.1"` | major must be 1; newer minor warns |
| Ruby | `opentelemetry-config` 0.1.1 | v1.0.0-rc.3 | `"1.0"` (README) | none |
| .NET SDK | none released ([#7658](https://github.com/open-telemetry/opentelemetry-dotnet/issues/7658)) | - | - | - |
| .NET auto-instrumentation | `opentelemetry-dotnet-instrumentation` v1.17.0, experimental, needs `OTEL_EXPERIMENTAL_FILE_BASED_CONFIGURATION_ENABLED=true` | hand-written model, no pin | `"1.0"` (docs) | none |

Evidence, at each release tag or in the published artifact:

- Go: `Makefile` `genjsonschema` downloads the `v1.0.0-rc.3` schema; CHANGELOG moved to rc.3 in
  contrib 1.41.0 (#8505) with no later bump. The fixture `testdata/v1.0.0.yaml` says `"1.0-rc.2"`.
- Java SDK: `build.gradle.kts` `configurationTag = "1.2.0"`;
  `OpenTelemetryConfigurationFactory` `EXPECTED_FILE_FORMAT = "1.2"`.
- Java agent: `dependencyManagement/build.gradle.kts` `otelSdkVersion = "1.66.0"`; the
  `smoke-tests` declarative fixture uses `1.1`.
- JavaScript: `scripts/generate-config.js` `CONFIG_VERSION = 'v1.1.0'`.
- Python: the wheel's vendored `schema.json` equals schema `v1.1.0`.
- Ruby: the gem's `generated_constants.rb` names schema `v1.0.0-rc.3`.
- .NET auto-instrumentation: `docs/file-based-configuration.md` enables it with the flag above and
  reads `OTEL_CONFIG_FILE`; `YamlConfiguration.FileFormat` is never checked.

For a file shared by several runtimes, use the highest literal that every selected parser accepts
without error, and name the runtimes that will warn.
