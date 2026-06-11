# `interval`: advanced use-cases

## Keep gauges spiky, smooth everything else

```yaml
processors:
  interval:
    interval: 30s
    pass_through:
      gauge: true
      summary: true
```

Use this when your gauges represent things you can't afford to flatten — request latency, queue depth, current connections.

## Multiple intervals for different pipelines

```yaml
processors:
  interval/fast:
    interval: 15s
  interval/slow:
    interval: 5m

service:
  pipelines:
    metrics/realtime:
      receivers: [otlp]
      processors: [memory_limiter, interval/fast]
      exporters: [otlp/dashboard]
    metrics/longterm:
      receivers: [otlp]
      processors: [memory_limiter, interval/slow]
      exporters: [otlp/warehouse]
```
