# Runtime compatibility

Evidence and parser validation behind the SKILL.md "Parsers verified" table.

## Verified 2026-10-25

| Runtime | Full package | Schema | Validation |
|---|---|---|---|
| Go | `go.opentelemetry.io/contrib/otelconf` | v1.0.0-rc.3 | presence only |
| Java SDK | `opentelemetry-sdk-extension-declarative-config` (`-alpha` artifact, BOM version) | v1.2.0 | `0.4`/`1.x` accepted; anything but `"1.2"` warns |
| Java agent / Spring Boot starter | `opentelemetry-javaagent`, `opentelemetry-spring-boot-starter` | v1.2.0 | `0.4`/`1.x` accepted; anything but `"1.2"` warns |
| JavaScript | `@opentelemetry/configuration` | v1.1.0 | major must be 1; newer minor warns |
| Python | `opentelemetry-configuration` (SDK 1.45.1) | v1.1.0 | major must be 1; newer minor warns |
| Ruby | `opentelemetry-config` | v1.0.0-rc.3 | none (`file_format` not checked) |
| .NET SDK | none released ([#7658](https://github.com/open-telemetry/opentelemetry-dotnet/issues/7658)) | - | - |
| .NET auto-instrumentation | `opentelemetry-dotnet-instrumentation`, needs `OTEL_EXPERIMENTAL_FILE_BASED_CONFIGURATION_ENABLED=true` | hand-written model, no pin | none |

Evidence, at each release tag or in the published artifact:

- Go: `Makefile` `genjsonschema` downloads the `v1.0.0-rc.3` schema; CHANGELOG moved to rc.3 in
  contrib 1.41.0 (#8505) with no later bump. The fixture `testdata/v1.0.0.yaml` says `"1.0-rc.2"`.
- Java SDK: `build.gradle.kts` `configurationTag = "1.2.0"`;
  `OpenTelemetryConfigurationFactory` `EXPECTED_FILE_FORMAT = "1.2"`.
- Java agent: `dependencyManagement/build.gradle.kts` `otelSdkVersion = "1.67.0"`, so the embedded
  SDK parser above applies; the agent `smoke-tests` fixture still uses `1.1`, while the
  `smoke-tests-otel-starter` fixtures use `"1.2"`.
- JavaScript: `scripts/generate-config.js` `CONFIG_VERSION = 'v1.1.0'`.
- Python: the wheel's vendored `schema.json` equals schema `v1.1.0`.
- Ruby: the gem's `generated_constants.rb` names schema `v1.0.0-rc.3`; `file_format` is only copied
  into the model.
- .NET auto-instrumentation: `docs/file-based-configuration.md` enables it with the flag above and
  reads `OTEL_CONFIG_FILE`; `YamlConfiguration.FileFormat` is never checked.

For a file shared by several runtimes, use the lowest of the selected parsers' target literals (a
newer literal may load with a warning in an older parser but is not supported there), keep to
fields every selected runtime supports (its schema, or for .NET auto-instrumentation its documented
model), and name the runtimes that will warn.
