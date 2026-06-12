# `load_balancing`: configuration

All keys live under the exporter instance (e.g. `exporters: { load_balancing: { … } }`). Facts below are traced to the `v0.154.0` contrib source (`config.go`, `factory.go`, `loadbalancer.go`, the per-signal `*_exporter.go` files, and the resolver files).

## Top-level keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `routing_key` | string | empty → see [per-signal table](#routing_key-support-per-signal) | Which value the hash ring is keyed on. |
| `routing_attributes` | list of string | — | Attribute names to build the key from; **only** valid with `routing_key: attributes`. |
| `protocol.otlp` | otlp exporter config | otlp defaults | The per-backend transport (see [protocol.otlp](#protocolotlp)). |
| `resolver` | object | — (required) | Exactly **one** of `static` / `dns` / `k8s` / `aws_cloud_map` (see [Resolvers](#resolvers)). |
| `timeout` | duration | exporter-helper default | Timeout for the load-balancer's **own** queue, before per-backend dispatch. |
| `retry_on_failure` | object | exporter-helper `retry_on_failure` | Retry for the load-balancer level (data bounced back from a sub-exporter). |
| `sending_queue` | object | exporter-helper `sending_queue` | Queue at the load-balancer level, ahead of distribution. |

`timeout`, `retry_on_failure`, and `sending_queue` are the standard exporter-helper knobs — see the `otlp` exporter page for their sub-keys. Here they apply to the **load balancer's own** buffer (the outer level); the **per-backend** queue/retry/timeout are configured separately inside `protocol.otlp` (the inner level). The two levels form the resiliency model described in [advanced.md](advanced.md).

### `routing_key` support per signal

`routing_key` is validated **per signal exporter at startup**, and support differs by signal — this is stricter than the upstream README's prose. The table reflects **v0.154.0**:

| `routing_key` | Traces | Logs | Metrics | Keyed on |
|---------------|:------:|:----:|:-------:|----------|
| `traceID` | ✅ default | ✅ | ❌ | The trace ID. |
| `service` | ✅ | ✅ default | ✅ default | `service.name` resource attribute. |
| `attributes` | ✅ | ✅ | ✅ | The `routing_attributes` values. |
| `resource` | ❌ | ✅ | ✅ | Hash of all resource attributes. |
| `metric` | ❌ | ❌ | ✅ | Metric name. |
| `streamID` | ❌ | ❌ | ✅ | Datapoint stream identity (resource + scope + attributes). |

- **Traces** accept `traceID` (default, also when empty), `service`, `attributes`. Any other value fails startup with `unsupported routing_key: <value>`.
- **Logs** accept `service` (default, also when empty), `traceID`, `resource`, `attributes`. For `traceID`, a log without a trace ID gets a random one (so it still lands somewhere). **Version note:** logs routing arrived in **v0.154.0** (contrib PR #46241); at **v0.152.0–v0.153.0** the log exporter ignored `routing_key` and always routed by trace ID without validating the value. See [quirks.md](quirks.md).
- **Metrics** accept `service` (default, also when empty), `resource`, `metric`, `streamID`, `attributes`. Metrics are **Development** stability.

### `protocol.otlp`

`protocol.otlp` is a full nested **`otlp` exporter** config — TLS, headers, compression, `sending_queue`, `retry_on_failure`, `timeout`, auth, etc. all apply, per backend.

The one rule: **do not set `protocol.otlp.endpoint`.** The exporter overwrites it with each resolved backend address; the factory seeds it to a placeholder. Setting it yourself has no effect (or is misleading).

```yaml
exporters:
  load_balancing:
    routing_key: traceID
    protocol:
      otlp:
        # endpoint: DO NOT SET — populated per backend
        tls:
          insecure: true
        sending_queue:
          enabled: true          # per-backend queue
        retry_on_failure:
          enabled: true          # per-backend retry
    resolver:
      static:
        hostnames:
          - backend-1:4317
          - backend-2:4317
```

## Resolvers

Configure **exactly one**. Zero resolvers fails startup with `no resolvers specified for the exporter`; more than one fails with `only one resolver should be specified`.

### `static`

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `hostnames` | list of string | — | Fixed `host:port` backends. No dynamic updates; the ring only changes on restart. |

### `dns`

Periodically resolves a hostname (e.g. a headless Service) to its A/AAAA records.

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `hostname` | string | — (required) | Hostname to resolve. |
| `port` | string | — (empty) | Port appended to each resolved IP. When unset, the load balancer appends `:4317` to any address that has no port. |
| `interval` | duration | `5s` | Re-resolution interval. |
| `timeout` | duration | `1s` | Per-resolution timeout. |

### `k8s`

Watches a Kubernetes Service's `EndpointSlice`s directly (reacts faster than DNS, no DNS caching).

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `service` | string | — (required) | `name` or `name.namespace`; namespace defaults to `default` when omitted/undeterminable. |
| `ports` | list of int32 | — (empty) | Ports per backend; multiple ports produce multiple backends per pod. When unset, the load balancer appends `:4317` to each address that has no port. |
| `timeout` | duration | `1m` | List/watch timeout. |
| `return_hostnames` | bool | `false` | Return pod hostnames instead of IPs (requires a headless Service backing a StatefulSet). |

**RBAC:** the Collector's ServiceAccount needs `get`, `list`, `watch` on `discovery.k8s.io/v1` `EndpointSlice`.

### `aws_cloud_map`

Discovers backends via AWS Cloud Map service discovery.

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `namespace` | string | — (required) | Cloud Map namespace. |
| `service_name` | string | — (required) | Cloud Map service. |
| `health_status` | string | `HEALTHY` | Filter: `HEALTHY`, `UNHEALTHY`, `ALL`, `HEALTHY_OR_ELSE_ALL`. |
| `interval` | duration | `30s` | Discovery interval. |
| `timeout` | duration | `5s` | Discovery timeout. |
| `port` | uint16 | — | Optional port override for resolved instances. |
| `owner_account` | string | — | Optional AWS account ID for cross-account service discovery. |

AWS region defaults to `us-east-1` via the SDK default config chain. **Limitation:** returns at most 100 hosts — pagination is not implemented.

## Validation summary

| Condition | Error | When |
|-----------|-------|------|
| `routing_key: attributes` with empty `routing_attributes` | `routing_attributes must be specified when routing_key is "attributes"` | config validation |
| `routing_attributes` set with `routing_key` ≠ `attributes` | `routing_attributes can only be used when routing_key is "attributes"; …` | config validation |
| no resolver configured | `no resolvers specified for the exporter` | startup |
| more than one resolver configured | `only one resolver should be specified` | startup |
| unsupported `routing_key` for the signal | `unsupported routing_key: <value>` | startup (all three signals; traces print the value unquoted, logs/metrics quoted) |
