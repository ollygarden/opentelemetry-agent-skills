# `clickhouse` exporter: known quirks

Facts trace to contrib v0.165.1 and `clickhouse-go` v2.47.0 (see [configuration](configuration.md)).

## Schema DDL failure stops the Collector

With the default `create_schema: true`, the DDL runs in the exporter's `start`. An unreachable server, a missing `CREATE` grant, or a DDL the server rejects makes the Collector exit (both observed):

```
Error: cannot start pipelines: failed to start "clickhouse" exporter: create database: dial tcp 10.89.0.2:9999: connect: connection refused
Error: cannot start pipelines: failed to start "clickhouse" exporter: create database: code: 497, message: writer: Not enough privileges. To execute this query, it's necessary to have the grant CREATE DATABASE ON otel.*
```

ClickHouse checks the grant even for `CREATE DATABASE IF NOT EXISTS` on a database that already exists, so a user with only `INSERT, SELECT, SHOW COLUMNS ON otel.*` cannot start with `create_schema: true` ([`GRANT`](https://clickhouse.com/docs/reference/statements/grant)). Grant the `CREATE` rights, or set `create_schema: false` and create the schema yourself ([schema management](advanced.md#schema-management)).

With `create_schema: false`, a failed startup `DESC TABLE` ([schema management](advanced.md#schema-management)) is only logged. An unreachable server then does not stop startup; requests are retried and dropped (observed: `no more retries left: dial tcp …: connect: connection refused`, `dropped_items: 3`).

## Every error is retried

The exporter returns no permanent errors, so [`retry_on_failure`](../otlp_exporter/configuration.md#retry_on_failure) retries each failed push until `max_elapsed_time`, and then the queue logs `Exporting failed. Dropping data.` (with the queue disabled, the exporter logs `Exporting failed. Rejecting data.` and returns the error). That includes errors that repeat on every attempt: ClickHouse exceptions such as a missing column (16), missing grants (497) and bad credentials (516); dial and TLS errors; and client-side conversion and JSON errors. Two consequences:

- **The retry budget is spent on errors that cannot succeed.** A caller that waits for the result (exporter queue off, or `wait_for_result: true`) waits per batch until the retries end (sooner if the caller's deadline comes first) before it sees the error. What each queue setting returns to a [`failover`](../failover/README.md) connector: [async stages hide failures](../failover/quirks.md#async-stages-in-a-priority-pipeline-hide-failures).
- **Retries can duplicate rows.** A retry resends the whole request, and rows ClickHouse already wrote are written again. One metrics push runs one `INSERT` per metric type ([rows per `INSERT`](advanced.md#rows-per-insert-and-batching)); when one type fails, the retry re-inserts the others. When the push into `otel_traces_trace_id_ts_mv` failed with 497, the spans were already in `otel_traces`: five attempts left 10 rows for 2 distinct `SpanId`s (observed). The exporter sets no deduplication settings; see ClickHouse's [deduplicating inserts on retries](https://clickhouse.com/docs/concepts/features/operations/insert/deduplicating-inserts-on-retries).

## `IF NOT EXISTS` never changes existing tables

Every DDL statement is `IF NOT EXISTS`, so schema keys (`ttl`, engines, `json`, …) and DDL changes in a release apply only to tables created afterwards. Examples: tables created before v0.165.0 keep `ZSTD(1)` codecs, before v0.157.0 the older metric-table layout, and before v0.151.0 the older logs `ORDER BY` and no `__otel_materialized_*` columns. Optional columns that are missing, such as `EventName`, are skipped on `INSERT`. Missing required columns fail every `INSERT`. Add columns with `ALTER TABLE`; the exporter never runs it.

## Metric timestamps are second-precision

Since v0.157.0, the metric tables declare `StartTimeUnix`, `TimeUnix`, and exemplar `TimeUnix` as `DateTime` (seconds). Logs, traces, and profiles use `DateTime64(9)`. Older metric tables keep `DateTime64(9)` ([why](#if-not-exists-never-changes-existing-tables)).

## Silently ignored settings

None of these logs a warning:

- `password` without `username`: the DSN user info, or the driver default user `default`, applies.
- `metrics_table_name`: it applies only when every `metrics_tables.<type>.name` is empty, and the factory fills all five, so `metrics_table_name: legacy` still creates `otel_metrics_*` tables (observed).
- `tls.ca_pem`, `cert_pem`, or `key_pem` alone: they do not trigger the `tls` block (below).
- An unrecognized `compress` value, such as `snappy`: compression is disabled.

## The `tls` block replaces DSN TLS

If any of `tls.ca_file`, `cert_file`, `key_file`, `server_name_override`, `insecure`, or `insecure_skip_verify` is set, the exporter builds the TLS config from the `tls` block and discards the one from `secure=true` / `skip_verify` / `tls_server_name`:

- On a native endpoint (`tcp://`, `clickhouse://`), `tls.ca_file` enables TLS without `secure=true`.
- On a native endpoint, `tls.insecure: true` without a CA leaves the connection in **plaintext**, even with `secure=true`. HTTP endpoints follow the URL scheme.

## Setting values are lowercased

In `clickhouse-go` v2.47.0, `true` and `false` become `1` and `0`, integers stay integers, and any other query-string value is lowercased before it is sent as a setting. Observed: `connection_params: { log_comment: OTel-Exporter }` reached `system.query_log` as `otel-exporter`.

## Profiles

The profiles signal needs the `service.profilesSupport` gate (`otel-profiles` skill). The profiles table always uses `text` indexes, so creating it needs ClickHouse 26.2 or newer.

## Metrics log lines can name the wrong instance

Insert logs from every metrics instance share one logger, set by the last instance to start, so their component ID can be wrong.

## Errors → fixes

| Condition | Error | Fix |
|-----------|-------|-----|
| `endpoint` unset | `endpoint must be specified` | Set `endpoint`. |
| Host but no scheme (`127.0.0.1:9000`) | `endpoint must be url format: parse "127.0.0.1:9000": first path segment in URL cannot contain colon` | Use a URL with a scheme and a host, such as `tcp://host:9000`. |
| No host (`localhost:9000` parses as scheme `localhost`) | `parse dsn address failed` | As above. |
| `http://` with `secure=true` | `clickhouse [dsn parse]: http with TLS specify` | Use `https://`, or remove `secure=true`. |
| Unparsable driver option (`dial_timeout`, `secure`, `compress_level`, …) | `clickhouse [dsn parse]: …` or `<option> invalid value: …` | Fix the value ([DSN reference](https://github.com/ClickHouse/clickhouse-go/blob/v2.47.0/README.md#dsn)). |
| `compress: gzip`, `deflate`, or `br` on a native endpoint | Passes validation; every connection fails with `unsupported compression method for native protocol`, and with `create_schema: true` the Collector exits at start (observed) | Use `lz4`, `lz4hc`, `zstd`, or `none`, or an HTTP endpoint. |
| No `CREATE` grant, `create_schema: true` | `create database: code: 497 …` at start | [Grant `CREATE`, or `create_schema: false`](#schema-ddl-failure-stops-the-collector). |
| `DESC TABLE` fails, `create_schema: false` | `schema detection failed` (logged) | Make the table readable at start; detection runs only then, so optional columns are not written until the next start. |
| Table lacks a required column | `code: 16, message: No such column ScopeSchemaUrl in table otel.otel_logs_old` on every push (observed) | `ALTER TABLE … ADD COLUMN` ([why](#if-not-exists-never-changes-existing-tables)). |
| Only `INSERT ON otel.*`, `create_schema: false`, default traces tables | `code: 497 ... necessary to have the grant SELECT(TraceId, Timestamp) ON otel.otel_traces: while pushing to view otel.otel_traces_trace_id_ts_mv` (observed; spans still land in `otel_traces`) | Also grant `SELECT` on the traces table. `INSERT ON otel.*` alone writes logs. |
| Bad credentials, `create_schema: false` (with `true`, startup fails) | `code: 516` on every push (observed) | Fix `username` / `password`. |

The last three rows fail every push until `max_elapsed_time` ([every error is retried](#every-error-is-retried)). Retry-block validation is generic: [`retry_on_failure`](../otlp_exporter/configuration.md#retry_on_failure).
