# `otlp_grpc` exporter: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

The exporter ships in every distribution (`core`, `contrib`, `k8s`, `otlp`). The cleanest proof is a **two-collector chain**: collector **A** receives OTLP and exports it to collector **B**; collector **B** receives it and prints it with `debug`. If spans sent to A appear in B's log, the export hop worked. Verified on `otel/opentelemetry-collector-contrib:0.154.0`.

> The config uses the deprecated alias `otlp` for the exporter (still the most common form). The canonical `otlp_grpc` name is identical in behavior. This recipe is a **minimal repro** — it omits `memory_limiter` and other production scaffolding on purpose, to isolate the export hop.

Collector **A** config (`a.yaml`) — receives, then exports to B over plaintext gRPC:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  otlp:                            # the exporter under test (alias of otlp_grpc)
    endpoint: otlp-b:4317
    tls:
      insecure: true               # plaintext gRPC to B
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp]
```

Collector **B** config (`b.yaml`) — receives and prints:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  debug:
    verbosity: detailed            # basic verbosity prints no per-span lines
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
```

Start both on one Docker network (distinct names; only A publishes a host port):

```bash
docker network create otlpnet
IMG=otel/opentelemetry-collector-contrib:0.154.0
docker run -d --name otlp-b --network otlpnet \
  -v "$PWD/b.yaml:/etc/otelcol-contrib/config.yaml" $IMG
docker run -d --name otlp-a --network otlpnet -p 4317:4317 \
  -v "$PWD/a.yaml:/etc/otelcol-contrib/config.yaml" $IMG
```

Send 5 spans into **A** — see the `otel-telemetrygen` skill:

```bash
telemetrygen traces --otlp-endpoint localhost:4317 --otlp-insecure \
  --traces 5 --workers 1
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-endpoint` (string), `--otlp-insecure` (bool, plaintext to A's TLS-less receiver), `--traces` (int, **per worker**; ignored when `--duration` is set, so no `--duration` and `--workers 1` makes the count exact). No flag invents a value.

**What proves it worked:** the spans must appear in **B's** debug log, having traversed A's `otlp` exporter:

```bash
docker logs otlp-b 2>&1 | grep -c 'Span #'
```

`--traces N` counts **traces, not spans** — `telemetrygen` emits **2 spans per trace** (a parent and a child), so `--traces 5 --workers 1` produces **10 span records**, all of which should reach B. Verified on `otel/opentelemetry-collector-contrib:0.154.0`:

```
10
```

Collector A also logs, at startup, the exporter's deprecation warning when the `otlp` alias is used:

```
warn  builders/builders.go:40  "otlp" alias is deprecated; use "otlp_grpc" instead  {"otelcol.component.id": "otlp", "otelcol.component.kind": "exporter", ...}
```

A count of 0 in B (but spans visible in A's own log, if you add a `debug` exporter to A) points at the export hop — usually a wrong `endpoint`, or missing `tls.insecure: true` against a plaintext B.

Tear down:

```bash
docker rm -f otlp-a otlp-b
docker network rm otlpnet
```
