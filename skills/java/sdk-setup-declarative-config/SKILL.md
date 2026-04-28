---
name: java-sdk-setup-declarative-config
description: Set up OpenTelemetry SDK in Java applications using declarative YAML configuration. Use when initializing tracing, metrics, or logging in a Java service, adding OpenTelemetry to a Java project, choosing between Javaagent/Spring Boot/manual setup, or configuring OTel providers in Java. Triggers on "setup otel in java", "java telemetry", "java tracing setup", "javaagent otel", "Spring Boot observability", "TracerProvider java", or when working on a Java project that needs observability.
---

# Java SDK Setup with Declarative Configuration

Set up OpenTelemetry in Java using declarative YAML configuration as the preferred approach.
The Javaagent provides automatic instrumentation; declarative config controls how that
telemetry is processed and exported.

For the YAML configuration schema, read the `general/declarative-config` skill.

## Setup Decision Tree

```
Is zero-code instrumentation sufficient?
├── Yes → Javaagent with declarative config (recommended)
│         -javaagent:opentelemetry-javaagent.jar -Dotel.config.file=otel.yaml
└── No  → Manual SDK setup
          ├── Spring Boot? → Spring Boot Starter
          └── Plain Java?  → Autoconfigure SDK extension
```

All paths support declarative configuration via `-Dotel.config.file`.

## Path A: Javaagent + Declarative Config (Recommended)

The Javaagent automatically instruments HTTP, gRPC, DB, and messaging frameworks.
Declarative config replaces the long list of `-Dotel.*` system properties.

### Dependencies

Download the Javaagent JAR (no compile dependencies needed for auto-instrumentation):

```bash
curl -L -o opentelemetry-javaagent.jar \
  https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
```

For manual instrumentation on top of the agent, add the API:

```xml
<dependencyManagement>
    <dependencies>
        <dependency>
            <groupId>io.opentelemetry</groupId>
            <artifactId>opentelemetry-bom</artifactId>
            <version>1.60.1</version>
            <type>pom</type>
            <scope>import</scope>
        </dependency>
    </dependencies>
</dependencyManagement>

<dependencies>
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-api</artifactId>
    </dependency>
</dependencies>
```

### Running with Declarative Config

```bash
java -javaagent:opentelemetry-javaagent.jar \
     -Dotel.config.file=configs/otel.yaml \
     -jar myservice.jar
```

Requires Javaagent version 2.26.0+. When `-Dotel.config.file` is set, all other
`-Dotel.*` properties are ignored (except agent-only properties like
`otel.javaagent.extensions` and `otel.javaagent.enabled`).

### YAML Config (file_format "1.0")

```yaml
file_format: "1.0"
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-myservice}"
    - name: deployment.environment.name
      value: "${DEPLOY_ENV:-development}"

propagator:
  composite: [tracecontext, baggage]

tracer_provider:
  sampler:
    parent_based:
      root:
        trace_id_ratio_based:
          ratio: ${SAMPLE_RATE:-1.0}
  processors:
    - batch:
        exporter:
          otlp:
            protocol: grpc
            endpoint: "${OTEL_ENDPOINT:-http://localhost:4317}"
            headers:
              api-key: "${API_KEY}"
            compression: gzip

meter_provider:
  readers:
    - periodic:
        interval: 60000
        exporter:
          otlp:
            protocol: grpc
            endpoint: "${OTEL_ENDPOINT:-http://localhost:4317}"

logger_provider:
  processors:
    - batch:
        exporter:
          otlp:
            protocol: grpc
            endpoint: "${OTEL_ENDPOINT:-http://localhost:4317}"
```

**Duration values must be in milliseconds** (e.g., `5000` not `5s`).

### Adding Manual Instrumentation

The Javaagent registers `GlobalOpenTelemetry` automatically. Use it to get tracers/meters:

```java
import io.opentelemetry.api.GlobalOpenTelemetry;
import io.opentelemetry.api.trace.Tracer;
import io.opentelemetry.api.metrics.Meter;

Tracer tracer = GlobalOpenTelemetry.getTracer("mycompany.com/myservice");
Meter meter = GlobalOpenTelemetry.getMeter("mycompany.com/myservice");
```

## Path B: Spring Boot Starter

For Spring Boot applications without the Javaagent:

```xml
<dependency>
    <groupId>io.opentelemetry.instrumentation</groupId>
    <artifactId>opentelemetry-spring-boot-starter</artifactId>
</dependency>
```

Configure via `application.properties` or use declarative config with:

```properties
otel.config.file=configs/otel.yaml
```

## Path C: Manual Autoconfigure

For non-Spring applications without the Javaagent:

```xml
<dependencies>
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-api</artifactId>
    </dependency>
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-sdk-extension-autoconfigure</artifactId>
        <scope>runtime</scope>
    </dependency>
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-exporter-otlp</artifactId>
        <scope>runtime</scope>
    </dependency>
</dependencies>
```

```java
import io.opentelemetry.sdk.autoconfigure.AutoConfiguredOpenTelemetrySdk;

// Reads -Dotel.config.file or OTEL_CONFIG_FILE, falls back to env vars
AutoConfiguredOpenTelemetrySdk sdk = AutoConfiguredOpenTelemetrySdk.builder().build();
```

Always use the BOM to align dependency versions.

## Key Details

- **BOM alignment**: Always import `opentelemetry-bom` to prevent version conflicts
- **API at compile, SDK at runtime**: Depend on `opentelemetry-api` at compile scope, SDK/exporter at runtime. This keeps application code decoupled from SDK internals.
- **Shutdown hook**: The Javaagent and autoconfigure both register a JVM shutdown hook automatically
- **Agent-only properties**: `otel.javaagent.extensions`, `otel.javaagent.enabled`, `otel.javaagent.debug` cannot be set via declarative config — they must remain as system properties
