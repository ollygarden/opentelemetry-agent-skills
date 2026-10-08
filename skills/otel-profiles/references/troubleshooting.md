# Troubleshooting the eBPF profiler

Match the exact log or error text. Strings are from Collector `0.162.0` / ebpf-profiler `v0.0.202636` and `v0.0.202640` unless noted.

| Error or symptom | Cause | Fix |
|---|---|---|
| `pipeline "profiles": profiling signal support is at alpha level, gated under the "service.profilesSupport" feature gate` | Gate not enabled. | Add `--feature-gates=+service.profilesSupport` (also on any relay Collector with a profiles pipeline). |
| `'config.Config' has invalid keys: pid_namespace_translation_mode` | Key from ebpf-profiler `v0.0.202640`+ on an older pin (for example `0.162.0`). | Use `pid_namespace_translation: true` there, or move to a `v0.0.202640`+ image. |
| `'config.Config' has invalid keys: pid_namespace_translation` | Boolean key removed in `v0.0.202640`. | Use `pid_namespace_translation_mode`. |
| `failed to create "batch" processor, in pipeline "profiles": telemetry type is not supported` | `batch` has no profiles support. | Remove it from the profiles pipeline. |
| `'extensions' unknown type: "offcpu"` | Distribution does not bundle the off-CPU probe (`0.162.0`). | Use `0.163.0-nightly.3d037de` or later, or an OCB build importing `go.opentelemetry.io/ebpf-profiler/probes/offcpu`. |
| `failed to attach scheduler monitor: failed to configure tracepoint on tracer.hookPoint{group:"sched", name:"sched_process_free"}: neither debugfs nor tracefs are mounted` | tracefs absent in the container's `/sys`. | Mount `/sys/kernel/tracing` from the host. If the host path is empty (kind/k3d nodes), mount tracefs on the node first: `nested-containers.md`. |
| `mount: not found` from a tracefs init container | k3s keeps `mount` in `/bin/aux`. | Set `PATH` explicitly inside `nsenter` (see `nested-containers.md`). |
| `failed to determine system configs: system analysis request was not handled for pid N` | Profiler is not in the kernel's root PID namespace (nested node container) and translation is off. | `pid_namespace_translation_mode: auto` (or the boolean on older pins), plus the visibility caveat below. |
| Only node processes (kubelet/k3s, containerd, shims) appear; no pods. Log: `only processes traces within the profiler namespace will be collected` | Boolean translation (`v0.0.202636`) only covers the profiler's own PID namespace. | `pid_namespace_translation_mode: auto` on ebpf-profiler `v0.0.202640`+. |
| `recursive PID namespace translation requires readable kernel BTF ...` | `recursive` mode without kernel BTF. | Use `auto` or `exact`, or a kernel with `/sys/kernel/btf/vmlinux`. |
| `host Agent requires kernel version 5.10 or newer but got X.Y.Z` | Old kernel, or a distro kernel with backported eBPF. | Upgrade the kernel, or set `no_kernel_version_check: true` only if the distro backports the needed features. |
| `failed to load eBPF tracer: failed to read kernel symbols: unable to read kallsyms addresses - check capabilities` (or other eBPF load or permission errors) | Not privileged (observed with `docker run` without `--privileged`). | Run privileged or with `CAP_SYS_ADMIN`, `CAP_BPF`, `CAP_PERFMON`, and the host PID namespace. |
| `memory_limiter` logs `total_memory_mib` equal to node memory | Percentage limits resolved against the node, not the container. | Use `limit_mib` and `spike_limit_mib`. |
| Profiles arrive without `k8s.*` attributes | No `container.id` pod association, RBAC gap, or the process is not in a pod. | `pod_association` on `container.id`; RBAC for pods and namespaces; see `otel-collector` `components/k8s_attributes/quirks.md`. |
| Many `containerd-shim-*` or `unknown_service:*` services in the backend | Host processes with no pod and no `service.name`. | Expected. Fallback `service.name` via `resource` `action: insert` (`kubernetes.md`), or drop downstream with `filter` `profile_conditions` on `process.executable.name` (`filter` is not in `otelcol-ebpf-profiler`). |
| Exporter logs only an HTTP status (for example 400) for every batch | The Collector does not log the response body. | Capture one request body (point `otlp_http` at a local listener) and replay it with `curl -v --data-binary @body.bin` and the same headers (`Content-Type: application/x-protobuf`, `Content-Encoding` if compressed, auth) to read the backend's error. For Grafana Cloud's 400 on OTLP/HTTP, see `backends.md`. |
| `Skip pinning eBPF map to share OTel span/trace IDs` (info) | `obi_process_ctx` is off. | Informational. Enable it only to correlate with OBI. |
| Frames show addresses or `unknown` | Symbols not available to the backend. | Backend-dependent; check the backend's symbolization docs. Go binaries are symbolized on the host. |

For anything else, raise `service.telemetry.logs.level: debug` (the receiver's `verbose_mode` adds more eBPF detail) and check the [ebpf-profiler issues](https://github.com/open-telemetry/opentelemetry-ebpf-profiler/issues).
