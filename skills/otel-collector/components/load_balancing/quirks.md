# `load_balancing`: known quirks

## Do not set `protocol.otlp.endpoint`

The exporter overwrites `endpoint` with each resolved backend address — the factory seeds it to a placeholder (`placeholder:4317`). Any value you put there is ignored. Configure everything else (`tls`, `headers`, `compression`, `sending_queue`, `retry_on_failure`, auth) under `protocol.otlp`; leave `endpoint` out.

## Routing-key support is signal-specific (and partly version-specific)

The code is stricter than the upstream README's prose, and logs changed behavior across releases:

- **Logs gained `routing_key` in v0.154.0** (contrib PR #46241), accepting `service` (default), `traceID`, `resource`, `attributes`. At **v0.152.0–v0.153.0** the log exporter **ignored** `routing_key` — it always routed by trace ID and did **not** validate the value, so a `routing_key` set on a logs pipeline was silently inert. If you run an older build, don't assume `service`/`resource` log routing works.
- **Traces** accept only `traceID` (default), `service`, `attributes`. `resource`, `metric`, `streamID` fail startup with `unsupported routing_key: <value>`.
- **Metrics** accept `service` (default), `resource`, `metric`, `streamID`, `attributes`, and are **Development** stability — treat metric load balancing as experimental.

Always confirm support against the exact contrib version you run; this surface has been changing.

## Exactly one resolver

`resolver` must contain **one** of `static` / `dns` / `k8s` / `aws_cloud_map`. Zero → `no resolvers specified for the exporter`; two or more → `only one resolver should be specified`. Both are exporter-build errors rather than `Config.Validate()` errors, but `validate` builds the pipeline, so it does surface them (verified on v0.156.0).

## `routing_attributes` is coupled to `routing_key: attributes`

`routing_attributes` is rejected at config validation unless `routing_key` is `attributes`, and `routing_key: attributes` is rejected unless `routing_attributes` is non-empty. They travel together.

## Static resolver + all backends down = unavoidable loss

A `static` resolver never changes its list at runtime. If every listed backend is unreachable, there is nowhere to re-route and data is dropped once the queues drain. Dynamic resolvers (`dns`/`k8s`/`aws_cloud_map`) eventually reflect topology changes, but a mismatch window still exists between a backend dying and the resolver noticing.

## DNS caching delays rebalancing

The `dns` resolver is only as fresh as the DNS responses it gets; intermediate caches/TTLs can delay a backend list update beyond `interval`. The `k8s` resolver (EndpointSlice watch) avoids this and reacts faster to pod churn.

## Hash distribution, not load awareness

The ring balances **keys**, not bytes or backend CPU. It assumes routing-key values are reasonably uniform; a few very hot trace IDs or services can skew load even though key counts are even. The exporter does not look at actual backend load.

## `return_hostnames` needs a headless StatefulSet

`k8s` `return_hostnames: true` only yields stable, resolvable names when the Service is headless and backs a StatefulSet (stable pod hostnames). With a normal Deployment, use IPs (the default).

## Rebalancing churn

On a backend-set change, roughly `R/N` routes (R = distinct routing-key values, N = backends) move to a different backend. More backends means proportionally less disruption per change — but every move briefly breaks affinity for those keys (e.g. a trace mid-flight can split across two sampling instances). Pairing with `groupbytrace` upstream can tighten per-trace atomicity.

## Single point of failure

One LB Collector in front of the fleet is itself a SPOF. Run several identical LB front-ends (same `resolver`, same `routing_key`) behind an L4 balancer — the deterministic ring means a key still maps to the same backend regardless of which front-end handles it. See [advanced.md](advanced.md).
