# `clickhouse` exporter: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

This recipe sends `telemetrygen` logs and traces through a contrib Collector into a disposable ClickHouse server, then queries ClickHouse to prove that rows landed and that the exporter created its schema with the default `create_schema: true`. Commands are bash; the output blocks are from running them as pinned.

Create a network and start ClickHouse with a password user. Every loop gives up after 60s:

```bash
docker network create ch-net
docker run -d --name ch-server --network ch-net --ulimit nofile=262144:262144 \
  -e CLICKHOUSE_USER=otel -e CLICKHOUSE_PASSWORD=otel \
  clickhouse/clickhouse-server:26.8.49.6
ch() { docker exec ch-server clickhouse-client --user otel --password otel "$@"; }
for _ in $(seq 60); do ch -q 'SELECT 1' >/dev/null 2>&1 && break; sleep 1; done
```

Collector config (`config.yaml`). The exporter uses the native protocol on 9000, and it creates database `otel` and its tables at start:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  clickhouse:
    endpoint: tcp://ch-server:9000
    username: otel
    password: otel
    database: otel
service:
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [clickhouse]
    traces:
      receivers: [otlp]
      exporters: [clickhouse]
```

Start the Collector and wait for readiness. If ClickHouse is unreachable here, the Collector exits instead ([quirks](quirks.md#schema-ddl-failure-stops-the-collector)):

```bash
docker run -d --name ch-col --network ch-net \
  -v "$PWD/config.yaml:/etc/otelcol-contrib/config.yaml:ro" \
  otel/opentelemetry-collector-contrib:0.165.0
for _ in $(seq 60); do docker logs ch-col 2>&1 | grep -q 'Everything is ready' && break; sleep 1; done
```

Send 10 logs and 5 traces of 2 spans each (flags: the `otel-telemetrygen` skill's `references/flags.md`), then wait for the rows:

```bash
TG=ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.165.1
docker run --rm --network ch-net $TG logs --otlp-insecure \
  --otlp-endpoint ch-col:4317 --logs 10 --workers 1 --rate 1
docker run --rm --network ch-net $TG traces --otlp-insecure \
  --otlp-endpoint ch-col:4317 --traces 5 --workers 1 --rate 1
counts() { ch -q 'SELECT (SELECT count() FROM otel.otel_logs), (SELECT count() FROM otel.otel_traces),
  (SELECT uniqExact(TraceId) FROM otel.otel_traces_trace_id_ts) FORMAT CSV'; }
for _ in $(seq 60); do [ "$(counts)" = "10,10,5" ] && break; sleep 1; done; counts
```

Then read the schema and the `INSERT` shape from `system.query_log`:

```bash
ch --multiquery -q "
SYSTEM FLUSH LOGS;
SHOW TABLES FROM otel;
SELECT tables[1] AS tbl, count() AS inserts, sum(written_rows) AS rows,
  any(client_name) AS client, any(Settings['max_execution_time']) AS max_exec
FROM system.query_log
WHERE type = 'QueryFinish' AND query_kind = 'Insert' AND client_name LIKE 'otelcol/%'
GROUP BY tbl ORDER BY tbl FORMAT TSV"
```

**What proves it worked:**

```
10,10,5
otel_logs
otel_traces
otel_traces_trace_id_ts
otel_traces_trace_id_ts_mv
otel.otel_logs	1	10	otelcol/0.165.0 clickhouse-go/2.47.0 (lv:go/1.27.3; os:linux)	9
otel.otel_traces	5	10	otelcol/0.165.0 clickhouse-go/2.47.0 (lv:go/1.27.3; os:linux)	9
```

- `10,10,5`: every log and span landed, and the materialized view filled the trace-ID lookup table. The view writes one row per `TraceId` per inserted block, so a trace split across `INSERT`s gets several rows; count distinct IDs, not rows.
- The four tables are what `create_schema` made.
- `otel.otel_logs` shows one `INSERT` of 10 rows: the 10 logs arrived in one OTLP request ([rows per `INSERT`](advanced.md#rows-per-insert-and-batching)). The traces `INSERT` count follows how `telemetrygen` split its exports: 5 here, 6 in another run.
- `client` comes from `client_info_product`, and `max_exec` `9` is derived from `timeout: 5s` ([DSN and settings](configuration.md#dsn-and-settings)).

Tear down:

```bash
docker rm -f ch-col ch-server
docker network rm ch-net
```
