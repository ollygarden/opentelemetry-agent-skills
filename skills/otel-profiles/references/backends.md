# Exporting profiles to a backend

Profiles leave the Collector over OTLP: `otlp_grpc`, or `otlp_http`, which posts to `<endpoint>/v1development/profiles` unless `profiles_endpoint` overrides it. To relay through a regular Collector, its `otlp` receiver needs the same feature gate; see the `otel-collector` skill's `components/otlp/` pages.

Backend support moves fast. Status below was checked on 2026-10-07; re-check the linked docs before relying on it.

| Backend | OTLP profiles ingestion | Status and source |
|---|---|---|
| Grafana Pyroscope (OSS) | gRPC and HTTP on the HTTP port `4040` | Pyroscope docs call it suitable for development and testing ([docs](https://grafana.com/docs/pyroscope/latest/configure-client/opentelemetry/ebpf-profiler/)). Routes verified in `v2.3.1` source. |
| Elastic (EDOT Collector) | `profiling` receiver plus the `elasticsearch` exporter | Preview from Elastic Stack 9.2, behind `service.profilesSupport` ([docs](https://www.elastic.co/docs/reference/edot-collector/config/configure-profiles-collection)). |
| devfiler | OTLP | Desktop app for development only, not a production backend ([repo](https://github.com/elastic/devfiler)). |
| Datadog Full-Host Profiler | Not documented as accepting upstream OTLP profiles | Preview; a standalone executable built on the OpenTelemetry eBPF profiler that sends through the Datadog Agent ([docs](https://docs.datadoghq.com/profiler/enabling/full_host/)). |
| Grafana Cloud Profiles | gRPC on the stack's Profiles host, port `443` | Verified 2026-10-08 with the `0.162.0` image. OTLP/HTTP authenticates but rejects the profiler's data; see [Grafana Cloud Profiles](#grafana-cloud-profiles). Not in Grafana Cloud's OTLP gateway docs, which list traces, metrics, and logs only. |

## Pyroscope

```yaml
exporters:
  otlp_grpc:
    endpoint: pyroscope.<namespace>.svc:4040   # no separate 4317 port
    tls:
      insecure: true
```

- The Helm chart (`grafana/pyroscope`) enables an Alloy subchart by default; set `alloy.enabled: false` if profiles should arrive only through OpenTelemetry.
- `service_name` comes from `service.name`. Without it, Pyroscope `v2.3.1` uses `unknown_service:<process.executable.name>`; the Pyroscope docs also show an `ingestion_relabeling_rules` `labelmap` from `process.executable.name` to `service_name`.
- Label sanitization is off by default since Pyroscope 2.0 (`-validation.disable-label-sanitization=true`), so dotted OTel attribute names are stored as-is (`k8s.namespace.name`, not `k8s_namespace_name`). The `LabelNames` API lists only Prometheus-style names, which makes the `k8s.*` labels look missing, but `LabelValues` and selectors with quoted dotted names work.
- CPU samples appear as profile type `process_cpu:cpu:nanoseconds:cpu:nanoseconds`.

Query without a UI (port-forward or reach Pyroscope on `4040`; times in milliseconds):

```bash
end=$(($(date +%s)*1000)); start=$((end-300000))
# Which services arrived
curl -s -X POST http://localhost:4040/querier.v1.QuerierService/LabelValues \
  -H 'content-type: application/json' \
  -d "{\"name\":\"service_name\",\"start\":$start,\"end\":$end}"
# CPU per service in one namespace, using a dotted label name
curl -s -X POST http://localhost:4040/querier.v1.QuerierService/SelectSeries \
  -H 'content-type: application/json' \
  -d "{\"profileTypeID\":\"process_cpu:cpu:nanoseconds:cpu:nanoseconds\",
       \"labelSelector\":\"{\\\"k8s.namespace.name\\\"=\\\"default\\\"}\",
       \"groupBy\":[\"service_name\"],\"start\":$start,\"end\":$end,\"step\":300}"
```

In Grafana, add a `grafana-pyroscope-datasource` pointing at `http://<pyroscope>:4040` and browse flame graphs in Drilldown > Profiles.

## Grafana Cloud Profiles

Use `otlp_grpc`. The stack's Profiles details in the Cloud portal give the host (`profiles-prod-NNN.grafana.net`) and the numeric user (instance) ID; create a Cloud Access Policy token with `profiles:write` there. `otelcol-ebpf-profiler` has no `basicauth` extension, so send a precomputed header:

```yaml
exporters:
  otlp_grpc:
    endpoint: profiles-prod-NNN.grafana.net:443
    headers:
      authorization: "Basic ${env:GRAFANA_PROFILES_BASIC_AUTH}"   # base64 of <user-id>:<token>
```

- Stacks symbolize and resource attributes are kept.
- OTLP/HTTP (`https://profiles-prod-NNN.grafana.net/v1development/profiles`) authenticates, but with Collector `0.162.0` every batch fails with HTTP 400 `invalid labels '{="<pid>", ...}' with error: invalid label name ''`. Grafana Cloud's HTTP decoder turns the integer resource attribute `process.pid` into an empty-named label and drops string resource attributes. Deleting `process.pid` gets a 200, but all resource attributes are lost (`service_name="unknown_service"`), so it is not a fix.
- Only `service.name`, `service.namespace`, `service.version`, and `deployment.environment.name` become underscore labels (`service_namespace`, `deployment_environment_name`, ...); every other attribute (`k8s.*`, `container.id`, `process.*`, `thread.name`) keeps its dotted name and is queried quoted, as in self-hosted Pyroscope. Self-hosted Pyroscope `v2.3.1` keeps all four dotted too. Observed 2026-10-10.
- Without `service.name`, `service_name` is `unknown_service:<process.executable.name>`, as in Pyroscope.
