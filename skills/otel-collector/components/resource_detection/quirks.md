# `resource_detection` — known quirks

## A failed detector stops the Collector from starting

If a configured detector fails during detection, the error **propagates and prevents the Collector from starting** — it is not a warning you can ignore. This bites when you configure a cloud detector for a platform the Collector is not actually running on, or when a metadata service is unreachable. Only configure detectors for the environment you are in.

Some metadata-service detectors soften this with a `fail_on_missing_metadata` flag (default `false`): when the endpoint is unavailable they log an error and return an empty resource instead of failing. Detectors that expose it include `ec2`, `upcloud`, `vultr`, `nova`, `alibaba_ecs`, and `tencent_cvm`. The `oraclecloud` detector uses a fast-probe approach instead — it returns an empty resource (no error) if the IMDS probe fails, and only errors if the probe succeeds but the follow-up fetch fails.

## `override: true` is the default — and it overwrites SDK-set attributes

By default detected attributes **overwrite** resource attributes already on incoming telemetry. If your SDKs already set `service.name`/`host.name` and the Collector then detects different values for its own host, you can silently replace good per-service identity with the Collector's host identity. For Collectors that receive pre-enriched telemetry (gateway/aggregation tier), prefer `override: false`. The `dynatrace` detector documentation explicitly recommends `override: false` for exactly this reason. (Verified on contrib 0.154.0: with a colliding `service.name`, `override: true` took the detected value and `override: false` kept the incoming one.)

## It describes the Collector's host — not the workload that emitted the telemetry

Every detector reports on the environment **the Collector itself runs in**. The k8s-flavored detectors here (`k8s_api`, `kubeadm`, `openshift`, `eks`/`aks` cluster name) describe the node/cluster the Collector runs on, not the pod that produced each record. For per-record pod/namespace/workload metadata, use [`k8s_attributes`](../k8s_attributes/README.md) instead — the two are complementary, not interchangeable.

## Detector ordering vs `override` are two different things

- **Order** within `detectors: [...]` decides which *detector* wins when two detectors emit the same attribute (first wins).
- **`override`** decides whether *detected* values beat values *already on incoming telemetry*.

They are independent; mixing them up leads to surprising precedence. See [advanced](advanced.md) for the recommended AWS ordering.

## `refresh_interval` multiplies metric cardinality

Changing a resource attribute creates a **new metric time series**. With `refresh_interval` set, every refresh that picks up a changed attribute forks new series — which can sharply raise backend storage cost and query cost. Each refresh also re-runs all detectors, so short intervals add CPU/memory load. Leave it at the default (`0`, detect once) unless attributes genuinely change at runtime; intervals below 1 minute are strongly discouraged.

## Docker / container caveats

- Use the **`docker`** detector instead of `system` when the Collector runs as a Docker container, and mount the Docker socket (`/var/run/docker.sock`). Note the official images run as non-root since 0.40.0, so socket access needs group permissions.
- The `docker` detector **does not work on macOS**.
- `host.name` from the `system` detector defaults to the FQDN (`hostname_sources: ["dns", "os"]`); inside a container this is often the container ID. Set `hostname_sources: ["os"]` to use the OS-provided hostname only.

## The `resourcedetection` type is a deprecated alias

The component type was renamed `resourcedetection` → `resource_detection` (with an underscore) in **v0.153.0**. The old `resourcedetection` name still works as a deprecated alias and logs a deprecation warning at startup; new configs should use `resource_detection`. The Go module path and internal package are still `resourcedetectionprocessor`.

## Stability is per-signal

Beta for traces, metrics, and logs; **Development** for profiles. Treat profiles support as experimental.
