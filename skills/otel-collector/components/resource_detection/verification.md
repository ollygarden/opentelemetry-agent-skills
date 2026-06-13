# `resource_detection` — verification

Confirms that detected attributes appear in the **Resource attributes** block of telemetry. Uses the two detectors that work without a cloud/metadata service: `env` (reads `OTEL_RESOURCE_ATTRIBUTES` from the Collector's environment) and `system` (reads the host machine).

See the `otel-collector` SKILL.md **Verification harness** note: this config is a minimal repro and omits `memory_limiter` and other production scaffolding on purpose.

## Config (`rdcol.yaml`)

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  resource_detection:
    detectors: [env, system]
    override: true
    system:
      hostname_sources: ["os"]

exporters:
  debug:
    verbosity: detailed

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resource_detection]
      exporters: [debug]
```

## Run

The `env` detector reads the **Collector's** environment, so set `OTEL_RESOURCE_ATTRIBUTES` on the container (not on telemetrygen):

```bash
docker run --rm --name rdcol -p 4317:4317 \
  -e OTEL_RESOURCE_ATTRIBUTES='deployment.environment.name=prod,my.detector.test=hello' \
  -v "$PWD/rdcol.yaml:/etc/otelcol-contrib/config.yaml" \
  otel/opentelemetry-collector-contrib:0.154.0

# in another shell — telemetrygen emits 2 spans per trace
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 5 --workers 1
```

## Expected output

The `debug` exporter prints a **Resource attributes** block carrying both the `env`-detected and `system`-detected attributes:

```
Resource attributes:
     -> service.name: Str(telemetrygen)
     -> deployment.environment.name: Str(prod)   # from env detector
     -> my.detector.test: Str(hello)             # from env detector
     -> host.name: Str(<container hostname>)      # from system detector
     -> os.type: Str(linux)                       # from system detector
```

10 spans total reach the exporter (5 traces × 2 spans). Tear down with `docker rm -f rdcol`.

## Verified result (contrib 0.154.0)

Run end-to-end on `otel/opentelemetry-collector-contrib:0.154.0`. The `env` detector populated `deployment.environment.name=prod` and `my.detector.test=hello` from `OTEL_RESOURCE_ATTRIBUTES`; the `system` detector populated `host.name` (the container ID, via `hostname_sources: ["os"]`) and `os.type=linux`. All 10 spans carried the merged resource, and telemetrygen's own `service.name=telemetrygen` was kept (neither detector produces it).

`override` was confirmed separately by setting `OTEL_RESOURCE_ATTRIBUTES=service.name=fromenv` (which collides with telemetrygen's `service.name=telemetrygen`): with `override: true` the resource showed `service.name=fromenv` (detector wins); with `override: false` it showed `service.name=telemetrygen` (incoming value preserved).
