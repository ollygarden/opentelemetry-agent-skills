# `prometheus_remote_write` exporter: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

Unlike the pull-based [`prometheus`](../prometheus_exporter/verification.md) exporter (where you scrape the exporter's own `/metrics`), here you **push** to a real backend and query the series back from it. The recipe runs two containers on a shared Docker network: a Prometheus server with the remote-write receiver enabled, and a collector that pushes into it. This is a **minimal repro** — it omits `memory_limiter` and `batch` on purpose, to isolate the exporter's behavior.

**Verified on `otel/opentelemetry-collector-contrib:0.154.0` (2026-06-14).** A `telemetrygen` cumulative Sum named `gen` (datapoints 0/1/2) pushed via RW1 landed in Prometheus as `testns_gen_total{job="telemetrygen", verified_by="ollygarden"} 2` — confirming the `testns_` namespace prefix, the `_total` counter suffix (`add_metric_suffixes` defaults to `true`), the `service.name`→`job` mapping, the `external_labels` entry, RW1 (`ProtoMsg: prometheus.WriteRequest`), and that `tls.insecure: true` lets the plaintext push succeed. The deprecated alias `prometheusremotewrite` logs `"prometheusremotewrite" alias is deprecated; use "prometheus_remote_write" instead` and otherwise behaves identically.

Create a network:

```bash
docker network create prw-net
```

Start Prometheus with the remote-write receiver enabled (exposes `/api/v1/write`, accepts RW1). The image ships a default config at `/etc/prometheus/prometheus.yml`:

```bash
docker run -d --name prw-prom --network prw-net -p 19090:9090 \
  prom/prometheus:v3.1.0 \
  --config.file=/etc/prometheus/prometheus.yml \
  --web.enable-remote-write-receiver
```

Collector config (`config.yaml`) — a `metrics` pipeline from the `otlp` receiver into `prometheus_remote_write`, pushing to the Prometheus container by its network alias (`debug` is added only to see what arrives):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  debug:
    verbosity: detailed
  prometheus_remote_write:
    endpoint: http://prw-prom:9090/api/v1/write
    tls:
      insecure: true
    namespace: testns
    external_labels:
      verified_by: ollygarden
service:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [prometheus_remote_write, debug]
```

> The exporter `endpoint` host must match the Prometheus container's network alias (`prw-prom` here). A wrong host doesn't fail fast — the push hangs until the request timeout and the collector logs `Permanent error: context deadline exceeded`, `Dropping data`.

Run the collector on the same network:

```bash
docker run -d --name prw-col --network prw-net \
  -v "$PWD/config.yaml:/etc/otelcol-contrib/config.yaml" \
  otel/opentelemetry-collector-contrib:0.154.0
```

Drive it with `telemetrygen` — see the `otel-telemetrygen` skill. Run it as a container **on the same network** targeting the collector's alias (this avoids a host-port-mapped gRPC stall some Docker Desktop setups hit). This emits one **cumulative Sum** named `gen`; `telemetrygen` already defaults to cumulative temporality, which is required here — the exporter **drops non-cumulative (delta) monotonic sums** (see [quirks](quirks.md)). `--metrics N` is **per worker** and is **ignored when `--duration` is set**, so use `--workers 1` and no `--duration`:

```bash
docker run --rm --network prw-net \
  ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:latest \
  metrics --otlp-insecure --otlp-endpoint prw-col:4317 \
  --metrics 3 --workers 1 --metric-type Sum
```

**What proves it worked:** query the series back from Prometheus. The counter gets the `_total` suffix (`add_metric_suffixes` defaults to `true`) and the `testns_` namespace prefix:

```bash
curl -s 'http://localhost:19090/api/v1/query?query=testns_gen_total'
```

```json
{"status":"success","data":{"resultType":"vector","result":[
  {"metric":{"__name__":"testns_gen_total","job":"telemetrygen","verified_by":"ollygarden"},
   "value":[1781411226.941,"2"]}]}}
```

- **`namespace: testns`** → the `testns_` series prefix.
- **`_total` suffix** → `add_metric_suffixes` defaults to `true`, and this is a counter.
- **`external_labels`** → the `verified_by="ollygarden"` label on the series.
- **`job="telemetrygen"`** → `service.name` is mapped to the `job` label.
- **value `2`** → the latest cumulative value of the 0/1/2 sequence.
- **`tls.insecure: true`** → plaintext push works against the receiver (TLS is on by default otherwise).

Tear down:

```bash
docker rm -f prw-col prw-prom
docker network rm prw-net
```
