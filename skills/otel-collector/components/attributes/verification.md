# `attributes`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`attributes` ships in the `core`, `contrib`, and `k8s` distributions.

Config (`attributes-verify.yaml`) — an `upsert` action that sets (or overwrites) an `env` attribute on every span:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  attributes:
    actions:
      - key: env
        value: prod
        action: upsert
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [attributes]
      exporters: [debug]
```

Send a known number of traces, each carrying an `env` attribute with a *different* value so the `upsert` overwrite is observable. Use `--workers 1` so the `--traces` count (which is **per worker**) is exact:

```bash
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 \
  --workers 1 --traces 5 --telemetry-attributes 'env="staging"'
```

Flags used, all confirmed against the `otel-telemetrygen` skill (`references/flags.md`):

- `--otlp-insecure` — disable transport security (common flag).
- `--otlp-endpoint` — destination, defaults to `localhost:4317` for gRPC (common flag).
- `--workers` — concurrent workers; `1` keeps the count exact (common flag).
- `--traces` — traces **per worker** (trace-specific flag).
- `--telemetry-attributes` — span-level attribute as `key="value"` (must be quoted). Resource-level `--otlp-attributes` would attach to the Resource, which the `attributes` processor does **not** touch.

**What proves it worked:** in the `debug` exporter output, every span shows `env: Str(prod)` in its span attributes — the value sent (`staging`) has been overwritten by the `upsert` action. To prove the **insert** half of `upsert`, drop `--telemetry-attributes` and send plain traces; `env: Str(prod)` still appears because the key was absent and got inserted. (Use `insert` instead of `upsert` to confirm it leaves a pre-existing `env` untouched, or `update` to confirm it only writes when the key already exists.)

You can run the same recipe with `telemetrygen logs ... --logs 5` and a `logs` pipeline to confirm log-record attributes; `--logs` is likewise per-worker, so keep `--workers 1`.
