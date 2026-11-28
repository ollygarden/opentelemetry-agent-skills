# `clickhouse` exporter: advanced use-cases

Behavior below comes from contrib v0.165.1 source ([`exporter_logs.go`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/exporter/clickhouseexporter/exporter_logs.go), [`exporter_traces.go`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/exporter/clickhouseexporter/exporter_traces.go), [`exporter_metrics.go`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/exporter/clickhouseexporter/exporter_metrics.go), [`internal/clickhouse.go`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/exporter/clickhouseexporter/internal/clickhouse.go)).

## Schema management

With `create_schema: true`, each signal's exporter runs its DDL in `start`. The client connects to the `default` database, and every statement uses fully qualified names:

| Signal | Statements (all `IF NOT EXISTS`) |
|--------|-----------------------------------|
| every signal | `CREATE DATABASE "<db>" [ON CLUSTER] [ENGINE = …]`. Skipped when the database is `default`. |
| logs | `CREATE TABLE <db>.<logs_table>`. In map mode on ClickHouse 26.2 or newer, attribute and body indexes are `TYPE text(...)`; otherwise `bloom_filter` / `tokenbf_v1`, which the `json` logs table always uses. The exporter detects the server version through the driver. |
| traces | `<traces_table>`; `<traces_table>_trace_id_ts`; materialized view `<traces_table>_trace_id_ts_mv`, which writes min/max `Timestamp` per `TraceId` for each inserted block. |
| metrics | Five tables: `metrics_tables.{gauge,sum,summary,histogram,exponential_histogram}.name`. |
| profiles | `<profiles_table>` ([ClickHouse 26.2+](quirks.md#profiles)). |

The DDL templates are in [`internal/sqltemplates/`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/exporter/clickhouseexporter/internal/sqltemplates). An existing object is [never compared or altered](quirks.md#if-not-exists-never-changes-existing-tables), and a DDL error [stops the Collector](quirks.md#schema-ddl-failure-stops-the-collector).

With `create_schema: false`, the exporter sends no DDL. At start, the logs exporter (and the traces exporter in `json` mode) runs `DESC TABLE` to detect optional columns: `EventName`, and the `*AttributesKeys` columns in JSON mode. Your own tables must keep the column names and types of the exporter's `INSERT` statements. Indexes, partitioning, ordering, TTL, and engine are free. The upstream [schema-management section](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/exporter/clickhouseexporter#schema-management) lists the `ALTER TABLE ... ADD COLUMN` statements for optional columns. Grants each signal needs: [errors → fixes](quirks.md#errors--fixes).

## Clusters and replicated engines

```yaml
exporters:
  clickhouse:
    endpoint: tcp://ch-1:9000,ch-2:9000
    database: otel
    cluster_name: my_cluster
    table_engine:
      name: ReplicatedMergeTree
    # or: database_engine: { name: Replicated, params: "'/clickhouse/databases/otel', '{shard}', '{replica}'" }
```

`cluster_name` adds ``ON CLUSTER `my_cluster` `` to the `CREATE DATABASE` and to each table and view. With `database_engine.name: Replicated` (and a database other than `default`), only `CREATE DATABASE` carries `ON CLUSTER`, because a Replicated database replicates table DDL itself and rejects `ON CLUSTER`. `table_engine.params` is copied verbatim into `ENGINE = <name>(<params>)`.

## Rows per `INSERT` and batching

Each exporter request runs `PrepareBatch("INSERT INTO <db>.<table> (<columns>) VALUES")`, appends every row client-side, and sends once, so one request becomes one `INSERT` per table. The metrics exporter groups points by type and runs up to five `INSERT`s concurrently; their errors are joined.

Request size follows the upstream pipeline unless [`sending_queue.batch`](../otlp_exporter/configuration.md#sending_queue) merges requests before each `INSERT`.

## Async inserts

With `async_insert` on (default), every query carries `async_insert=1`, so the server buffers rows and flushes them itself. In `system.query_log`, the client's statement appears as `query_kind = 'Insert'` and the server flush as `AsyncInsertFlush` (observed). Whether the server acknowledges before or after its flush depends on `wait_for_async_insert`, which the exporter never sets, so the user profile or server value applies. Pass it, and any other async-insert setting, through `connection_params`:

```yaml
exporters:
  clickhouse:
    connection_params:
      wait_for_async_insert: "<0 or 1>"
      async_insert_busy_timeout_ms: "<milliseconds>"
```

For what each setting does, see ClickHouse's [asynchronous inserts guide](https://clickhouse.com/docs/concepts/features/operations/insert/asyncinserts).
