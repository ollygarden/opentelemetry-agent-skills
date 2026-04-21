# OpenTelemetry Go Instrumentation Libraries

## Detecting Existing Instrumentation

Before adding manual instrumentation, check for these imports and middleware patterns in the codebase:

```go
import (
    // HTTP frameworks
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
    "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

    // gRPC
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

    // Logging bridges
    "go.opentelemetry.io/contrib/bridges/otelzap"
    "go.opentelemetry.io/contrib/bridges/otelslog"
    "go.opentelemetry.io/contrib/bridges/otellogrus"
    "go.opentelemetry.io/contrib/bridges/otelzerolog"
)
```

If these middleware/wrapper patterns are present, spans are already being created:

```go
handler := otelhttp.NewHandler(mux, "service-name")       // net/http
router.Use(otelgin.Middleware("service-name"))             // gin (also emits HTTP metrics)
router.Use(otelmux.Middleware("service-name"))             // gorilla/mux (also emits HTTP metrics)
```

## Library Catalog

### HTTP

| Library | Package |
|---------|---------|
| net/http | `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` |
| Gin | `go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin` |
| Gorilla mux | `go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux` |
| Echo | `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho` |
| go-restful | `go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful` |

```go
// HTTP server with automatic instrumentation
func setupHTTPServer() *http.Server {
    mux := http.NewServeMux()
    mux.HandleFunc("/users", handleUsers)

    // Automatic span creation and context propagation
    handler := otelhttp.NewHandler(mux, "user-service")

    return &http.Server{
        Addr:    ":8080",
        Handler: handler,
    }
}

// HTTP client with automatic instrumentation
// Note: otelhttp.DefaultClient, Get, Head, Post, PostForm were removed in contrib v1.40.0
// Always create a custom client with otelhttp.NewTransport instead
func setupHTTPClient() *http.Client {
    return &http.Client{
        Transport: otelhttp.NewTransport(http.DefaultTransport),
    }
}
```

### gRPC

| Library | Package |
|---------|---------|
| gRPC | `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` |

```go
// gRPC server with automatic instrumentation (prefer stats handlers over interceptors)
func setupGRPCServer() *grpc.Server {
    return grpc.NewServer(
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
    )
}

// gRPC client with automatic instrumentation
func setupGRPCClient(target string) (*grpc.ClientConn, error) {
    return grpc.NewClient(target,
        grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
    )
}

// Override default span kind in gRPC (contrib v1.41.0+)
func setupGRPCServerWithSpanKind() *grpc.Server {
    return grpc.NewServer(
        grpc.StatsHandler(otelgrpc.NewServerHandler(
            otelgrpc.WithSpanKind(trace.SpanKindInternal),
        )),
    )
}
```

### Database

| Library | Package |
|---------|---------|
| database/sql | `go.opentelemetry.io/contrib/instrumentation/database/sql/otelsql` |
| MongoDB | `go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo` |

### AWS

| Library | Package |
|---------|---------|
| AWS SDK v2 | `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws` |
| Lambda | `go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda` |

### Logging Bridges

| Logger | Package |
|--------|---------|
| zap | `go.opentelemetry.io/contrib/bridges/otelzap` |
| slog | `go.opentelemetry.io/contrib/bridges/otelslog` |
| logrus | `go.opentelemetry.io/contrib/bridges/otellogrus` |
| zerolog | `go.opentelemetry.io/contrib/bridges/otelzerolog` |
| logr | `go.opentelemetry.io/contrib/bridges/otellogr` |

> **Logging bridge change (contrib v1.35.0):** otelzap and otelslog now emit `code.function` with the full package path-qualified function name (e.g., `github.com/user/pkg.MyFunc`) instead of just the function name. The `code.namespace` attribute is no longer emitted.

