# Nested-container environments (kind, k3d, Docker-in-Docker)

In kind, k3d, minikube with the docker driver, and Docker-in-Docker, each Kubernetes "node" is itself a container. Two things break that work on real nodes (EKS, GKE, AKS, bare-metal kubelets), where `hostPID` reaches the kernel's root PID namespace and tracefs is usually mounted.

Docker Desktop, Colima, and similar VMs are not nested by themselves: a `docker run --pid=host` there reaches the VM's root PID namespace. They become nested only when you run kind or k3d inside them.

## 1. PID namespaces

`hostPID: true` reaches only the node container's PID namespace; pods sit in descendant namespaces. Without translation the receiver fails at startup. The boolean `pid_namespace_translation: true` (ebpf-profiler `v0.0.202636`, Collector `0.162.0`) starts but sees only the node's own processes (kubelet/k3s, containerd, shims), never pods. Use `pid_namespace_translation_mode: auto` (ebpf-profiler `v0.0.202640`+; images in the SKILL.md matrix). Exact errors and logs for each state, including the invalid-key errors when the key does not match the pin: `troubleshooting.md`.

Modes: `none` (default; correct on real nodes, which run in the root PID namespace); `exact` (own namespace only, no BTF needed); `auto` (adds descendants when kernel BTF at `/sys/kernel/btf/vmlinux` exposes the layout, else falls back to exact); `recursive` (requires BTF, fails at startup without it). Tasks outside the profiler's namespace tree are dropped.

## 2. tracefs missing inside the node

The node container gets a fresh sysfs without tracefs, so a hostPath of `/sys/kernel/tracing` (or `/sys/kernel/debug`) is an empty directory. Startup fails with `... neither debugfs nor tracefs are mounted` (full string in `troubleshooting.md`).

Fix: a privileged init container that enters the node's mount namespace and mounts tracefs if absent, plus the existing hostPath mount in the profiler container. It is a no-op where tracefs is already mounted. Add to the DaemonSet pod spec from `kubernetes.md`:

```yaml
      initContainers:
        - name: mount-tracefs
          image: busybox:1.37
          command:
            - sh
            - -c
            - >-
              nsenter -t 1 -m -- env PATH=/bin/aux:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin sh -c
              'grep -q " /sys/kernel/tracing tracefs " /proc/mounts
              || mount -t tracefs nodev /sys/kernel/tracing'
          securityContext:
            privileged: true
```

`nsenter -t 1` works because the pod has `hostPID: true`, so PID 1 is the node's init. The explicit `PATH` matters on k3s nodes, which keep `mount` in `/bin/aux`; without it the init container fails with `mount: not found`.

For plain `docker run` of the profiler inside a Docker-in-Docker host, bind-mount `/sys/kernel/tracing` from a level that has tracefs, or mount it in that inner host first.

## Combined changes for kind/k3d

1. An image that pins ebpf-profiler `v0.0.202640`+ (see the SKILL.md version matrix; `0.163.0` once released).
2. `pid_namespace_translation_mode: auto` under `receivers.profiling`.
3. The `mount-tracefs` init container above.
