# `k8s_attributes`: verification

See [Verification harness](../../SKILL.md#verification-harness) for the general approach. Unlike the other component recipes, this one **needs a real Kubernetes cluster** — the processor enriches from the live Kubernetes API, so there is nothing to verify without one. A throwaway [`kind`](https://kind.sigs.k8s.io/) cluster is enough. `k8s_attributes` ships in the `contrib` and `k8s` distributions, so the stock contrib image works.

The idea: run the collector **inside** the cluster (so its ServiceAccount can watch the API), deploy a throwaway workload to enrich against, then send a trace tagged with that workload's pod IP. The default `k8s.pod.ip` pod-association matches it to the cached pod, and the pod's metadata appears on the trace's resource.

## 1. Cluster

```bash
kind create cluster --name k8sattr-verify --wait 90s
```

## 2. RBAC + collector + a workload to associate against

Apply this (ServiceAccount, ClusterRole/Binding, config, collector Deployment):

```yaml
apiVersion: v1
kind: ServiceAccount
metadata: {name: otel-collector, namespace: default}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: {name: otel-collector}
rules:
  - apiGroups: [""]
    resources: ["pods", "namespaces", "nodes"]
    verbs: ["get", "watch", "list"]
  - apiGroups: ["apps"]
    resources: ["replicasets", "deployments"]   # optional: only for deployment_name_from_replicaset:false or k8s.deployment.uid; the default heuristic needs neither
    verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata: {name: otel-collector}
subjects: [{kind: ServiceAccount, name: otel-collector, namespace: default}]
roleRef: {kind: ClusterRole, name: otel-collector, apiGroup: rbac.authorization.k8s.io}
---
apiVersion: v1
kind: ConfigMap
metadata: {name: otel-collector-config, namespace: default}
data:
  config.yaml: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
    processors:
      k8s_attributes:
        auth_type: serviceAccount
        extract:
          metadata: [k8s.namespace.name, k8s.pod.name, k8s.pod.uid, k8s.deployment.name, k8s.node.name]
        pod_association:
          - sources: [{from: resource_attribute, name: k8s.pod.ip}]
    exporters:
      debug:
        verbosity: detailed
    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [k8s_attributes]
          exporters: [debug]
---
apiVersion: apps/v1
kind: Deployment
metadata: {name: otel-collector, namespace: default}
spec:
  replicas: 1
  selector: {matchLabels: {app: otel-collector}}
  template:
    metadata: {labels: {app: otel-collector}}
    spec:
      serviceAccountName: otel-collector
      containers:
        - name: otel-collector
          image: otel/opentelemetry-collector-contrib:0.156.0
          args: ["--config=/etc/otel/config.yaml"]
          ports: [{containerPort: 4317}]
          volumeMounts: [{name: config, mountPath: /etc/otel}]
      volumes: [{name: config, configMap: {name: otel-collector-config}}]
```

```bash
kubectl apply -f collector.yaml
kubectl create deployment demo --image=nginx --replicas=1   # the workload we'll enrich against
kubectl rollout status deployment/otel-collector
kubectl rollout status deployment/demo
```

## 3. Send a trace tagged with the demo pod's IP

```bash
DEMO_IP=$(kubectl get pod -l app=demo -o jsonpath='{.items[0].status.podIP}')
kubectl port-forward deployment/otel-collector 14317:4317 &   # background

telemetrygen traces --otlp-insecure --otlp-endpoint localhost:14317 \
  --traces 1 --workers 1 \
  --otlp-attributes "k8s.pod.ip=\"$DEMO_IP\""
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-insecure`, `--otlp-endpoint`, `--traces` (int, **per worker**; no `--duration`, or the count is ignored), `--workers`, and `--otlp-attributes` (**resource-level** — the right scope, since pod association and the added `k8s.*` attributes all live on the resource; `--telemetry-attributes` would put it at span level where association can't see it). The `k8s.pod.ip` value must be a real pod IP from the cluster so it matches a cached pod.

## What proves it worked

```bash
kubectl logs deployment/otel-collector | grep -A7 'Resource attributes'
```

The trace went in carrying only `k8s.pod.ip` and came out with the demo pod's metadata added to its resource (verified run):

```
Resource attributes:
     -> k8s.pod.ip: Str(10.244.0.5)
     -> service.name: Str(telemetrygen)
     -> k8s.pod.name: Str(demo-54d7464888-9zhvg)
     -> k8s.namespace.name: Str(default)
     -> k8s.pod.uid: Str(a4c56ba1-2f78-480c-99d0-3160d9c7c42e)
     -> k8s.deployment.name: Str(demo)
     -> k8s.node.name: Str(k8sattr-verify-control-plane)
```

`k8s.deployment.name: demo` is the strongest signal — producing it means the processor resolved the pod purely from the `k8s.pod.ip` we set, then derived the deployment name from the pod's owner ReplicaSet name via the default heuristic (which is why the `replicasets`/`deployments` grants above are optional — needed only if you set `deployment_name_from_replicaset: false` or extract `k8s.deployment.uid`).

## Teardown

```bash
kind delete cluster --name k8sattr-verify
```
