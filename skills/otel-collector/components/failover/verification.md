# `failover`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

The recipe uses two collectors on one host. **A** runs the connector, with a primary level that exports over OTLP to **B** and a backup level that writes a local `file`. Stopping B makes the primary fail, and starting it again shows failback. The primary exporter has `sending_queue` and `retry_on_failure` disabled, so A's connector sees each export error at once ([why](quirks.md#async-stages-in-a-priority-pipeline-hide-failures)).

Collector **A** (`failover-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 127.0.0.1:4317
connectors:
  failover:
    priority_levels:
      - [logs/primary]
      - [logs/backup]
    retry_interval: 10s
exporters:
  otlp_grpc/primary:
    endpoint: 127.0.0.1:5317   # IPv4 loopback: B does not listen on [::1]
    tls:
      insecure: true
    sending_queue:
      enabled: false
    retry_on_failure:
      enabled: false
  file/backup:
    path: /tmp/failover-backup.json
service:
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [failover]
    logs/primary:
      receivers: [failover]
      exporters: [otlp_grpc/primary]
    logs/backup:
      receivers: [failover]
      exporters: [file/backup]
```

Collector **B**, the primary destination (`failover-backend.yaml`). Its internal metrics are off so it does not clash with A on port 8888:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 127.0.0.1:5317
exporters:
  file/primary:
    path: /tmp/failover-primary.json
    append: true             # B restarts during the test; keep earlier batches
service:
  telemetry:
    metrics:
      level: none
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [file/primary]
```

Start B and then A, and send three phases of 5 logs each; each log body names its phase:

```bash
rm -f /tmp/failover-primary.json /tmp/failover-backup.json   # B appends: stale output fakes a pass
otelcol-contrib --config failover-backend.yaml & B=$!
otelcol-contrib --config failover-verify.yaml & A=$!
sleep 5

send()  { telemetrygen logs --otlp-insecure --otlp-endpoint 127.0.0.1:4317 --logs 5 --workers 1 --rate 1 --body "$1"; }
count() { for f in /tmp/failover-primary.json /tmp/failover-backup.json; do
            echo "$f:"; grep -o '"body":{"stringValue":"[a-z0-9]*"' "$f" 2>/dev/null | cut -d'"' -f6 | sort | uniq -c
          done; }

send phase1                                          # primary up
kill $B; send phase2                                 # primary down -> failover
otelcol-contrib --config failover-backend.yaml & B=$!
sleep 15; send phase3; sleep 2; count                # wait > retry_interval -> failback; files flush every 1s
kill $A $B
```

Flags: the `otel-telemetrygen` skill's `references/flags.md`.

**What proves it worked.** `count` output:

```
/tmp/failover-primary.json:
      5 phase1
      5 phase3
/tmp/failover-backup.json:
      5 phase2
```

`phase2` reached only the backup: failover worked. `phase3` reached the primary: failback worked. At the failover, A logs `Exporting failed. Rejecting data.` from `otlp_grpc/primary` (`... connect: connection refused`, `"rejected_items": 5`).

If `phase3` lands in the backup, B was not reachable when the next probe ran. Send again after another `retry_interval`.

## Variant: an exporter queue hides the outage

Turn the primary's queue and retry back on, with a tiny queue so the effect shows quickly:

```yaml
  otlp_grpc/primary:
    # endpoint and tls as above
    sending_queue:
      enabled: true
      num_consumers: 1
      queue_size: 3
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_elapsed_time: 40s
```

With B down, send seven single-record batches:

```bash
for i in 1 2 3 4 5 6 7; do
  telemetrygen logs --otlp-insecure --otlp-endpoint 127.0.0.1:4317 --logs 1 --workers 1 --rate 1 --body "q$i"
done
```

Observed:
- The backup holds `q4` through `q7`.
- `q1`–`q3` never reach it. The primary's queue accepted them (`q1` in flight, `q2`–`q3` queued), so the connector saw success.
- `q4` was rejected with `"error": "sending queue is full"`, and that rejection triggered the failover.
- `q1`–`q3` are then retried inside the primary exporter. Each is dropped with `Exporting failed. Dropping data.` once its `max_elapsed_time` runs out.

The other rows of the [failure table](quirks.md#async-stages-in-a-priority-pipeline-hide-failures) for the primary exporter (queue without retry, retry without queue, `wait_for_result: true`) were also observed as stated, each with B down.
