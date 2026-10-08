# `profiling` receiver

Keys depend on the ebpf-profiler version the distribution pins, not on the Collector version. For any release other than those in the SKILL.md version matrix, read the pin and the config struct at that tag, then run `validate` with the target image:

```bash
gh api 'repos/open-telemetry/opentelemetry-collector-releases/contents/distributions/otelcol-ebpf-profiler/manifest.yaml?ref=v<collector-version>' --jq .content | base64 -d | grep -A1 ebpf-profiler
# then: https://github.com/open-telemetry/opentelemetry-ebpf-profiler/blob/<pin>/collector/config/config_linux.go
```

## Keys (ebpf-profiler `v0.0.202640`; identical on `v0.0.202636` except the translation key)

Source of truth: [`collector/config/config_linux.go`](https://github.com/open-telemetry/opentelemetry-ebpf-profiler/blob/v0.0.202640/collector/config/config_linux.go) and defaults in [`collector/factory_linux.go`](https://github.com/open-telemetry/opentelemetry-ebpf-profiler/blob/v0.0.202640/collector/factory_linux.go).

| Key | Default | Notes |
|---|---|---|
| `samples_per_second` | `20` | CPU sampling frequency. |
| `reporter_interval` / `reporter_jitter` | `5s` / `0.2` | Export cadence. |
| `interpreters.<name>.disabled` | all enabled | `python`, `perl`, `php`, `hotspot`, `ruby`, `v8`, `dotnet`, `go`, `beam`, `luajit`, `thread_context`. |
| `pid_namespace_translation_mode` | `none` | `none`, `auto`, `exact`, `recursive`. Pins before `v0.0.202640` use boolean `pid_namespace_translation`; each rejects the other's key. Only for [nested containers](nested-containers.md). |
| `probes` | `[]` | Extension IDs, for example `[offcpu]`. |
| `include_env_vars` | `""` | Comma-separated env var names to attach as attributes. |
| `filter_min_process_age` | `0` | Skip short-lived processes. |
| `send_error_frames` / `send_idle_frames` | `false` | Include unwinding-error or idle frames. |
| `error_mode` | `propagate` | `ignore` logs receiver startup errors instead of failing the Collector. |
| `no_kernel_version_check` | `false` | For distro kernels with backported eBPF features. |
| `obi_process_ctx` / `bpf_fs_root` | `false` / `/sys/fs/bpf/` | Share span/trace IDs with OpenTelemetry eBPF Instrumentation (OBI) via a pinned map under `<bpf_fs_root>/otel`. |

Other keys are tuning or debugging knobs; read the source before setting them.

## What it emits

Resource attributes `process.pid`, `process.executable.name`, `process.executable.path`, and `container.id` for containerized processes; sample attributes `thread.name`, `thread.id`, `cpu.logical_number`. It does not set `service.name`: add it with `k8s_attributes`, the `resource` processor, or backend relabeling.

## Off-CPU

Off-CPU needs the `offcpu` extension (`go.opentelemetry.io/ebpf-profiler/probes/offcpu`, present in the module since at least `v0.0.202636`). Not bundled in `0.162.0` (images that bundle it: SKILL.md matrix); otherwise build with OCB importing it:

```yaml
extensions:
  offcpu:
    threshold: 0.1          # capture probability, (0.0, 1.0]
receivers:
  profiling:
    probes: [offcpu]
service:
  extensions: [offcpu]
```

Off-CPU samples arrive as sample type `off_cpu` in `nanoseconds` alongside the CPU samples.
