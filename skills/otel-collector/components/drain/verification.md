# `drain`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

Confirm `drain` is in your distribution (check the component's metadata); build via OCB if it is not yet bundled.

Config (`drain-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  drain:
    template_attribute: log.record.template
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [drain]
      exporters: [debug]
```

Send log records that share structure but differ in values (see the `otel-telemetrygen` skill):

```bash
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 100 --body "user alice logged in from 10.0.0.1" --duration 5s
```

telemetrygen sends a **constant** body per invocation (`--body` defaults to `"the message"`), so a single run produces one template with no wildcards. To watch several distinct values collapse to one template (e.g. `user <*> logged in from <*>`), run the command a few times with different `--body` values — for example `"user bob logged in from 192.168.1.1"` and `"user carol logged in from 172.16.0.9"` — or point the pipeline at a real log source.

**What proves it worked:** the `debug` exporter shows each record annotated with a `log.record.template` attribute holding the clustered template string (e.g. `user <*> logged in from <*>`).
