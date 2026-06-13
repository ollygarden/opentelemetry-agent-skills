# `prometheus` exporter: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

Unlike most exporters, you observe the result by **scraping the exporter's own `/metrics` endpoint with `curl`** — there is no downstream backend to watch, because this exporter exposes rather than pushes (the `debug` exporter below is included only to confirm what entered the pipeline). The recipe drives an OTLP pipeline with `telemetrygen` and reads the exposed series back. This is a **minimal repro** — it omits `memory_limiter` and other production scaffolding on purpose, to isolate the exporter's behavior. Verified on `otel/opentelemetry-collector-contrib:0.154.0`.

Config (`config.yaml`) — a `metrics` pipeline from the `otlp` receiver into both `prometheus` and `debug`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  prometheus:
    endpoint: 0.0.0.0:9464
    namespace: testns
    const_labels:
      verified_by: ollygarden
    send_timestamps: true
  debug:
    verbosity: detailed
service:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [prometheus, debug]
```

Run it, publishing both the OTLP receiver and the `/metrics` server to the host:

```bash
docker run -d --name prom-exp \
  -p 14317:4317 -p 19464:9464 \
  -v "$PWD/config.yaml:/etc/otelcol-contrib/config.yaml" \
  otel/opentelemetry-collector-contrib:0.154.0
```

Drive it with `telemetrygen` — see the `otel-telemetrygen` skill. This emits one cumulative Sum metric named `gen` with **3 datapoints, values 0, 1, 2** (confirmed in the `debug` exporter's output as three `Metric #0 Name: gen` lines with `Value: 0/1/2`):

```bash
telemetrygen metrics --otlp-insecure --otlp-endpoint localhost:14317 \
  --metrics 3 --workers 1 --metric-type Sum \
  --telemetry-attributes 'tg_label="v1"'
```

**What proves it worked:** scrape the exporter's own endpoint:

```bash
curl -s localhost:19464/metrics
```

produces exactly:

```
# HELP testns_gen_total 
# TYPE testns_gen_total counter
testns_gen_total{job="telemetrygen",otel_scope_name="",otel_scope_schema_url="",otel_scope_version="",tg_label="v1",verified_by="ollygarden"} 2 1781369164374
```

Each piece confirms a specific behavior:

- **`namespace: testns`** → the `testns_` series prefix.
- **`_total` suffix and `# TYPE … counter`** → `add_metric_suffixes` defaults to `true`, and this is a counter, so the type/unit suffix is appended.
- **`const_labels`** → the `verified_by="ollygarden"` label on the series.
- **the telemetry attribute** → the `tg_label="v1"` label.
- **scope labels present** (`otel_scope_name`/`otel_scope_version`/`otel_scope_schema_url`, empty here) → `without_scope_info` defaults to `false`.
- **`send_timestamps: true`** → the trailing millisecond timestamp `1781369164374`.
- **latest-value accumulation** → the three datapoints had values 0, 1, 2 at increasing timestamps; the exposed value is **2**. The exporter keeps and exposes the **latest** aggregated value per series (its in-memory accumulator picks the most recent timestamp), not a sum or a stream.

No `target_info` series appeared with `telemetrygen`'s minimal resource — that mechanism is covered in [advanced.md](advanced.md), not claimed here.

Tear down:

```bash
docker rm -f prom-exp
```
