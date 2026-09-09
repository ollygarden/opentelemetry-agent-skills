# Java SDK Setup with Declarative Configuration

Configure the OpenTelemetry SDK in Java via declarative YAML configuration. Three setup
paths exist. The Javaagent and manual autoconfigure both read an external file via
the `otel.config.file` system property or `OTEL_CONFIG_FILE` environment variable. The Spring Boot
Starter is different: it embeds the declarative config inline under the `otel:` key in
`application.yaml`/`application.properties`, opting in via the `otel.file_format` property — it
does not read an external `otel.config.file` (see Key API Facts).

For the YAML configuration schema, load the `otel-declarative-config` skill.

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config`
skill. For Java-specific facts:

| Fact | Fetch |
|---|---|
| Latest BOM (`opentelemetry-bom`) | `gh api repos/open-telemetry/opentelemetry-java/releases/latest -q '.tag_name'` |
| Latest Javaagent | `gh api repos/open-telemetry/opentelemetry-java-instrumentation/releases/latest -q '.tag_name'` |
| SDK declarative-config accepted and preferred `file_format` for a selected BOM tag | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java/<selected-sdk-tag>/sdk-extensions/declarative-config/src/main/java/io/opentelemetry/sdk/autoconfigure/declarativeconfig/OpenTelemetryConfigurationFactory.java` |
| Javaagent declarative-config docs (current activation flag, supported `file_format`) | `WebFetch https://opentelemetry.io/docs/zero-code/java/agent/declarative-configuration/` |
| Javaagent declarative-config smoke fixture (parser truth for selected agent tag) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/smoke-tests/src/test/resources/declarative-config.yaml` |
| Javaagent CHANGELOG (when each schema rc landed) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/CHANGELOG.md` |
| Generated Java instrumentation config reference (names, types, defaults, deprecations) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/docs/declarative-configuration-example.yaml` |
| Spring Boot Starter declarative-config fixture (selected starter tag) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/<selected-agent-tag>/smoke-tests-otel-starter/spring-boot-2/src/testDeclarativeConfig/resources/application.yaml` |
| Spring Boot starter docs | `WebFetch https://opentelemetry.io/docs/zero-code/java/spring-boot-starter/` |

## Javaagent Download

The Javaagent JAR has no compile dependencies for auto-instrumentation. Download the latest
release directly:

```bash
curl -L -o opentelemetry-javaagent.jar \
  https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
```

## Activation

```bash
java -javaagent:opentelemetry-javaagent.jar \
     -Dotel.config.file=configs/otel.yaml \
     -jar myservice.jar
```

Declarative config has been supported since Javaagent 2.9.0; the property is now the stable
`otel.config.file` (the experimental `otel.experimental.config.file` alias was removed in the
SDK 1.63.0 bundled with Javaagent 2.29.0). Newer agent versions track newer schema versions.
Confirm both the accepted range and preferred `file_format` from the tag-matched parser, then use
the preferred value from that release's fixture to avoid compatibility warnings for experimental
properties. As of 2026-09-09, the latest SDK BOM is 1.65.0, and Javaagent/Spring Boot Starter
2.31.1 target SDK 1.65.0. The released SDK parser accepts `0.4` and `1.*` and prefers
`"1.1"`; both 2.31.1 instrumentation fixtures use `1.1`. Keep the selected distribution's
embedded SDK distinct from the independently released BOM, and do not infer either from `main` or
the generic language support matrix alone.

When `otel.config.file` / `OTEL_CONFIG_FILE` is set, all other SDK autoconfigure properties are
ignored except agent-only properties (see Key API Facts).

The same property and environment-variable forms work for the autoconfigure SDK extension without
the Javaagent.

## YAML Config

For the canonical structure, fetch `examples/otel-sdk-config.yaml` (see the
`otel-declarative-config` skill's Sources of Truth). For the correct `file_format` string,
use the selected Javaagent parser/docs/fixtures. The generic language support matrix is
coverage metadata and may not be the exact YAML literal accepted by the Javaagent.

```yaml
# file_format: use the preferred literal for the selected Javaagent version
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-myservice}"
    - name: deployment.environment.name
      value: "${DEPLOY_ENV:-development}"

# Tracer/meter/logger provider blocks: structure per the canonical example.
# Java-specific quirk: all duration values must be in milliseconds (e.g., 5000, not "5s").
```

## Adding Manual Instrumentation

The Javaagent registers `GlobalOpenTelemetry` automatically. Use it to get tracers/meters:

```java
import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.api.metrics.Meter;

Tracer tracer = GlobalOpenTelemetry.getTracer("mycompany.com/myservice");
Meter meter = GlobalOpenTelemetry.getMeter("mycompany.com/myservice");
```

For autoconfigure setups (no Javaagent):

```java
import io.opentelemetry.sdk.autoconfigure.AutoConfiguredOpenTelemetrySdk;

// Reads -Dotel.config.file or OTEL_CONFIG_FILE, falls back to env vars
AutoConfiguredOpenTelemetrySdk sdk =
    AutoConfiguredOpenTelemetrySdk.builder().setResultAsGlobal().build();
```

## Key API Facts

- **Spring Boot Starter activation**: unlike the Javaagent/autoconfigure, the starter does
  not load an external file. Embed the declarative config inline under the `otel:` key in
  `application.yaml` (or as `otel.*` properties in `application.properties`) and opt in by
  setting `otel.file_format` (for example, `file_format: "1.1"` for starter 2.31.1; verify the
  selected release fixture). The presence of `otel.file_format` is what switches the starter into
  declarative-config mode.
- **Shutdown hook**: The Javaagent and autoconfigure both register a JVM shutdown hook automatically — no manual `sdk.close()` needed.
- **Zipkin exporter removal**: SDK/BOM 1.65.0 no longer publishes
  `opentelemetry-exporter-zipkin`. It was previously deprecated; migrate direct Zipkin export to
  OTLP or send OTLP to a Collector with a Zipkin-compatible exporter before upgrading. Javaagent
  2.31.1 still accepts `otel.traces.exporter=zipkin` and `otel.exporter.zipkin.endpoint`, but both
  are deprecated in favor of OTLP configuration.
- **Agent-only properties**: `otel.javaagent.extensions`, `otel.javaagent.enabled`, and
  `otel.javaagent.debug` cannot be set via declarative config. Set them as system properties or
  their corresponding environment variables instead.
- **Semantic-convention selection**: In 2.31.1 declarative config, prefer
  `instrumentation/development.general.stability_opt_in_list`, a comma-separated string such as
  `database,code,messaging` or `<group>/dup` for dual emission. The older
  `general.semconv_stability.opt_in` field is deprecated. Check the tag-matched generated config
  and instrumentation metadata because support differs by domain and instrumentation.
- **Starter thread details**: Starter 2.31.1 can add experimental `thread.id` and `thread.name`
  to spans with `distribution.spring_starter.thread_details_enabled: true`. This path is
  Starter-only; the Javaagent uses the separate
  `distribution.javaagent.thread_details_enabled` path.
