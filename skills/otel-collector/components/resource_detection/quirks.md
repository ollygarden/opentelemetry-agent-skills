# `resource_detection` — known quirks

## Fatal detector errors can stop startup

Fatal errors from a configured detector **propagate and can prevent the Collector from starting** after the configured retry budget is exhausted. Every detector uses the global `retry` block; retries are enabled by default, with the default session bounded by `timeout: 5s`. For supported network metadata detectors, an unreachable service is fatal when `fail_on_missing_metadata` is enabled; otherwise it produces an empty resource. Only configure detectors for the environment you are in.

Since v0.158.0, the top-level `fail_on_missing_metadata` flag (default `false`) controls supported network metadata detectors: enable it when an unreachable metadata service should be a hard error that participates in processor retry rather than returning an empty resource. The older per-detector fields on `ec2`, `upcloud`, `vultr`, `nova`, `alibaba_ecs`, and `tencent_cvm` are deprecated; migrate to the top-level key. Since v0.163.0 the flag also covers the `ec2` detector's instance identity document and hostname lookups. The `oraclecloud` detector uses a fast probe: a failed probe returns an empty resource; if the probe succeeds but the follow-up fetch fails, it errors only when `fail_on_missing_metadata: true` and otherwise returns an empty resource.

## `override: true` is the default — and it overwrites SDK-set attributes

By default detected attributes **overwrite** resource attributes already on incoming telemetry. If your SDKs already set `service.name`/`host.name` and the Collector then detects different values for its own host, you can silently replace good per-service identity with the Collector's host identity. For Collectors that receive pre-enriched telemetry (gateway/aggregation tier), prefer `override: false`. The `dynatrace` detector documentation explicitly recommends `override: false` for exactly this reason. (Verified on contrib 0.159.0: with a colliding `service.name`, `override: true` took the detected value and `override: false` kept the incoming one.)

## It describes the Collector's host — not the workload that emitted the telemetry

Every detector reports on the environment **the Collector itself runs in**. The k8s-flavored detectors here (`k8s_api`, `kubeadm`, `openshift`, `eks`/`aks` cluster name) describe the node/cluster the Collector runs on, not the pod that produced each record. For per-record pod/namespace/workload metadata, use [`k8s_attributes`](../k8s_attributes/README.md) instead — the two are complementary, not interchangeable.

## Detector ordering vs `override` are two different things

- **Order** within `detectors: [...]` decides which *detector* wins when two detectors emit the same attribute (first wins).
- **`override`** decides whether *detected* values beat values *already on incoming telemetry*.

They are independent; mixing them up leads to surprising precedence. See [advanced](advanced.md) for the recommended AWS ordering.

## `refresh_interval` multiplies metric cardinality

Changing a resource attribute creates a **new metric time series**. With `refresh_interval` set, every refresh that picks up a changed attribute forks new series — which can sharply raise backend storage cost and query cost. Each refresh also re-runs all detectors, so short intervals add CPU/memory load. Leave it at the default (`0`, detect once) unless attributes genuinely change at runtime; intervals below 1 minute are strongly discouraged.

## Internal telemetry added in v0.157.0

The processor now emits three enabled-by-default, **Development**-stability metrics:

| Metric | Type / unit | Attributes | Meaning |
|--------|-------------|------------|---------|
| `otelcol.resourcedetection.attributes.detected` | asynchronous non-monotonic sum / `{attribute}` | — | Number of attributes in the currently detected resource. |
| `otelcol.resourcedetection.detector.duration` | histogram / `s` | `detector`, `outcome` | Per-detector runtime, including retry backoff. |
| `otelcol.resourcedetection.detector.results` | monotonic sum / `{detection}` | `detector`, `outcome`, `error.type` on failure | Detection outcomes by detector; useful for identifying failing metadata lookups. |

The names and attribute sets may still change because all three metrics are Development stability.

## Docker / container caveats

- Use the **`docker`** detector instead of `system` when the Collector runs as a Docker container, and mount the Docker socket (`/var/run/docker.sock`). Note the official images run as non-root since 0.40.0, so socket access needs group permissions.
- The `docker` detector **does not work on macOS**.
- `host.name` from the `system` detector defaults to the FQDN (`hostname_sources: ["dns", "os"]`); inside a container this is often the container ID. Set `hostname_sources: ["os"]` to use the OS-provided hostname only.

## Detectors can be compiled out of custom builds

Since v0.163.0 a custom build can compile detectors out (mechanism: the `otel-collector-builder` skill's `dist.resource.detection.detectors` manifest key). An omitted detector's constructor returns no detector, so such a Collector must only list compiled-in detectors under `detectors:`. If a detector fails only in a custom build, check the builder config before the processor config. Upstream distributions are unaffected.

## The `resourcedetection` type is a deprecated alias

The component type was renamed `resourcedetection` → `resource_detection` (with an underscore) in **v0.153.0**. The old `resourcedetection` name still works as a deprecated alias and logs a deprecation warning at startup; new configs should use `resource_detection`. The Go module path and internal package are still `resourcedetectionprocessor`.

## Stability is per-signal

Beta for traces, metrics, and logs; **Development** for profiles. Treat profiles support as experimental.
