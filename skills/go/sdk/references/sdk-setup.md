# OpenTelemetry Go SDK Setup

## Dependencies

```go
import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/log/global"
    "go.opentelemetry.io/otel/propagation"
    semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
    otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
    "go.opentelemetry.io/contrib/bridges/otelzap"
)
```

> **Migration note (contrib v1.35.0):** The `go.opentelemetry.io/contrib/config` module is deprecated. Use `go.opentelemetry.io/contrib/otelconf` instead. The API is identical — only the import path changes.

## Provider Setup

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

// Providers holds the OpenTelemetry providers and logger
type Providers struct {
    TracerProvider trace.TracerProvider
    MeterProvider  metric.MeterProvider
    LoggerProvider log.LoggerProvider
    Logger         *zap.Logger
    Closer         func(ctx context.Context) error
}

// SetupTelemetry initializes OpenTelemetry providers from configuration
func SetupTelemetry(ctx context.Context, serviceName, version, configFile string) (*Providers, error) {
    providers, err := providersFromConfig(ctx, serviceName, version, configFile)
    if err != nil {
        return nil, err
    }

    // Set global providers
    otel.SetTracerProvider(providers.TracerProvider)
    otel.SetMeterProvider(providers.MeterProvider)
    global.SetLoggerProvider(providers.LoggerProvider)
    
    // Set up context propagation, needed until this is fixed: https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return providers, nil
}

// providersFromConfig creates providers from YAML configuration file
func providersFromConfig(ctx context.Context, scope, version, cfgFile string) (*Providers, error) {
    b, err := os.ReadFile(cfgFile)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            // Return default providers if config doesn't exist
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

    // Expand environment variables in config
    b = []byte(os.ExpandEnv(string(b)))

    // Parse OpenTelemetry configuration
    conf, err := otelconf.ParseYAML(b)
    if err != nil {
        return nil, err
    }

    // Set resource attributes
    if conf.Resource == nil {
        conf.Resource = &otelconf.Resource{}
    }
    if conf.Resource.Attributes == nil {
        conf.Resource.Attributes = []otelconf.AttributeNameValue{}
    }

    // Add service metadata
    conf.Resource.Attributes = insertAttribute(conf.Resource.Attributes, 
        string(semconv.ServiceVersionKey), version)
    conf.Resource.Attributes = insertAttribute(conf.Resource.Attributes, 
        string(semconv.ServiceInstanceIDKey), uuid.New().String())

    // Create SDK
    sdk, err := otelconf.NewSDK(
        otelconf.WithContext(ctx), 
        otelconf.WithOpenTelemetryConfiguration(*conf),
    )
    if err != nil {
        return nil, err
    }

    // Create zap logger with OpenTelemetry bridge
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

## Global Provider Registration

After creating the providers, register them globally so instrumentation libraries and application code can access them via the OpenTelemetry API:

```go
otel.SetTracerProvider(providers.TracerProvider)
otel.SetMeterProvider(providers.MeterProvider)
global.SetLoggerProvider(providers.LoggerProvider)
```

Set up context propagation for distributed tracing:

```go
otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
    propagation.TraceContext{},
    propagation.Baggage{},
))
```

> **Note:** Explicit propagator setup is needed as a workaround until [contrib issue #6712](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712) is resolved. The declarative configuration should handle this automatically once that issue is fixed.

## Service Integration

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "myservice/internal/telemetry"
)

func main() {
    ctx := context.Background()

    // Setup telemetry
    providers, err := telemetry.SetupTelemetry(ctx, 
        telemetry.ServiceName, 
        telemetry.ServiceVersion,
        "configs/otel.yaml")
    if err != nil {
        log.Fatalf("Failed to setup telemetry: %v", err)
    }

    // Graceful shutdown
    defer func() {
        shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
        defer cancel()
        if err := providers.Closer(shutdownCtx); err != nil {
            providers.Logger.Error("Failed to shutdown telemetry", zap.Error(err))
        }
    }()

    // Start your application
    app := NewApp(providers.Logger)
    if err := app.Run(ctx); err != nil {
        providers.Logger.Fatal("Application failed", zap.Error(err))
    }
}

type App struct {
    logger *zap.Logger
    tracer trace.Tracer
    meter  metric.Meter
}

func NewApp(logger *zap.Logger) *App {
    return &App{
        logger: logger,
        tracer: otel.Tracer(telemetry.Scope),
        meter:  otel.Meter(telemetry.Scope),
    }
}

func (a *App) Run(ctx context.Context) error {
    ctx, span := a.tracer.Start(ctx, "app startup")

    a.logger.Info("Application starting")
    
    // Application logic here
    
    span.End() // Close span before blocking
    
    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    a.logger.Info("Application shutting down")
    return nil
}
```

## Configuration File

Example OpenTelemetry configuration file (`configs/otel.yaml`):

```yaml
file_format: "0.3"
resource:
  schema_url: https://opentelemetry.io/schemas/1.26.0
  attributes:
    - name: service.name
      value: "user-service"
    - name: deployment.environment.name
      value: "development"

propagator:
  composite: [ tracecontext, baggage ]

tracer_provider:
  processors:
    - batch:
        timeout: 1s
        send_batch_size: 1024
        exporter:
          otlp:
            protocol: http/protobuf
            endpoint: "http://localhost:4318"

meter_provider:
  readers:
    - periodic:
        interval: 30s
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

## Declarative Configuration with otelconf

The `otelconf` package (`go.opentelemetry.io/contrib/otelconf`) implements the language-agnostic OpenTelemetry declarative configuration schema in Go. It parses a YAML configuration file conforming to the OpenTelemetry configuration schema and creates fully configured SDK providers.

The full YAML schema is covered by the `opentelemetry-sdk-configuration` skill.

Key functions:

```go
// Parse a YAML configuration file
conf, err := otelconf.ParseYAML(yamlBytes)

// Create an SDK instance from the parsed configuration
sdk, err := otelconf.NewSDK(
    otelconf.WithContext(ctx),
    otelconf.WithOpenTelemetryConfiguration(*conf),
)

// Access providers from the SDK
tracerProvider := sdk.TracerProvider()
meterProvider := sdk.MeterProvider()
loggerProvider := sdk.LoggerProvider()

// Shut down all providers, flushing pending telemetry
err = sdk.Shutdown(ctx)
```
