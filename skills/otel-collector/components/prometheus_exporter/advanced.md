# `prometheus` exporter: advanced use-cases

## Resource attributes: `target_info` vs `resource_constant_labels`

By default, resource attributes are **not** copied onto each metric's labels — they are exposed on a separate `target_info` series, following Prometheus convention. To use them in PromQL you join on the standard identity labels:

```promql
my_metric * on (job, instance) group_left target_info
```

Two ways to bring the attributes onto the metric labels directly instead:

- **Copy the few you need** with the [`transform`](../transform/README.md) processor (datapoint context), so only the relevant attributes become labels and cardinality stays bounded:

  ```yaml
  processors:
    transform:
      metric_statements:
        - context: datapoint
          statements:
            - set(attributes["namespace"], resource.attributes["k8s.namespace.name"])
  ```

- **Select resource labels** with wildcard `included` / `excluded` patterns:

  ```yaml
  exporters:
    prometheus:
      endpoint: 0.0.0.0:9464
      resource_constant_labels:
        included: ["k8s.*", "deployment.environment.name"]
        excluded: ["k8s.pod.uid"]
  ```

The older `resource_to_telemetry_conversion` block is deprecated in v0.160.0 and rejected by default in v0.166.0 (see [configuration.md](configuration.md#feature-gate)).

## OpenMetrics and exemplars

Exemplars are exported **only** in OpenMetrics format, and **only** for histograms and monotonic sums (counters). Enable OpenMetrics to get them:

```yaml
exporters:
  prometheus:
    endpoint: 0.0.0.0:9464
    enable_open_metrics: true
```

There is no exemplar support for native histograms (below).

## Native histograms

OTel exponential histograms are automatically converted to the Prometheus **native histogram** format. To scrape them, the Prometheus server must scrape using the protobuf format and accept native histograms. (No exemplars are emitted for native histograms.)

## Choosing a `translation_strategy`

`translation_strategy` replaces the deprecated, ignored `add_metric_suffixes` (see [configuration.md](configuration.md#translation_strategy)). Pick by what your consumer expects:

```yaml
exporters:
  prometheus:
    endpoint: 0.0.0.0:9464
    translation_strategy: UnderscoreEscapingWithSuffixes   # classic Prometheus compatibility
```

- `UnderscoreEscapingWithSuffixes` / `UnderscoreEscapingWithoutSuffixes` — escape names to classic Prometheus form, with or without type/unit suffixes.
- `NoUTF8EscapingWithSuffixes` — keep UTF-8 names, still append suffixes.
- `NoTranslation` — pass names/labels through unaltered.

Note the content-negotiation caveat: Prometheus may still apply underscore escaping on the wire even when you keep UTF-8. The `exporter.prometheusexporter.DisableAddMetricSuffixes` feature gate (stable in v0.166.0, so always on) forces `translation_strategy` to always be used and the deprecated `add_metric_suffixes` to be ignored.

## The `sending_queue` caveat

This exporter accumulates in memory and is scraped, so the standard `exporterhelper` `sending_queue` has far less meaning here than for a pushing exporter — there is no downstream send to queue against. Don't over-tune it. If you do need the full queue reference, it is documented on the [`otlp_grpc` exporter](../otlp_exporter/configuration.md#sending_queue).
