---
name: go-sdk-setup-declarative-config
description: Set up OpenTelemetry SDK in Go applications using declarative YAML configuration (otelconf). Use when initializing tracing, metrics, or logging in a Go service, adding OpenTelemetry to a Go project, or setting up OTel providers in Go. Triggers on "setup otel in go", "go telemetry setup", "go tracing setup", "otelconf go", "TracerProvider go", "MeterProvider go", or when working on a Go project that needs observability.
---

# Go SDK Setup with otelconf

Set up OpenTelemetry in Go using `otelconf` — the declarative YAML configuration package.
This is the recommended approach over programmatic SDK construction or scattered env vars.

For the YAML configuration schema, read the `general/declarative-config` skill.

## Dependencies

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/log/global"
    "go.opentelemetry.io/otel/propagation"
    semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
    otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
    "go.opentelemetry.io/contrib/bridges/otelzap"
)
```

> **Migration note (contrib v1.35.0):** `go.opentelemetry.io/contrib/config` is deprecated.
> Use `go.opentelemetry.io/contrib/otelconf` instead. API is identical, only the import path changes.

## Project Structure

```
internal/telemetry/
├── const.go          # Service scope and telemetry constants
├── setup.go          # SDK initialization (code below)
├── providers.go      # Provider management utilities
└── carriers.go       # Custom propagation carriers (if needed)
configs/
└── otel.yaml         # Declarative configuration
```

## Setup Pattern

The core setup reads a YAML config file, injects runtime attributes, and creates an SDK
instance that provides all three providers (tracer, meter, logger).

```go
package telemetry

import (
    "context"
    "errors"
    "fmt"
    "os"

    "github.com/google/uuid"
    "go.opentelemetry.io/contrib/bridges/otelzap"
    otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/log"
    "go.opentelemetry.io/otel/log/global"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/propagation"
    semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
    "go.opentelemetry.io/otel/trace"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type Providers struct {
    TracerProvider trace.TracerProvider
    MeterProvider  metric.MeterProvider
    LoggerProvider log.LoggerProvider
    Logger         *zap.Logger
    Closer         func(ctx context.Context) error
}

func SetupTelemetry(ctx context.Context, serviceName, version, configFile string) (*Providers, error) {
    providers, err := providersFromConfig(ctx, serviceName, version, configFile)
    if err != nil {
        return nil, err
    }

    // Set global providers
    otel.SetTracerProvider(providers.TracerProvider)
    otel.SetMeterProvider(providers.MeterProvider)
    global.SetLoggerProvider(providers.LoggerProvider)

    // Set propagation — needed until https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712 is fixed
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return providers, nil
}

func providersFromConfig(ctx context.Context, scope, version, cfgFile string) (*Providers, error) {
    b, err := os.ReadFile(cfgFile)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            logger := zap.Must(zap.NewProduction())
            logger.Warn("OpenTelemetry config file not found, using no-op providers",
                zap.String("config_file", cfgFile))
            return &Providers{
                TracerProvider: trace.NewNoOpTracerProvider(),
                MeterProvider:  metric.NewNoOpMeterProvider(),
                LoggerProvider: log.NewNoOpLoggerProvider(),
                Logger:         logger,
                Closer:         func(ctx context.Context) error { return nil },
            }, nil
        }
        return nil, fmt.Errorf("failed to read config file %s: %w", cfgFile, err)
    }

    b = []byte(os.ExpandEnv(string(b)))

    conf, err := otelconf.ParseYAML(b)
    if err != nil {
        return nil, err
    }

    // Inject runtime resource attributes
    if conf.Resource == nil {
        conf.Resource = &otelconf.Resource{}
    }
    if conf.Resource.Attributes == nil {
        conf.Resource.Attributes = []otelconf.AttributeNameValue{}
    }
    conf.Resource.Attributes = insertAttribute(conf.Resource.Attributes,
        string(semconv.ServiceVersionKey), version)
    conf.Resource.Attributes = insertAttribute(conf.Resource.Attributes,
        string(semconv.ServiceInstanceIDKey), uuid.New().String())

    sdk, err := otelconf.NewSDK(
        otelconf.WithContext(ctx),
        otelconf.WithOpenTelemetryConfiguration(*conf),
    )
    if err != nil {
        return nil, err
    }

    // Zap logger with OTel bridge — logs go to both stdout and OTel
    core := zapcore.NewTee(
        zapcore.NewCore(
            zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
            zapcore.AddSync(os.Stdout),
            zapcore.InfoLevel,
        ),
        otelzap.NewCore(scope, otelzap.WithLoggerProvider(global.GetLoggerProvider())),
    )

    return &Providers{
        TracerProvider: sdk.TracerProvider(),
        MeterProvider:  sdk.MeterProvider(),
        LoggerProvider: sdk.LoggerProvider(),
        Logger:         zap.New(core),
        Closer:         sdk.Shutdown,
    }, nil
}

func insertAttribute(attrs []otelconf.AttributeNameValue, name, value string) []otelconf.AttributeNameValue {
    for _, attr := range attrs {
        if attr.Name == name {
            return attrs
        }
    }
    return append(attrs, otelconf.AttributeNameValue{Name: name, Value: value})
}
```

## Main Integration

```go
func main() {
    ctx := context.Background()

    providers, err := telemetry.SetupTelemetry(ctx,
        telemetry.ServiceName,
        telemetry.ServiceVersion,
        "configs/otel.yaml")
    if err != nil {
        log.Fatalf("Failed to setup telemetry: %v", err)
    }

    defer func() {
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        if err := providers.Closer(shutdownCtx); err != nil {
            providers.Logger.Error("Failed to shutdown telemetry", zap.Error(err))
        }
    }()

    // Get tracer/meter from global providers
    tracer := otel.Tracer(telemetry.Scope)
    meter := otel.Meter(telemetry.Scope)

    // Application logic...
}
```

## YAML Config (file_format "0.3")

Go's otelconf v0.3.0 uses file_format `"0.3"`:

```yaml
file_format: "0.3"
resource:
  attributes:
    - name: service.name
      value: "my-service"
    - name: deployment.environment.name
      value: "development"

propagator:
  composite: [tracecontext, baggage]

tracer_provider:
  processors:
    - batch:
        exporter:
          otlp:
            protocol: http/protobuf
            endpoint: "http://localhost:4318"

meter_provider:
  readers:
    - periodic:
        interval: 30000
        exporter:
          otlp:
            protocol: http/protobuf
            endpoint: "http://localhost:4318"

logger_provider:
  processors:
    - batch:
        exporter:
          otlp:
            protocol: http/protobuf
            endpoint: "http://localhost:4318"
```

## Key Details

- **No-op fallback**: If the config file doesn't exist, the setup returns no-op providers instead of failing. The application runs without telemetry.
- **Propagator workaround**: `otel.SetTextMapPropagator()` must be called manually due to [open-telemetry/opentelemetry-go-contrib#6712](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712). The YAML `propagator` section alone is not sufficient.
- **Runtime attributes**: `service.version` and `service.instance.id` are injected programmatically because they vary per deployment, not per environment.
- **Zap bridge**: The `otelzap` bridge sends structured logs to the OTel LoggerProvider, enabling log correlation with traces.
