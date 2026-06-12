# `load_balancing`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`load_balancing` ships in the `contrib` and `k8s` distributions. This recipe needs **three** collectors — one front-end running the exporter and **two** backends — so they can talk over a shared network. The idea: route by `service`, send four distinct `service.name`s, and confirm each service's spans always land on **exactly one** backend (affinity), with the services spread across both.

> Use **v0.153.0 or newer** — the type is `loadbalancing` (one word) in v0.152.0 and earlier; it was renamed to `load_balancing` in v0.153.0. This was verified on `otel/opentelemetry-collector-contrib:0.154.0`.

Front-end config (`front.yaml`) — the exporter under test, with a `static` resolver listing the two backends:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  load_balancing:
    routing_key: service          # each service.name hashes to one backend
    protocol:
      otlp:
        tls:
          insecure: true          # do NOT set protocol.otlp.endpoint — populated per backend
    resolver:
      static:
        hostnames:
          - lb-backend-a:4317
          - lb-backend-b:4317
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [load_balancing]
```

Backend config (`backend.yaml`, used by both backends) — a plain `debug` exporter so each backend prints what it received:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  debug:
    verbosity: detailed           # basic verbosity prints no per-span lines
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
```

Start all three on one Docker network (unique names; only the front-end publishes a host port):

```bash
docker network create lbnet
IMG=otel/opentelemetry-collector-contrib:0.154.0
docker run -d --name lb-backend-a --network lbnet -v "$PWD/backend.yaml:/etc/otelcol-contrib/config.yaml" $IMG
docker run -d --name lb-backend-b --network lbnet -v "$PWD/backend.yaml:/etc/otelcol-contrib/config.yaml" $IMG
docker run -d --name lb-front     --network lbnet -p 14317:4317 \
  -v "$PWD/front.yaml:/etc/otelcol-contrib/config.yaml" $IMG
```

Send four services (and `delta` twice, to prove the route is stable) — see the `otel-telemetrygen` skill:

```bash
for svc in alpha beta gamma delta; do
  telemetrygen traces --otlp-endpoint localhost:14317 --otlp-insecure \
    --traces 30 --workers 1 --service "$svc"
done
# repeat delta -> must hit the same backend as the first delta batch
telemetrygen traces --otlp-endpoint localhost:14317 --otlp-insecure \
  --traces 30 --workers 1 --service delta
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-endpoint` (string), `--otlp-insecure` (bool), `--traces` (int, **per worker**; ignored if `--duration` is set, so no `--duration` here and `--workers 1` makes the per-service count exact), and `--service` (string, default `telemetrygen`; sets the `service.name` resource attribute — the exact attribute `routing_key: service` hashes on). No flag invents a value.

**What proves it worked:** count which `service.name`s appear in each backend's debug log:

```bash
for c in lb-backend-a lb-backend-b; do
  echo "--- $c ---"
  docker logs "$c" 2>&1 | grep -oE 'service.name: Str\((alpha|beta|gamma|delta)\)' | sort | uniq -c
done
```

Each service must appear in **exactly one** backend — never both. Verified end-to-end on `otel/opentelemetry-collector-contrib:0.154.0`:

```
--- lb-backend-a ---
  30 service.name: Str(beta)
  39 service.name: Str(gamma)
--- lb-backend-b ---
  80 service.name: Str(alpha)
  34 service.name: Str(delta)
```

`alpha`/`delta` reached only `lb-backend-b`, `beta`/`gamma` only `lb-backend-a` — clean per-service affinity, with the four services split across both backends. (The counts are per-export-batch occurrences, not span totals — they vary with batching; what matters is which backend each service appears in.) The repeated `delta` batch landed on `lb-backend-b` again, confirming the routing is a deterministic hash of `service.name`, not round-robin. Which service maps to which backend depends on the hash and the backend list, so the exact split varies; what is invariant is that **a given service never appears in more than one backend**.

Tear down:

```bash
docker rm -f lb-front lb-backend-a lb-backend-b
docker network rm lbnet
```
