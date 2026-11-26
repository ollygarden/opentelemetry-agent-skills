# Context propagation into build and test processes

How a process started by a pipeline joins the pipeline's trace. Values hold for the named releases;
check the current release before relying on them.

- [The specification](#the-specification)
- [SDK carriers](#sdk-carriers)
- [Passing context between steps](#passing-context-between-steps)
- [Command-line spans](#command-line-spans)

## The specification

[Environment Variables as Context Propagation Carriers](https://github.com/open-telemetry/opentelemetry-specification/blob/v1.63.0/specification/context/env-carriers.md)
is **Stable** in specification v1.63.0 (Release Candidate since v1.59.0; see the
[CHANGELOG](https://github.com/open-telemetry/opentelemetry-specification/blob/v1.63.0/CHANGELOG.md)).
It names CI/CD environments as a use case.

- The configured propagators choose the keys; the carrier only normalizes them. With the default
  `OTEL_PROPAGATORS=tracecontext,baggage`, the variables are `TRACEPARENT`, `TRACESTATE` and
  `BAGGAGE`, holding the W3C header values.
- Only the normalized, uppercase name is read, so a lowercase `traceparent` variable is ignored
  (Windows lookups are case-insensitive); other propagators' keys are uppercased with every
  character other than letters, digits and `_` replaced by `_` (`x-b3-traceid` becomes
  `X_B3_TRACEID`).
- SDKs MUST NOT spawn child processes as part of propagation. The parent copies its environment,
  injects the context into the copy, and starts the child with it; the child extracts at startup.
  Use one copy per child, and do not rewrite the parent's own context variables.
- The specification calls environment variables not appropriate for sensitive information.

## SDK carriers

The carriers are APIs only: none of these SDKs reads `TRACEPARENT` on its own. Application code
calls the configured propagator's extract (or inject) with the carrier, then uses the returned
context as the parent. The
[compliance matrix](https://github.com/open-telemetry/opentelemetry-specification/blob/v1.63.0/spec-compliance-matrix.md)
tracks support per language.

| Language | Package and API | Status | First release |
|---|---|---|---|
| Go | `go.opentelemetry.io/contrib/propagators/envcar`, `&envcar.Carrier{}` (pointer receivers; set `SetEnvFunc` to inject) | experimental module; its go.mod requires Go 1.25 | v0.67.0 ([`otel-go`](../../otel-go/references/instrumentation-libraries.md#propagators)); core `go.opentelemetry.io/otel` has no env carrier |
| Java | `io.opentelemetry:opentelemetry-api-incubator`, `io.opentelemetry.api.incubator.propagation.EnvironmentGetter` / `EnvironmentSetter` | incubating (alpha) | [v1.60.0](https://github.com/open-telemetry/opentelemetry-java/releases/tag/v1.60.0); normalization changed in v1.61.0 |
| Python | `opentelemetry-api`, `opentelemetry.propagators._envcarrier.EnvironmentGetter` / `EnvironmentSetter` | private module | [v1.40.0](https://github.com/open-telemetry/opentelemetry-python/releases/tag/v1.40.0); `get` takes the environment mapping as its first argument from v1.44.0 ([`otel-python`](../../otel-python/references/breaking-changes.md)) |
| JavaScript | `@opentelemetry/propagator-env-carrier`, `EnvironmentGetter` / `EnvironmentSetter` | experimental (0.x) | [0.220.0](https://github.com/open-telemetry/opentelemetry-js/releases/tag/experimental/v0.220.0) |
| .NET | `OpenTelemetry.Api`, `OpenTelemetry.Context.Propagation.EnvironmentVariableCarrier` | stable | [core-1.21.0](https://github.com/open-telemetry/opentelemetry-dotnet/releases/tag/core-1.21.0) (experimental from 1.16.0-rc.1); extract/inject in [`otel-dotnet`](../../otel-dotnet/references/api.md#propagation) |
| C++ | `opentelemetry::context::propagation::EnvironmentCarrier` | API | [v1.26.0](https://github.com/open-telemetry/opentelemetry-cpp/releases/tag/v1.26.0) |
| Ruby | none at [`opentelemetry-api/v1.11.1`](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-api/v1.11.1/api/lib/opentelemetry/context/propagation) | n/a | n/a; extraction with a hash getter is in [`otel-ruby`](../../otel-ruby/references/api.md#propagation-and-fiber-local-context) |

Without a carrier (Ruby, an older SDK, or a Go module too old for `envcar`), extract from a map
built from the environment, for example Ruby's `{"traceparent" => ENV["TRACEPARENT"]}` with a hash
getter, or Go's
`propagation.MapCarrier{"traceparent": os.Getenv("TRACEPARENT"), "tracestate": os.Getenv("TRACESTATE")}`.

Go test example, child side (provider and propagator setup:
[`otel-go` propagation](../../otel-go/references/api.md#propagation); flushing:
[`ForceFlush`](../../otel-go/references/performance.md#forceflush)):

```go
var parentCtx context.Context // start test spans from this context

func TestMain(m *testing.M) {
	tp := newTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))
	parentCtx = otel.GetTextMapPropagator().Extract(context.Background(), &envcar.Carrier{})
	code := m.Run()
	if err := tp.ForceFlush(context.Background()); err != nil {
		code = 1 // spans were not exported
	}
	os.Exit(code)
}
```

`go test ./...` prints nothing from passing packages and replays cached results without running
`TestMain`, so use `-count=1` for traced runs. The exit code above catches a failed final flush;
a batch that fails to export during `m.Run()` is reported only to `otel.Handle`, so register
`otel.SetErrorHandler` to fail the run on those too.

## Passing context between steps

- **GitHub Actions:** each step is a new process. Pass `TRACEPARENT` through the `env:` of the step
  that needs it, which matches the specification's one copy per child; appending it to
  `$GITHUB_ENV` exposes it to every later step of the job. Jobs do not share environments, so pass
  values to other jobs through job outputs. To join the `github` receiver's trace instead,
  compute the value from its deterministic IDs
  ([github-receiver.md](github-receiver.md#joining-in-job-telemetry-to-the-receivers-trace)).
- **GitLab CI/CD:** each job's `script` runs in one shell, so `export TRACEPARENT=...` reaches
  later commands of that job; other jobs need artifacts or dotenv reports. The `gitlab` receiver's
  trace cannot be joined from inside a job ([gitlab-receiver.md](gitlab-receiver.md#limitations)).
- **CI systems that inject `TRACEPARENT`:** commands inherit the system's span; see the last column
  of [native-integrations.md](native-integrations.md).

## Command-line spans

To wrap a shell command in a span without writing SDK code, `otel-cli` reads `TRACEPARENT` from
the environment, exports with the standard `OTEL_EXPORTER_OTLP_*` variables, and passes the new
context to the wrapped command (`otel-cli exec --service ci --name "go test" -- go test ./...`;
`--tp-carrier` and `--tp-export` write the new `TRACEPARENT` for later steps). It is published as
[equinix-labs/otel-cli](https://github.com/equinix-labs/otel-cli/releases/tag/v0.4.5) (v0.4.5) and
the fork [tobert/otel-cli](https://github.com/tobert/otel-cli/releases/tag/v0.6.1) (v0.6.1).
Neither is an OpenTelemetry project, and neither emits the `cicd.*` conventions on its own. At
tobert/otel-cli v0.6.1:

- Pass `--kind` explicitly (`server` for a pipeline-run span, `internal` for tasks) and
  `--fail --verbose`, or export errors go unnoticed.
- `--force-trace-id`, `--force-span-id` and `--force-parent-span-id` set precomputed IDs.
- `--attrs` parses numeric-looking values as integers, then floats and booleans
  ([protobuf_span.go](https://github.com/tobert/otel-cli/blob/v0.6.1/otlpclient/protobuf_span.go)),
  so string span attributes such as `cicd.pipeline.task.run.id` arrive as integers. Converting
  them back takes a Collector in the path, with the `attributes` processor's `convert` action and
  `converted_type: string` ([`otel-collector`](../../otel-collector/components/attributes/configuration.md#action-fields));
  posting OTLP/JSON with explicit `stringValue` fields avoids the Collector.
- Resource attributes set through `OTEL_RESOURCE_ATTRIBUTES` stay strings.
- An environment value it cannot parse is echoed in the error (`could not parse %s value %q`,
  [config.go](https://github.com/tobert/otel-cli/blob/v0.6.1/otelcli/config.go)), so a malformed
  `OTEL_EXPORTER_OTLP_HEADERS` prints its token; rely on the CI system's secret masking.
