# `clickhouse` exporter: configuration

All keys live under `exporters: { clickhouse: { … } }`. Exporter facts come from contrib **v0.165.1** ([`config.go`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/exporter/clickhouseexporter/config.go), [`factory.go`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/exporter/clickhouseexporter/factory.go), [`internal/sqltemplates/`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/exporter/clickhouseexporter/internal/sqltemplates)). Driver facts come from `clickhouse-go` **v2.47.0**, the version that release pins ([`clickhouse_options.go`](https://github.com/ClickHouse/clickhouse-go/blob/v2.47.0/clickhouse_options.go)). Version notes come from the [contrib v0.165.1 changelog](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/CHANGELOG.md). Values hold for the named releases; check the current release before relying on them.

## Connection keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `endpoint` | string | — (**required**) | DSN URL. `http` selects HTTP. `https` selects HTTP over TLS, and the exporter adds `secure=true`. Any other scheme (`tcp`, `clickhouse`) selects the native protocol. A path (`/db`) names a database. Query parameters go to the driver (see [DSN and settings](#dsn-and-settings)). Several hosts (`tcp://h1:9000,h2:9000`): each connection dials one, chosen by the driver's `connection_open_strategy` (`in_order` default, `round_robin`, `random`). |
| `username` | string | `""` | Precedence per field, highest first: a `username` or `password` query parameter in `endpoint` or `connection_params` (the driver reads them after user info); this key and `password`, when `username` is set (they replace the DSN user info); the DSN user info; the driver default user `default`. |
| `password` | opaque string | `""` | Password for `username` ([ignored alone](quirks.md#silently-ignored-settings)). |
| `database` | string | `default` | Target database for DDL and `INSERT`. A value other than `default` wins over the DSN path. |
| `connection_params` | map[string]string | `{}` | Merged into the DSN query, replacing endpoint parameters with the same name. |
| `compress` | string | `lz4` | `lz4` (also when empty or `true`), `lz4hc`, `zstd`, `none`, `gzip`, `deflate`, `br`. The native protocol accepts only `none`, `lz4`, `lz4hc`, and `zstd`. Ignored when `compress` is already in `endpoint` or `connection_params`. |
| `async_insert` | bool | `true` | Sent as the `async_insert` setting (`1`/`0`) with every query, unless `async_insert` is already in `endpoint` or `connection_params`. |
| `tls` | object | — | [`configtls` client settings](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/config/configtls/README.md); can replace the DSN's TLS setup ([quirks](quirks.md#the-tls-block-replaces-dsn-tls)). |
| `timeout` | duration | `5s` | Per-attempt deadline for one push. The driver derives the server-side `max_execution_time` from it (see [DSN and settings](#dsn-and-settings)). |

## Schema keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `create_schema` | bool | `true` | Runs the database and table DDL at start (see [advanced](advanced.md#schema-management)). |
| `logs_table_name` | string | `otel_logs` | Logs table. |
| `traces_table_name` | string | `otel_traces` | Spans table. Schema creation also creates `<name>_trace_id_ts` and the materialized view `<name>_trace_id_ts_mv`. |
| `profiles_table_name` | string | `otel_profiles` | Profiles table, one row per OTLP `Sample`. |
| `metrics_tables.<type>.name` | string | `otel_metrics_<type>` | `<type>` is `gauge`, `sum`, `summary`, `histogram`, or `exponential_histogram`. The exponential default is `otel_metrics_exponential_histogram` (`config.go`); the v0.165.1 upstream README's `otel_metrics_exp_histogram` is wrong. |
| `metrics_table_name` | string | `""` | **Deprecated**, and [has no effect](quirks.md#silently-ignored-settings). Use `metrics_tables`. |
| `ttl` | duration | `0` (none) | Adds `TTL <time column> + toInterval<Unit>(n)`, using the largest whole unit (day, hour, minute, second). Like every schema key, it affects only [tables created afterwards](quirks.md#if-not-exists-never-changes-existing-tables). |
| `table_engine.name` / `.params` | string | `MergeTree` / `""` | Renders `ENGINE = <name>(<params>)`. When `name` is empty, `params` is ignored. |
| `database_engine.name` / `.params` | string | `""` (server default) | Renders `ENGINE = <name>` or `ENGINE = <name>(<params>)` on `CREATE DATABASE`. Not used for the `default` database. |
| `cluster_name` | string | `""` | Adds ``ON CLUSTER `<name>` `` to the database and table DDL. Table DDL omits it when `database_engine.name` is `Replicated` and the database is not `default`. |
| `json` | bool | `false` | Logs and traces tables and `INSERT`s use `JSON` attribute columns instead of `Map(LowCardinality(String), String)`. Metrics and profiles are unaffected. Add `enable_json_type=1` to `connection_params` if the server requires it; the upstream README recommends ClickHouse v25 or newer. Named instances (`clickhouse/json`, `clickhouse/map`) mix modes across pipelines. The `clickhouse.json` feature gate was removed in v0.153.0. |

## Queue and retry

`sending_queue` and `retry_on_failure` keep the core v0.165.0 defaults (`NewDefaultQueueConfig`, `NewDefaultBackOffConfig`); keys and validation are in [`sending_queue`](../otlp_exporter/configuration.md#sending_queue) and [`retry_on_failure`](../otlp_exporter/configuration.md#retry_on_failure).

**Connection pool.** Since v0.165.0, with the queue enabled, `max_open_conns` defaults to `num_consumers`, times 5 for the metrics exporter because it inserts its five metric types concurrently, and `max_idle_conns` to the same value, capped at the final `max_open_conns`. A positive value in `endpoint` or `connection_params` wins; zero or a negative value gets the default without the metrics ×5. So 10 consumers with `max_open_conns=3` cap metrics at 3 open and 3 idle connections. With the queue disabled, unset or non-positive values get the driver defaults: 5 idle, and idle + 5 open. Each signal's exporter [instance](../../SKILL.md#named-instances) opens its own pool.

## DSN and settings

At validation and start, the exporter merges `endpoint` and the keys above into one DSN ([`buildDSN`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/exporter/clickhouseexporter/config.go)), with the precedence each key row gives, and appends `otelcol/<collector version>` to `client_info_product`. ClickHouse records that as `client_name` (native) or `http_user_agent` (HTTP) in `system.query_log`.

`clickhouse-go` v2.47.0 consumes its own options from the query string ([DSN reference](https://github.com/ClickHouse/clickhouse-go/blob/v2.47.0/README.md#dsn)). **Every other parameter becomes a ClickHouse setting sent with every query**: in the query packet on the native protocol, and as URL parameters on HTTP ([value conversion](quirks.md#setting-values-are-lowercased)). Each setting is described in ClickHouse's [settings reference](https://clickhouse.com/docs/reference/settings/session-settings).

When a push context has more than 1s left, the driver also sends `max_execution_time` = remaining seconds + 5 ([`context.go`](https://github.com/ClickHouse/clickhouse-go/blob/v2.47.0/context.go)).

Validation, DSN, and runtime errors with fixes: [quirks](quirks.md#errors--fixes).
