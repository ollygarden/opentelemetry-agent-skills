# eBPF profiler on Kubernetes

Run one profiler per node as a DaemonSet. Each pod profiles every process on its node, so Kubernetes metadata comes from matching `container.id` to pods, not from pod IPs.

Component details are in the `otel-collector` skill: `components/k8s_attributes/` (association, `extract`, computed `service.name`) and `components/memory_limiter/` (limit keys and validation).

## Requirements

| Need | Why |
|---|---|
| `hostPID: true` | The profiler must see node processes. |
| `privileged: true` (or `CAP_SYS_ADMIN`, `CAP_BPF`, `CAP_PERFMON`) | Loads eBPF programs and reads other processes' memory. |
| hostPath `/sys/kernel/tracing` | Scheduler tracepoints need tracefs; the container's own `/sys` lacks it. |
| ServiceAccount with get/list/watch on `pods` and `namespaces` | `k8s_attributes` with the extract list below. Other metadata needs more resources; see the RBAC table in the `otel-collector` skill's `components/k8s_attributes/advanced.md`. |
| `memory_limiter` with `limit_mib`/`spike_limit_mib` | In a privileged hostPID pod, `limit_percentage` resolved against node memory instead of the container limit in field testing (logged `total_memory_mib` equal to the node). Absolute limits avoid that. Keep `limit_mib` below the container memory limit. |

## Manifest

Minimal and generic. Replace the exporter endpoint and cluster name; see `backends.md` for endpoints. On kind/k3d/Docker-in-Docker nodes, apply the changes in `nested-containers.md`.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: profiling
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: otel-ebpf-profiler
  namespace: profiling
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: otel-ebpf-profiler
rules:
  - apiGroups: [""]
    resources: [pods, namespaces]
    verbs: [get, list, watch]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: otel-ebpf-profiler
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: otel-ebpf-profiler
subjects:
  - kind: ServiceAccount
    name: otel-ebpf-profiler
    namespace: profiling
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-ebpf-profiler
  namespace: profiling
data:
  config.yaml: |
    receivers:
      profiling:
        samples_per_second: 97
    processors:
      memory_limiter:
        check_interval: 1s
        limit_mib: 800
        spike_limit_mib: 200
      k8s_attributes:
        filter:
          node_from_env_var: K8S_NODE_NAME
        pod_association:
          - sources:
              - from: resource_attribute
                name: container.id
        extract:
          metadata:
            - k8s.namespace.name
            - k8s.pod.name
            - k8s.deployment.name
            - k8s.node.name
            - k8s.container.name
            - service.name
      resource:
        attributes:
          - key: k8s.cluster.name
            value: my-cluster
            action: upsert
    exporters:
      otlp_grpc:
        endpoint: profiles-backend.observability.svc:4317
        tls:
          insecure: true
    service:
      pipelines:
        profiles:
          receivers: [profiling]
          processors: [memory_limiter, k8s_attributes, resource]
          exporters: [otlp_grpc]
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: otel-ebpf-profiler
  namespace: profiling
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: otel-ebpf-profiler
  template:
    metadata:
      labels:
        app.kubernetes.io/name: otel-ebpf-profiler
    spec:
      serviceAccountName: otel-ebpf-profiler
      hostPID: true
      tolerations:
        - operator: Exists
      containers:
        - name: profiler
          image: otel/opentelemetry-collector-ebpf-profiler:0.162.0
          args:
            - --config=/conf/config.yaml
            - --feature-gates=+service.profilesSupport
          env:
            - name: K8S_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              memory: 1Gi
          volumeMounts:
            - name: conf
              mountPath: /conf
            - name: tracefs
              mountPath: /sys/kernel/tracing
      volumes:
        - name: conf
          configMap:
            name: otel-ebpf-profiler
        - name: tracefs
          hostPath:
            path: /sys/kernel/tracing
```

Notes:

- `k8s_attributes` computes `service.name` for pod processes (order in the `otel-collector` skill's `components/k8s_attributes/configuration.md`). Processes with no matching pod (kubelet, container runtime, shims, host daemons) keep only `process.*` attributes; give them a fallback `service.name` (for example `resource` with `action: insert`, which keeps pod-derived values) if the backend needs one.
- Field measurement at 97 Hz on a small cluster: roughly 6 to 13 mCPU and 110 to 370 MiB per node. Size requests from your own measurement; upstream states 1% CPU and 250 MB as its testing upper bounds.
- To add the `offcpu` probe or OBI correlation, see `receiver.md`; OBI correlation also needs `/sys/fs/bpf` mounted from the host.

## Verify

```bash
kubectl --context <ctx> -n profiling rollout status ds/otel-ebpf-profiler
kubectl --context <ctx> -n profiling logs ds/otel-ebpf-profiler | grep -E 'Attached sched monitor|Everything is ready|error'
```

Then confirm pod workloads (not only node processes like the kubelet or container runtime) appear in the backend with `k8s.*` attributes. If only node processes appear on a kind/k3d cluster, read `nested-containers.md`.
