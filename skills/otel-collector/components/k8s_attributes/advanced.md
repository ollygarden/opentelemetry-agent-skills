# `k8s_attributes`: advanced use-cases

## RBAC

The processor reads the Kubernetes API, so its ServiceAccount needs permission for exactly the resources you extract. Grant `get`/`list`/`watch` and nothing more.

**Minimum (any deployment)** — `pods` and `namespaces`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: otel-collector
rules:
  - apiGroups: [""]
    resources: ["pods", "namespaces"]
    verbs: ["get", "watch", "list"]
```

Add resources as you extract more metadata:

| Extracted metadata | Add resource | API group |
|--------------------|--------------|-----------|
| `k8s.deployment.name` (default path) | `replicasets` | `apps`, `extensions` |
| `k8s.deployment.uid`, deployment labels/annotations | `deployments` | `apps` |
| `k8s.statefulset.*` / `k8s.daemonset.*` | `statefulsets` / `daemonsets` | `apps` |
| `k8s.job.*`, `k8s.cronjob.*` | `jobs` | `batch` |
| `k8s.node.*` | `nodes` | `` (core) |

Bind it cluster-wide with a `ClusterRoleBinding` when the collector receives telemetry from many namespaces. When the collector only handles one namespace, use a namespaced `Role` + `RoleBinding` with `filter.namespace` set — least privilege, but it **cannot** read nodes, `k8s.cluster.uid`, or namespace labels/annotations.

## Agent/gateway split (passthrough)

Run the processor twice across two tiers. A per-node **agent** in `passthrough: true` mode only stamps the source pod IP (`k8s.pod.ip`) — no API calls, no RBAC — while a central **gateway** does the real enrichment off that IP.

```yaml
# Agent (DaemonSet) — just record the pod IP
processors:
  k8s_attributes:
    passthrough: true

# Gateway (Deployment) — enrich off the IP the agent stamped
processors:
  k8s_attributes:
    auth_type: serviceAccount
    extract:
      metadata: [k8s.namespace.name, k8s.pod.name, k8s.deployment.name, k8s.node.name]
    pod_association:
      - sources:
          - {from: resource_attribute, name: k8s.pod.ip}
```

This concentrates the API watch and RBAC on the gateway tier instead of every node.

## Filtering for memory (large clusters)

Memory scales with pods watched (~5 KB/pod). The two biggest levers:

```yaml
# DaemonSet: watch only this node's pods — set KUBE_NODE_NAME via the downward API
filter:
  node_from_env_var: KUBE_NODE_NAME
```

```yaml
# Single-namespace collector: watch only that namespace
filter:
  namespace: production
```

`node_from_env_var` pairs with a pod env var sourced from `spec.nodeName`:

```yaml
env:
  - name: KUBE_NODE_NAME
    valueFrom:
      fieldRef:
        fieldPath: spec.nodeName
```

## Labels & annotations

Turn Kubernetes labels/annotations into resource attributes. Use exact `key` matching where possible; `key_regex` matches the **key** and can fan out to many attributes (and costs CPU):

```yaml
extract:
  labels:
    - {tag_name: app.label.component, key: app.kubernetes.io/component, from: pod}
    - {tag_name: team, key: team, from: deployment}   # needs deployments RBAC + watch
  annotations:
    - {tag_name: git.commit, key: git-commit, from: pod}
  otel_annotations: true   # also lift resource.opentelemetry.io/* annotations
```

Extracting `from:` a workload (deployment/statefulset/daemonset/job) makes the processor watch that resource — extra RBAC and roughly +30% memory. Only do it when you need it.

## `deployment_name_from_replicaset`

Derive `k8s.deployment.name` by trimming the pod-template-hash off the ReplicaSet name, so you don't have to watch ReplicaSets at all:

```yaml
extract:
  deployment_name_from_replicaset: true
```

Cheaper RBAC (no `replicasets`) and less memory, but it sets a wrong name for pods owned by a **standalone** ReplicaSet (one with no Deployment), and very long deployment names (>63 chars) come back truncated. Use it only when all pods are Deployment-managed.

## `wait_for_metadata`

Block startup until the metadata cache has synced, so telemetry arriving immediately on boot is still enriched:

```yaml
processors:
  k8s_attributes:
    wait_for_metadata: true
    metadata_sync_timeout: 30s
```

Trade-off: guaranteed enrichment of early data, but a slower start — and if the sync doesn't finish within `metadata_sync_timeout` the collector **fails to start**. Leave it `false` (default) when fast, best-effort startup matters more than enriching the first few records.
