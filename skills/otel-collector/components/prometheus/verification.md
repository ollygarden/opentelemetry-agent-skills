# `prometheus` receiver: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

The `prometheus` receiver is **scrape-based**, so `telemetrygen` does **not** apply — there is no
OTLP endpoint to push into; the receiver pulls from a Prometheus-format HTTP target. The
self-contained way to test it is to have the collector **scrape its own internal telemetry
endpoint**: `service.telemetry.metrics` can expose a Prometheus `/metrics` endpoint, and the
`prometheus` receiver scrapes that, proving the full scrape → parse → OTLP → `debug` path with no
external target. Verified on `otel/opentelemetry-collector-contrib:0.154.0`.

This is a deliberately minimal repro: no `memory_limiter`, single pipeline. Do not copy it verbatim
into production — a real metrics pipeline should front the receiver with `memory_limiter`.

Config (`prometheus-verify.yaml`) — the receiver scrapes the collector's own metrics port `:8888`,
`debug` at detailed verbosity so each metric is printed:

```yaml
receivers:
  prometheus:
    config:
      scrape_configs:
        - job_name: 'otel-self'
          scrape_interval: 5s
          static_configs:
            - targets: ['0.0.0.0:8888']
exporters:
  debug:
    verbosity: detailed
service:
  telemetry:
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: 0.0.0.0
                port: 8888
  pipelines:
    metrics:
      receivers: [prometheus]
      exporters: [debug]
```

Start the collector, publishing the telemetry port so the self-scrape target resolves:

```bash
IMG=otel/opentelemetry-collector-contrib:0.154.0
docker run -d --name prom-recv -p 8888:8888 \
  -v "$PWD/prometheus-verify.yaml:/etc/otelcol-contrib/config.yaml" $IMG
```

Wait ~12s so at least two 5s scrape intervals elapse (the first scrape fires while the collector is
still warming up and reports fewer series), then read the per-scrape batch summary the `debug`
exporter logs each time the receiver delivers a scrape:

```bash
sleep 12
docker logs prom-recv 2>&1 | grep '"otelcol.component.id": "debug"'
```

**What proves it worked:** each line is one completed scrape → parse → OTLP → `debug` cycle. Once
warmed up, every 5s scrape of the collector's own `/metrics` endpoint delivers a steady batch.
Verified on `otel/opentelemetry-collector-contrib:0.154.0` — the first (warm-up) scrape reported
`metrics: 12`, then each subsequent scrape was stable:

```
... "otelcol.component.id": "debug", ... "resource metrics": 1, "metrics": 17, "data points": 18}
```

Spot-check that a well-known self-metric came through (it appears once per scrape):

```bash
docker logs prom-recv 2>&1 | grep -c '\-> Name: otelcol_process_uptime'
```

returns one hit per elapsed scrape, and the distinct `otelcol_*` series scraped is **11**
(`docker logs prom-recv 2>&1 | grep '\-> Name: otelcol_' | sort -u | wc -l`).

If `debug` logs no batch lines: the most likely cause is that no scrape interval has elapsed yet
(wait longer), or the `:8888` target is unreachable — confirm `service.telemetry.metrics` exposes
the Prometheus reader on the same host/port the scrape config targets.

Tear down:

```bash
docker rm -f prom-recv
```