```go
// zap bridge setup
import (
    "go.opentelemetry.io/contrib/bridges/otelzap"
    "go.opentelemetry.io/otel/log/global"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func setupLogger() *zap.Logger {
    // Create a tee core that logs to both stdout and OpenTelemetry
    core := zapcore.NewTee(
        zapcore.NewCore(
            zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
            zapcore.AddSync(os.Stdout),
            zapcore.InfoLevel,
        ),
        otelzap.NewCore("myservice",
            otelzap.WithLoggerProvider(global.GetLoggerProvider())),
    )
    return zap.New(core)
}
```

```go
// slog bridge setup (Go 1.21+)
import (
    "log/slog"
    "go.opentelemetry.io/contrib/bridges/otelslog"
    "go.opentelemetry.io/otel/log/global"
)

func setupLogger() *slog.Logger {
    handler := otelslog.NewHandler("myservice",
        otelslog.WithLoggerProvider(global.GetLoggerProvider()))
    return slog.New(handler)
}
```

```go
// logrus bridge setup
import (
    "github.com/sirupsen/logrus"
    "go.opentelemetry.io/contrib/bridges/otellogrus"
    "go.opentelemetry.io/otel/log/global"
)

func setupLogger() {
    logrus.AddHook(otellogrus.NewHook("myservice",
        otellogrus.WithLoggerProvider(global.GetLoggerProvider())))
}
```

### Resource Detectors

| Cloud | Package |
|-------|---------|
| AWS EC2 | `go.opentelemetry.io/contrib/detectors/aws/ec2` |
| AWS ECS | `go.opentelemetry.io/contrib/detectors/aws/ecs` |
| AWS EKS | `go.opentelemetry.io/contrib/detectors/aws/eks` |
| AWS Lambda | `go.opentelemetry.io/contrib/detectors/aws/lambda` |
| GCP | `go.opentelemetry.io/contrib/detectors/gcp` |
| Azure VM | `go.opentelemetry.io/contrib/detectors/azure/azurevm` |

### Propagators

| Propagator | Package |
|------------|---------|
| Environment carrier | `go.opentelemetry.io/contrib/propagators/envcar` (new in v1.42.0) |

## Manual Instrumentation Patterns

Use these patterns when no contrib instrumentation library exists for the target.

### HTTP Client with Semconv Attributes

```go
func (c *CustomClient) CallExternalAPI(ctx context.Context, endpoint string) error {
    ctx, span := c.tracer.Start(ctx, "GET",
        trace.WithSpanKind(trace.SpanKindClient),
        trace.WithAttributes(
            semconv.HTTPRequestMethodGet,
            semconv.URLFull(endpoint)))
    defer span.End()

    resp, err := c.httpClient.Get(endpoint)
    if err != nil {
        return fmt.Errorf("making request: %w", err)
    }
    defer resp.Body.Close()

    span.SetAttributes(semconv.HTTPResponseStatusCode(resp.StatusCode))
    return nil
}
```

### Database Operation with Semconv

```go
func (r *Repository) CreateUser(ctx context.Context, userData *UserData) (*User, error) {
    ctx, span := r.tracer.Start(ctx, "INSERT users",
        trace.WithAttributes(
            semconv.DBSystem("postgresql"),
            semconv.DBNamespace(r.dbName),
            semconv.DBOperation("INSERT"),
            semconv.DBSQLTable("users")))
    defer span.End()

    query := `INSERT INTO users (email, name, tier) VALUES ($1, $2, $3) RETURNING id, created_at`

    var user User
    err := r.db.QueryRow(ctx, query, userData.Email, userData.Name, userData.Tier).
        Scan(&user.ID, &user.CreatedAt)

    if err != nil {
        span.SetAttributes(attribute.String("error.type", classifyDBError(err)))
        return nil, fmt.Errorf("inserting user: %w", err)
    }

    span.SetAttributes(
        attribute.String("user.id", user.ID),
        attribute.String("user.tier", userData.Tier),
    )

    return &user, nil
}

func classifyDBError(err error) string {
    if strings.Contains(err.Error(), "unique constraint") {
        return "duplicate_key"
    }
    if strings.Contains(err.Error(), "connection") {
        return "connection_error"
    }
    return "unknown"
}
```

