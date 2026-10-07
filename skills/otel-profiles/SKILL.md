---
name: otel-profiles
description: OpenTelemetry profiles signal and the eBPF profiler (otelcol-ebpf-profiler, `profiling` receiver). Use when setting up continuous profiling with OpenTelemetry, deploying the eBPF profiler on Linux hosts, Docker, or Kubernetes (DaemonSet, kind, k3d), building a profiles pipeline, sending OTLP profiles to Pyroscope or another backend, getting flame graphs from OTel, enabling off-CPU profiling, or debugging profiler errors about tracefs, PID namespaces, missing pods, or the service.profilesSupport feature gate. Not for SDK tracing or metrics setup.
---

# OpenTelemetry Profiles

Profiles are the fourth OpenTelemetry signal: sampled stack traces with resource and sample attributes. The main OpenTelemetry producer is the [eBPF profiler](https://github.com/open-telemetry/opentelemetry-ebpf-profiler), shipped as the `profiling` receiver in the [`otelcol-ebpf-profiler`](https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-ebpf-profiler) Collector distribution. It profiles every process on a Linux host with no application changes.

Maturity: the signal entered [public Alpha](https://opentelemetry.io/blog/2026/profiles-alpha/) in March 2026, the OTLP profiles proto (`v1development`) is Development, the `profiling` receiver is Development, and the Collector gate `service.profilesSupport` is alpha. Expect breaking changes; pin the profiler, Collector, and backend versions together.

Collector-side component facts (OTLP receiver/exporter profile support, the fixed `/v1development/profiles` HTTP path, `k8s_attributes`, `memory_limiter`, `filter` `profile_conditions`) live in the `otel-collector` skill's component pages; OTTL `profile`/`profilesample` contexts live in `otel-ottl`.

`otelcol-ebpf-profiler` `0.162.0` bundles only: receivers `profiling`, `nop`; processors `memory_limiter`, `k8s_attributes`, `resource_detection`, `resource`, `transform` (plus `batch`, which does not support profiles); exporters `otlp_grpc`, `otlp_http`, `debug`, `file`, `nop`; extensions `health_check`, `pprof`, `opamp`. For anything else (`filter`, vendor exporters), relay to a regular Collector or build a distribution (`otel-collector-builder` skill).

## Version matrix (verified 2026-10-07)

| Artifact | Version | What it means |
|---|---|---|
| `otel/opentelemetry-collector-ebpf-profiler` | `0.162.0` (latest release, 2026-09-29) | Pins ebpf-profiler `v0.0.202636`: boolean `pid_namespace_translation`; does not bundle the `offcpu` extension. |
| Same image, `0.163.0-nightly.*` | prerelease | Bundles `offcpu` from `0.163.0-nightly.3d037de` (2026-10-05); pins ebpf-profiler `v0.0.202640`, with `pid_namespace_translation_mode` ([#1801](https://github.com/open-telemetry/opentelemetry-ebpf-profiler/pull/1801)), from `0.163.0-nightly.2e6f5c3` (2026-10-07). Expected in release `0.163.0`, unreleased at verification time. |
| `profiling` receiver | Development | Linux `amd64`/`arm64` only, kernel 5.10+ (checked at config validation; `no_kernel_version_check` bypasses it). |
| `service.profilesSupport` gate | alpha | Required for any profiles pipeline. |

Receiver keys follow the ebpf-profiler version the distribution pins, not the Collector version: [references/receiver.md](references/receiver.md) has the key table, how to look up the pin for another release, what the receiver emits, and off-CPU.

## Workflow

1. **Pick the image and pin.** Use the latest release unless a needed key or component is missing from it (descendant PID namespace translation and the bundled `offcpu` extension are not in `0.162.0`). Alternatives: a pinned nightly, labeled as a prerelease with the release that replaces it, or an OCB build (`otel-collector-builder` skill) based on the [distribution manifest](https://github.com/open-telemetry/opentelemetry-collector-releases/blob/main/distributions/otelcol-ebpf-profiler/manifest.yaml).
2. **Pick the environment.**
   - Linux host or plain Docker: [Run on a host](#run-on-a-host) below.
   - Kubernetes: [references/kubernetes.md](references/kubernetes.md) (DaemonSet, RBAC, Kubernetes metadata, memory sizing).
   - Kubernetes nodes that are themselves containers (kind, k3d, Docker-in-Docker, minikube with the docker driver): also [references/nested-containers.md](references/nested-containers.md).
3. **Wire the pipeline.** `profiles` pipeline: `profiling` receiver, `memory_limiter` first, enrichment processors, then `otlp_grpc` or `otlp_http`. Start the Collector with `--feature-gates=+service.profilesSupport`. Run `validate --config=<file> --feature-gates=+service.profilesSupport` with the target image; it catches wrong keys for the pinned version. Receiver keys and off-CPU: [references/receiver.md](references/receiver.md).
4. **Export and verify.** Add a `debug` exporter first, then the backend: [references/backends.md](references/backends.md).
5. **Troubleshoot** by exact error string: [references/troubleshooting.md](references/troubleshooting.md).

## Run on a host

The profiler needs root or `CAP_SYS_ADMIN`/`CAP_BPF`/`CAP_PERFMON`, the host PID namespace, and a mounted tracefs. A container gets a fresh `/sys` without tracefs even when privileged, so bind-mount it:

```bash
docker run -d --name otel-ebpf-profiler --privileged --pid=host \
  -v /sys/kernel/tracing:/sys/kernel/tracing \
  -v "$PWD/config.yaml:/etc/otelcol/config.yaml:ro" \
  otel/opentelemetry-collector-ebpf-profiler:0.162.0 \
  --config=/etc/otelcol/config.yaml --feature-gates=+service.profilesSupport
```

```yaml
receivers:
  profiling:
    samples_per_second: 97      # default 20
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 800
    spike_limit_mib: 200
exporters:
  debug:
    verbosity: basic
  otlp_grpc:
    endpoint: profiles-backend:4317   # backend-specific, see references/backends.md
    tls:
      insecure: true
service:
  pipelines:
    profiles:
      receivers: [profiling]
      processors: [memory_limiter]
      exporters: [otlp_grpc, debug]
```

The image is built `FROM scratch`: no shell, so debug from logs, not `exec`. Expected startup lines: `eBPF tracer loaded`, `Attached sched monitor`, `Everything is ready`.

## Known limitations

- Linux only; the backend side may lag the proto, so check backend docs for the OTLP profiles version it accepts.
- Trace-to-profile correlation is partial: OTLP profiles can link samples to `trace_id`/`span_id`, but the profiler fills them only from a context producer (OBI via `obi_process_ctx`; the [thread-context OTEP](https://github.com/open-telemetry/opentelemetry-specification/pull/4947) implementation is in progress). Go pprof labels are read as custom labels.
- Symbolization of native code without symbols on the host may be incomplete in some backends.
