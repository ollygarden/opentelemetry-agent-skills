# `prometheus`: known quirks

## No auto-sharding — duplicate scrapes across replicas

The receiver is **stateful** and the Collector **cannot auto-scale scraping** across replicas. If
you run multiple collector replicas with the **same** config, each replica scrapes every target, so
you get **duplicate metrics**. Options:

- Use the **Target Allocator** (`target_allocator`) to shard targets across replicas — see [advanced.md](advanced.md).
- Or **manually shard**: give each replica a different scrape configuration.

This is the upstream README's headline warning; treat the receiver as work-in-progress for
horizontal scaling.

## `$` must be escaped as `$$`

The Collector interprets `$` in config as an environment-variable reference. To use a **literal**
`$` inside the Prometheus config — common in relabel regexes and `replacement` values — escape it as
`$$`. An unescaped `$foo` will be substituted (or blanked) at config load, silently breaking the
relabel rule.

## `collector_id: collector-${HOSTNAME}` — expanded, not rejected

You may see examples using a templated `collector_id` like `collector-${HOSTNAME}` for the Target
Allocator. Contrary to what the "must not contain `${`" rule suggests, the Collector's own
environment-variable substitution expands `${HOSTNAME}` **before** the receiver validates the
config, so `collector-${HOSTNAME}` becomes `collector-<hostname>` and validates fine — this is
actually a working way to get per-pod uniqueness when `HOSTNAME` is set (as it is in Kubernetes
pods). If `HOSTNAME` is unset the value silently collapses to `collector-`.

The `CollectorID is not a valid ID` validation error fires only when a **literal** `${` reaches the
receiver — i.e. when you escape it as `$${...}` (the same `$$` escaping used for literal `$` in
relabel rules) so it survives env-substitution. So the real trap is over-escaping: `$${HOSTNAME}`
is rejected, plain `${HOSTNAME}` is not. An empty `collector_id` is also rejected.

## `use_start_time_metric` / `start_time_metric_regex` were removed

These two keys used to live on this receiver but were **removed in v0.143.0** and are gone from the
current config. To adjust metric start times from `process_start_time_seconds`, use the
**`metric_start_time`** processor (renamed from `metricstarttime` in v0.148.0; the old name is kept
as a deprecated alias). Putting `use_start_time_metric` in the receiver config now is an unknown-key error.

## Prometheus server features are rejected

This is a *scrape-only* receiver, not a Prometheus server. These sections under `config:` are
rejected with `unsupported features: …`:

- `remote_write`
- `remote_read`
- `rule_files`
- `alert_config.relabel_configs`
- `alert_config.alertmanagers`

For Remote Write ingestion use the `prometheusremotewrite` receiver; for rules/alerting run an
actual Prometheus or use OTel processors/connectors.

## `fallback_scrape_protocol` defaulting (Prometheus 3.0)

Since Prometheus 3.0, scrapes without a proper `Content-Type` header fail. The receiver
auto-defaults each scrape config's `fallback_scrape_protocol` to `PrometheusText0.0.4` when unset,
preserving the older lenient behavior. If you scrape native histograms you must still add
`PrometheusProto` to `scrape_protocols` explicitly — the fallback does not cover that.

## Validation-error → fix

| Error | Cause | Fix |
|-------|-------|-----|
| `no Prometheus scrape_configs or target_allocator set` | Neither inline `scrape_configs`, nor `scrape_config_files`, nor `target_allocator` provided. | Add at least one scrape source. |
| `unsupported features: …` | A Prometheus server-only section is present under `config:`. | Remove `remote_write`/`remote_read`/`rule_files`/`alert_config.*`. |
| `TargetAllocator endpoint is not valid: …` | `target_allocator.endpoint` is not a valid request URI. | Use a full URL like `http://my-ta-service`. |
| `CollectorID is not a valid ID` | `collector_id` is empty, or a **literal** `${` reached the receiver (e.g. escaped as `$${`). Plain `${ENV}` is env-expanded before validation and does not trigger this. | Use a literal id (e.g. `collector-1`), or a `${ENV}` that expands to a valid id. |
| `interval must be a positive duration, got …` | `target_allocator.interval` is zero/negative/unset. | Set a positive duration (e.g. `30s`). |
| Empty/blanked relabel replacement | Unescaped `$` was treated as an env var. | Escape literal `$` as `$$`. |

## Stability

Metrics are **Beta**. The receiver embeds `prometheus/prometheus`'s scrape manager, so Prometheus
scrape behavior tracks the pinned upstream version. `trim_metric_suffixes` is flagged
**Experimental** upstream.
