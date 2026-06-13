# `file_log` receiver: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

> **`telemetrygen` cannot drive this receiver.** `telemetrygen` emits OTLP over the network; the
> `file_log` receiver reads files from disk. There is no `telemetrygen` flag that writes a log
> file. The input here is a **file you write yourself** and mount into the collector container.

The `file_log` receiver ships in the `contrib` and `k8s` distributions. This recipe runs **one**
collector — `file_log` tailing a mounted directory, feeding a `debug` exporter — and proves the
pipeline by writing three pre-formatted lines and reading them back parsed. Verified on
`otel/opentelemetry-collector-contrib:0.154.0`.

Config (`file_log.yaml`) — note `start_at: beginning`, without which an existing idle file is
skipped (the receiver's top gotcha):

```yaml
receivers:
  file_log:
    include: [ /logs/*.log ]
    start_at: beginning            # default `end` would read nothing from a static file
    operators:
      - type: regex_parser
        regex: '^(?P<ts>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) (?P<sev>[A-Z]+) (?P<msg>.*)$'
        timestamp:
          parse_from: attributes.ts
          layout: '%Y-%m-%d %H:%M:%S'
        severity:
          parse_from: attributes.sev
exporters:
  debug:
    verbosity: detailed            # basic verbosity prints no per-record lines
service:
  pipelines:
    logs:
      receivers: [file_log]
      exporters: [debug]
```

Create the log file the receiver will tail:

```bash
mkdir -p /tmp/filelog-verify/logs
cat > /tmp/filelog-verify/logs/app.log <<'EOF'
2026-06-13 09:00:01 INFO service started
2026-06-13 09:00:02 WARN cache miss for key=42
2026-06-13 09:00:03 ERROR failed to connect to upstream
EOF
```

Start the collector, mounting both the config and the log directory:

```bash
IMG=otel/opentelemetry-collector-contrib:0.154.0
docker run -d --name fcol \
  -v "$PWD/file_log.yaml:/etc/otelcol-contrib/config.yaml" \
  -v /tmp/filelog-verify/logs:/logs \
  $IMG
```

**What proves it worked:** count the records the `debug` exporter logged, and confirm severity
was parsed:

```bash
docker logs fcol 2>&1 | grep -c 'LogRecord #'
docker logs fcol 2>&1 | grep -E 'SeverityText|-> msg:'
```

Verified output on `otel/opentelemetry-collector-contrib:0.154.0` — **3 records**, each with the
raw line in the body, a parsed `msg` attribute, the default `log.file.name=app.log` attribute,
and severity mapped from the text:

```
3
SeverityText: INFO
     -> msg: Str(service started)
SeverityText: WARN
     -> msg: Str(cache miss for key=42)
SeverityText: ERROR
     -> msg: Str(failed to connect to upstream)
```

If the count is **0**, the most likely cause is the `start_at: end` default (the file existed
before the collector started and isn't being appended to) — set `start_at: beginning`, or append
a new line *after* the collector is running.

To watch live tailing instead, leave the collector running and append a line:

```bash
echo "2026-06-13 09:00:04 INFO new line after startup" >> /tmp/filelog-verify/logs/app.log
docker logs fcol 2>&1 | grep -c 'LogRecord #'     # now 4
```

Tear down:

```bash
docker rm -f fcol
rm -rf /tmp/filelog-verify
```

## Note on the deprecated alias

Using `filelog:` (the pre-v0.149.0 name) instead of `file_log:` still works but logs a warning
at startup — verified on 0.154.0:

```
warn  builders/builders.go:40  "filelog" alias is deprecated; use "file_log" instead  {"otelcol.component.id": "filelog", ...}
```
