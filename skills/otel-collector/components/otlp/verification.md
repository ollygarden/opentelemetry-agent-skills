# `otlp` receiver: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

The `otlp` receiver ships in every distribution (`core`, `contrib`, `k8s`, `otlp`). This recipe runs **one** collector — the `otlp` receiver feeding a `debug` exporter — and sends spans into it with `telemetrygen`, first over gRPC (`:4317`) then over HTTP (`:4318`). Verified on `otel/opentelemetry-collector-contrib:0.154.0`.

> **Why `0.0.0.0`, not `localhost`?** The default endpoint binds to `localhost`, which inside a container only accepts loopback traffic from *within* the container. `telemetrygen` runs on the host and reaches the container through a published port, i.e. a non-loopback interface — so the receiver must bind `0.0.0.0` or it silently accepts nothing. This is the receiver's most common gotcha; the recipe sets it explicitly.

Config (`otlp.yaml`) — both protocols bound to `0.0.0.0`, `debug` at detailed verbosity so each span is printed:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
exporters:
  debug:
    verbosity: detailed           # basic verbosity prints no per-span lines
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
```

Start the collector, publishing both OTLP ports to the host:

```bash
IMG=otel/opentelemetry-collector-contrib:0.154.0
docker run -d --name otlp-recv -p 4317:4317 -p 4318:4318 \
  -v "$PWD/otlp.yaml:/etc/otelcol-contrib/config.yaml" $IMG
```

Send 5 spans over **gRPC** (port 4317) — see the `otel-telemetrygen` skill:

```bash
telemetrygen traces --otlp-endpoint localhost:4317 --otlp-insecure \
  --traces 5 --workers 1
```

Then send 5 more over **HTTP** (port 4318) with `--otlp-http`:

```bash
telemetrygen traces --otlp-http --otlp-endpoint localhost:4318 --otlp-insecure \
  --traces 5 --workers 1
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-endpoint` (string), `--otlp-insecure` (bool, plaintext — matches the `tls`-less receiver), `--otlp-http` (bool, switches to the HTTP exporter and the `:4318` path), and `--traces` (int, **per worker**; ignored when `--duration` is set, so no `--duration` here and `--workers 1` makes the count exact). No flag invents a value.

**What proves it worked:** count the spans the `debug` exporter logged:

```bash
docker logs otlp-recv 2>&1 | grep -c 'Span #'
```

`--traces N` counts **traces, not spans** — `telemetrygen` emits **2 spans per trace** (a parent and a child), so each `--traces 5 --workers 1` run produces **10 span records**, and the two runs (gRPC + HTTP) together yield **20**. Verified on `otel/opentelemetry-collector-contrib:0.154.0`:

```
20
```

If the count is 0, the most likely cause is a `localhost`-bound endpoint (data never reached the receiver) — confirm `endpoint: 0.0.0.0:…` in the config.

Tear down:

```bash
docker rm -f otlp-recv
```
