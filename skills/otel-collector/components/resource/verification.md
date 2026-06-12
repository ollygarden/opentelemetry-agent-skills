# `resource`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`resource` ships in the `core`, `contrib`, and `k8s` distributions.

Config (`resource-verify.yaml`) — an `upsert` action that sets (or overwrites) the `deployment.environment` **resource** attribute:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  resource:
    attributes:
      - key: deployment.environment
        value: prod
        action: upsert
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resource]
      exporters: [debug]
```

Send a known number of traces whose **resource** carries `deployment.environment` with a *different* value, so the `upsert` overwrite is observable. telemetrygen sets resource attributes with `--otlp-attributes` (telemetry-level attributes use the separate `--telemetry-attributes` flag). Use `--workers 1` so the `--traces` count (which is **per worker**) is exact:

```bash
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 \
  --workers 1 --traces 5 --otlp-attributes 'deployment.environment="staging"'
```

Flags used, all confirmed against the `otel-telemetrygen` skill (`references/flags.md`):

- `--otlp-insecure` — disable transport security (common flag).
- `--otlp-endpoint` — destination, defaults to `localhost:4317` for gRPC (common flag).
- `--workers` — concurrent workers; `1` keeps the count exact (common flag).
- `--traces` — traces **per worker** (trace-specific flag).
- `--otlp-attributes` — **resource**-level attribute as `key="value"` (string values must be quoted; repeatable). This is the resource counterpart of `--telemetry-attributes`, which sets span/log/datapoint attributes that the `resource` processor does **not** touch.

**What proves it worked:** in the `debug` exporter output, the **Resource attributes** block (not the span attributes) shows `deployment.environment: Str(prod)` — the value sent (`staging`) has been overwritten by the `upsert` action. The attribute appears once per resource, shared across all spans in the batch.

To prove the **insert** half of `upsert`, drop `--otlp-attributes` and send plain traces; `deployment.environment: Str(prod)` still appears in the Resource attributes because the key was absent and got inserted. (Use `insert` instead of `upsert` to confirm it leaves a pre-existing resource value untouched, or `update` to confirm it only writes when the key already exists.)

You can run the same recipe with `telemetrygen logs ... --logs 5` (or `metrics ... --metrics 5`) and a matching pipeline; `--logs`/`--metrics` are likewise per-worker, so keep `--workers 1`. `--otlp-attributes` sets the resource for every signal type.