### Message Queue Producer/Consumer with Semconv

```go
func (p *Publisher) PublishOrderEvent(ctx context.Context, order *Order) error {
    ctx, span := p.tracer.Start(ctx, "send queue.orders",
        trace.WithSpanKind(trace.SpanKindProducer),
        trace.WithAttributes(
            semconv.MessagingSystem("nats"),
            semconv.MessagingOperationTypePublish,
            semconv.MessagingDestinationName("orders"),
            attribute.String("order.id", order.ID)))
    defer span.End()

    message, err := json.Marshal(order)
    if err != nil {
        return fmt.Errorf("marshaling order: %w", err)
    }

    if err := p.client.Publish("orders", message); err != nil {
        return fmt.Errorf("publishing message: %w", err)
    }

    span.SetAttributes(attribute.Int("message.size", len(message)))

    return nil
}

func (c *Consumer) HandleOrderEvent(ctx context.Context, msg []byte) error {
    ctx, span := c.tracer.Start(ctx, "consume queue.orders",
        trace.WithSpanKind(trace.SpanKindConsumer),
        trace.WithAttributes(
            semconv.MessagingSystem("nats"),
            semconv.MessagingOperationTypeReceive,
            semconv.MessagingDestinationName("orders")))
    defer span.End()

    var order Order
    if err := json.Unmarshal(msg, &order); err != nil {
        span.SetStatus(codes.Error, "message parsing failed")
        return fmt.Errorf("unmarshaling order: %w", err)
    }

    span.SetAttributes(attribute.String("order.id", order.ID))

    if err := c.processOrder(ctx, &order); err != nil {
        return fmt.Errorf("processing order: %w", err)
    }

    return nil
}
```

### Background Job

```go
func (w *Worker) ProcessBatch(ctx context.Context, batchID string) error {
    ctx, span := w.tracer.Start(ctx, "process batch",
        trace.WithAttributes(
            attribute.String("batch.id", batchID),
            attribute.String("worker.id", w.id)))
    defer span.End()

    items, err := w.repo.GetBatchItems(ctx, batchID)
    if err != nil {
        return fmt.Errorf("fetching batch items: %w", err)
    }

    span.SetAttributes(attribute.Int("batch.size", len(items)))

    processed := 0
    for _, item := range items {
        if err := w.processItem(ctx, item); err != nil {
            var rec log.Record
            rec.SetTimestamp(time.Now())
            rec.SetSeverity(log.SeverityWarn)
            rec.SetBody(log.StringValue("item processing failed"))
            rec.AddAttributes(
                log.String("item.id", item.ID),
                log.String("error", err.Error()),
            )
            w.logger.Emit(ctx, rec)
            continue
        }
        processed++
    }

    span.SetAttributes(
        attribute.Int("batch.processed", processed),
        attribute.Int("batch.failed", len(items)-processed),
    )

    return nil
}
```

## Enriching Spans from Instrumentation Libraries

When an instrumentation library (otelhttp, otelgin, otelmux, etc.) already creates spans, use `trace.SpanFromContext(ctx)` to add attributes to the existing span. Do not create a new span.

```go
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    // Span already created by instrumentation library (otelhttp, otelgin, etc.)
    ctx := r.Context()

    // Add business attributes to existing span
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("business.operation", "order_creation"),
        attribute.String("customer.tier", extractCustomerTier(r)))

    order, err := h.service.CreateOrder(ctx, extractOrderData(r))
    if err != nil {
        span.SetStatus(codes.Error, "order creation failed")
        http.Error(w, "Failed to create order", http.StatusInternalServerError)
        return
    }

    span.SetAttributes(attribute.String("order.id", order.ID))

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(order)
}
```
