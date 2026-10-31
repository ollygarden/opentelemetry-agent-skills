# `transform`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`transform` ships in the `contrib` and `k8s` distributions (and `core` since v0.164.0), so a stock contrib collector can run this.

Verified on `otel/opentelemetry-collector-contrib:0.159.0` with
`ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.159.0`
(2026-08-27). This recipe's schema is unchanged through v0.160.0; it does not use `shared_cache`.

Config (`transform-verify.yaml`) — add an `env` attribute to every log record that telemetrygen does **not** itself produce:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  transform:
    error_mode: ignore
    log_statements:
      - set(log.attributes["env"], "prod")
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [transform]
      exporters: [debug]
```

Send a batch of logs — see the `otel-telemetrygen` skill:

```bash
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 5 --body "transform me"
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-insecure` (bool), `--otlp-endpoint` (string), `--logs` (int, per-worker count; ignored if `--duration` is set, so no `--duration` here), and `--body` (string). No flag here invents a value.

**What proves it worked:** the `debug` exporter prints all 5 log records, and each carries an `env: Str(prod)` attribute in its `Attributes` block. telemetrygen never emits an `env` log attribute, so its presence in the output is unambiguous proof the statement ran. The log body ("transform me") is unchanged — only the attribute was added.

> The exact `debug` rendering of attributes (`-> env: Str(prod)`) can shift between collector versions; what matters is that the `env`/`prod` pair appears on records that didn't have it on input.
