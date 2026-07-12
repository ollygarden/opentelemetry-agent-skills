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
)
```

If these middleware/wrapper patterns are present, spans are already being created:

```go
handler := otelhttp.NewHandler(mux, "service-name")       // net/http
router.Use(otelgin.Middleware("service-name"))             // gin (also emits HTTP metrics)
router.Use(otelmux.Middleware("service-name"))             // gorilla/mux (also emits HTTP metrics)
```

## Context Propagation (read first)

In Go, spans link to their parent **only** through `context.Context`. An instrumentation library
installs correctly and still emits *detached* spans if the active context never reaches the call. A
SERVER span living in `c.Request.Context()` (gin) or `r.Context()` (net/http) parents a downstream
DB/client span **only if that exact context is passed into it**.

> **Symptom:** DB or client spans show up as CLIENT-kind *roots* in their own traces, disconnected
> from the request. OllyGarden flags this as "Root Client Span"; it means trace context was dropped
> at an internal boundary.

The usual cause is structural, not a missing option:

```go
// ANTI-PATTERN: global handle + ctx-less call → span parented to context.Background()
db := common.GetDB()             // package-level *gorm.DB
db.Where("id = ?", id).First(&u) // no ctx → detached CLIENT-root span

// CORRECT: thread the request context from the handler down to the call
func (h *Handler) GetUser(c *gin.Context) {
    u, err := h.users.Find(c.Request.Context(), id) // ctx flows in
}
func (r *UserRepo) Find(ctx context.Context, id int) (*User, error) {
    var u User
    return &u, r.db.WithContext(ctx).First(&u, id).Error // GORM links via WithContext(ctx)
}
```

A data layer built on a global `*gorm.DB` (or `*sql.DB`) plus functions that don't accept `ctx`
**cannot** produce connected traces — the call signatures have to carry `ctx`. Rules:

- Every DB/client/outbound call runs with the incoming request's context.
- For GORM, call `db.WithContext(ctx)` on every query; for `database/sql`, use the `...Context`
  methods (`QueryContext`, `ExecContext`).
- Never call instrumented clients with `context.Background()` or `context.TODO()` on a request path.

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
| database/sql | No package in current `go.opentelemetry.io/contrib`; check the OpenTelemetry registry for third-party instrumentation or instrument manually with `...Context` methods. |
| MongoDB | `go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo` |
| GORM | `gorm.io/plugin/opentelemetry/tracing` (registered via `db.Use(tracing.NewPlugin(...))`) |

> **PII in `db.query.text`:** the GORM plugin records the executed statement with bound parameter
> values **inlined** by default, so a query like `INSERT INTO users (email, ...) VALUES ('a@b.com', ...)`
> leaks the literal email onto the span. Disable query-variable capture so the statement is
> parameterized:
>
> ```go
> db.Use(tracing.NewPlugin(tracing.WithoutQueryVariables()))
> // db.query.text becomes: INSERT INTO "users" (...) VALUES (?, ?, ?)
> ```
>
> Treat this as the default for any service that touches user data. The same trap applies to other
> SQL instrumentation that captures full statement text; verify the selected package's option to
> disable or redact query text. Keep parameter values out of `db.query.text` before export. If the
> selected instrumentation cannot sanitize or parameterize query text, disable query-text capture;
> downstream redaction is defense-in-depth.

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
    // semconv helpers below track go.opentelemetry.io/otel/semconv/v1.41.0
    ctx, span := r.tracer.Start(ctx, "INSERT users",
        trace.WithAttributes(
            semconv.DBSystemNamePostgreSQL,        // db.system.name (was DBSystem/db.system)
            semconv.DBNamespace(r.dbName),
            semconv.DBOperationName("INSERT"),      // db.operation.name (was DBOperation)
            semconv.DBCollectionName("users")))     // db.collection.name (was DBSQLTable/db.sql.table)
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
            semconv.MessagingSystemKey.String("nats"), // messaging.system (no NATS enum const)
            semconv.MessagingOperationTypeSend,        // was MessagingOperationTypePublish
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
            semconv.MessagingSystemKey.String("nats"), // messaging.system (no NATS enum const)
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
