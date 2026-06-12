# `routing`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`routing` ships in the `contrib` and `k8s` distributions, so a stock contrib collector can run this.

The idea: send logs carrying a resource attribute `tenant`, route the ones where `tenant == "acme"` to one pipeline and let everything else fall through to `default_pipelines`. Each output pipeline writes to its **own** `file` exporter, so the split is visible as two distinct files. Run telemetrygen twice — once with the matching attribute, once without — and check which file each batch lands in.

Config (`routing-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
connectors:
  routing:
    error_mode: ignore
    default_pipelines: [logs/default]
    table:
      - context: resource
        condition: attributes["tenant"] == "acme"
        pipelines: [logs/match]
exporters:
  file/match:
    path: /tmp/routed-match.json
  file/default:
    path: /tmp/routed-default.json
service:
  pipelines:
    logs/in:
      receivers: [otlp]
      exporters: [routing]
    logs/match:
      receivers: [routing]
      exporters: [file/match]
    logs/default:
      receivers: [routing]
      exporters: [file/default]
```

Send a matching batch and a non-matching batch — see the `otel-telemetrygen` skill:

```bash
# matches the route -> /tmp/routed-match.json
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 3 --workers 1 --otlp-attributes 'tenant="acme"'

# no matching attribute -> falls through to /tmp/routed-default.json
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 2 --workers 1 --otlp-attributes 'tenant="other"'
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-insecure` (bool), `--otlp-endpoint` (string), `--logs` (int, **per worker**; ignored if `--duration` is set, so no `--duration` here, and `--workers 1` makes the count exact), and `--otlp-attributes` (`key="value"`, **resource-level** — the right scope, since the route uses `context: resource` and reads `attributes["tenant"]` off the resource). No flag here invents a value.

**What proves it worked:** `/tmp/routed-match.json` holds the 3 records sent with `tenant="acme"` and `/tmp/routed-default.json` holds the 2 records sent with `tenant="other"`. telemetrygen sends nothing on a `tenant` attribute by default, so the split is unambiguous — records only reach `file/match` when the route condition matched, and only reach `file/default` via the unmatched fallback. The `file` exporter writes one newline-delimited JSON object per export, so count the log records inside each file (the resource-level `tenant` attribute appears once per batch, not once per record):

```bash
# each batch is one JSON line carrying its logRecords; count records, not the resource attr
for f in /tmp/routed-match.json /tmp/routed-default.json; do
  echo "$f: $(grep -o '"timeUnixNano"' "$f" | wc -l | tr -d ' ') records"
done
# => routed-match.json: 3 records, routed-default.json: 2 records
```

> Verified end-to-end on `otel/opentelemetry-collector-contrib:0.152.0`: the `tenant="acme"` batch landed only in `routed-match.json` (3 records) and the `tenant="other"` batch only in `routed-default.json` (2 records). Exact JSON field rendering can shift between collector versions; what matters is the matching records land in `routed-match.json` and the rest in `routed-default.json` — and that removing `default_pipelines` would make the non-matching batch vanish entirely (dropped), the behavior [quirks](quirks.md) warns about.
