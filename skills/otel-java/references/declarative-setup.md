# Java SDK Setup with Declarative Configuration

Configure the OpenTelemetry SDK in Java via declarative YAML configuration. Three setup
paths exist (Javaagent, Spring Boot Starter, manual autoconfigure); all support
`-Dotel.config.file`.

For the YAML configuration schema, load the `otel-declarative-config` skill.

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config`
skill. For Java-specific facts:

| Fact | Fetch |
|---|---|
| Latest BOM (`opentelemetry-bom`) | `gh api repos/open-telemetry/opentelemetry-java/releases/latest -q '.tag_name'` |
| Latest Javaagent | `gh api repos/open-telemetry/opentelemetry-java-instrumentation/releases/latest -q '.tag_name'` |
| Javaagent declarative-config docs (current activation flag, supported `file_format`) | `WebFetch https://opentelemetry.io/docs/zero-code/java/agent/configuration/` |
| Javaagent CHANGELOG (when each schema rc landed) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-java-instrumentation/main/CHANGELOG.md` |
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

Declarative config support landed in Javaagent 2.26.0. Newer agent versions track newer
schema versions — fetch the agent CHANGELOG (see Sources of Truth) to confirm which
`file_format` your agent accepts.

When `-Dotel.config.file` is set, all other `-Dotel.*` properties are ignored except
agent-only properties (see Key API Facts).

For the autoconfigure SDK extension (no Javaagent), the same flag works, plus the
`OTEL_CONFIG_FILE` environment variable as a fallback.

## YAML Config

For the canonical structure and the correct `file_format` string for your agent version,
fetch `examples/otel-sdk-config.yaml` and `language-support-status.md` (see the
`otel-declarative-config` skill's Sources of Truth). The minimal example below illustrates
the Java-specific quirk.

```yaml
# file_format: pick from language-support-status.md based on your Javaagent version
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
AutoConfiguredOpenTelemetrySdk sdk = AutoConfiguredOpenTelemetrySdk.builder().build();
```

## Key API Facts

- **Spring Boot Starter activation**: `otel.config.file=configs/otel.yaml` in `application.properties`.
- **Shutdown hook**: The Javaagent and autoconfigure both register a JVM shutdown hook automatically — no manual `sdk.close()` needed.
- **Agent-only properties**: `otel.javaagent.extensions`, `otel.javaagent.enabled`, `otel.javaagent.debug` cannot be set via declarative config — they must remain as system properties.
