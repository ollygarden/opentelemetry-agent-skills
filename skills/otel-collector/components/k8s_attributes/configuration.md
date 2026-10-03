# `k8s_attributes`: configuration

## Typical config

```yaml
processors:
  k8s_attributes:
    auth_type: serviceAccount
    extract:
      metadata:
        - k8s.namespace.name
        - k8s.pod.name
        - k8s.pod.uid
        - k8s.deployment.name
        - k8s.node.name

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [k8s_attributes]
      exporters: [otlphttp]
```

## Configuration reference

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `auth_type` | string | `serviceAccount` | How to reach the API: `serviceAccount` (in-cluster), `kubeConfig` (local dev, reads `~/.kube/config` or `$KUBECONFIG`), or `none`. |
| `context` | string | `""` | Kubeconfig context to use; only with `auth_type: kubeConfig`. |
| `kube_api_qps` | float32 | `5` | Max queries/sec to the API. Raise if you see client-side throttling warnings. |
| `kube_api_burst` | int | `10` | Max request burst to the API. Raise alongside `kube_api_qps` under throttling. |
| `passthrough` | bool | `false` | Agent mode: only annotate the record with the pod IP (`k8s.pod.ip`), don't query the API or extract metadata. The gateway does the enrichment. |
| `extract` | object | (see below) | Which metadata, labels, and annotations to add. |
| `filter` | object | `{}` | Restrict which pods are watched (node/namespace/label/field). The main memory lever. |
| `pod_association` | array | (see below) | Ordered strategies for tying a record to a pod; first match wins. |
| `exclude` | object | excludes `jaeger-agent`, `jaeger-collector` | Pods to skip (by name, regex allowed; an invalid regex fails config validation since v0.162.0). A pod can also opt out with the annotation `opentelemetry.io/k8s-processor.ignore: "true"`. |
| `wait_for_metadata` | bool | `false` | Block startup until the metadata cache has synced. |
| `wait_for_metadata_timeout` | duration | `10s` | Max wait for the initial sync when `wait_for_metadata: true`; on timeout the collector fails to start. |
| `watch_sync_period` | duration | `5m` | Informer cache resync period (v0.152.0). Set `0s` to disable resync — recommended for very large clusters, where periodic resyncs cause CPU spikes and memory churn. |
| `pod_delete_grace_period` | duration | `120s` | How long a deleted pod's metadata is kept in the cache before eviction (v0.154.0), so late-arriving records for that pod can still be enriched. |

## `extract`

```yaml
extract:
  metadata:        # resource attributes derived from pod/namespace/node/workload
    - k8s.namespace.name
    - k8s.pod.name
  labels:          # turn k8s labels into resource attributes
    - tag_name: app.label.component       # attribute name to write
      key: app.kubernetes.io/component    # exact label key (mutually exclusive with key_regex)
      from: pod                           # pod | namespace | node | deployment | replicaset | statefulset | daemonset | job | cronjob
  annotations:     # same shape as labels, for annotations
    - tag_name: git.commit
      key: git-commit
      from: pod
  otel_annotations: true                  # auto-extract resource.opentelemetry.io/* annotations
```

**`labels` / `annotations` fields:**

| Field | Default | Meaning |
|-------|---------|---------|
| `tag_name` | `k8s.<from>.label.<key>` / `k8s.<from>.annotation.<key>` (singular, v1 semconv — default since v0.161.0; plural `labels`/`annotations` when the semconv gates are disabled, see [quirks](quirks.md)) | Output attribute name. With `key_regex`, may use `$1` backreferences. |
| `key` | — | Exact key to match. Mutually exclusive with `key_regex`. |
| `key_regex` | — | Regex matched against the **key** (not the value). Mutually exclusive with `key`. |
| `from` | `pod` | Source object: `pod`, `namespace`, `node`, `deployment`, `replicaset`, `statefulset`, `daemonset`, `job`, `cronjob`. |

`otel_annotations: true` lifts pod annotations prefixed `resource.opentelemetry.io/` straight onto the resource (`resource.opentelemetry.io/foo: bar` → attribute `foo: bar`).

`deployment_name_from_replicaset` was removed in v0.160.0 and is now an unknown-key startup error.
Deployment names use the ReplicaSet-name heuristic unless `k8s.deployment.uid` or deployment /
ReplicaSet label or annotation extraction starts the informer lookup. When configuring an explicit
`extract.metadata` list, include `k8s.deployment.name` (or `service.name`) to emit the derived name;
omitting the list uses the defaults below.

### Default-extracted metadata

Even with no `extract.metadata` list, these are added: `k8s.namespace.name`, `k8s.pod.name`, `k8s.pod.uid`, `k8s.pod.start_time`, `k8s.deployment.name`, `k8s.node.name`, and (when a container identifier is present) `container.image.name` plus the image-tag attribute selected by the semantic-convention gates (`container.image.tags` with the v1 gates enabled — the default since v0.161.0 — or `container.image.tag` on v0; see [quirks](quirks.md)). Set an explicit `metadata` list to narrow this.

### Available metadata fields

| Field | Requires |
|-------|----------|
| `k8s.namespace.name`, `k8s.pod.name`, `k8s.pod.uid`, `k8s.pod.hostname`, `k8s.pod.ip`, `k8s.pod.start_time` | — |
| `k8s.deployment.name`, `k8s.deployment.uid`, `k8s.replicaset.name`, `k8s.replicaset.uid` | pod owned by a ReplicaSet |
| `k8s.statefulset.*`, `k8s.daemonset.*`, `k8s.job.*` | pod owned by that workload |
| `k8s.cronjob.name`, `k8s.cronjob.uid` | pod owned by a Job owned by a CronJob |
| `k8s.node.name`, `k8s.node.uid` | — |
| `k8s.cluster.uid` | reads the `kube-system` namespace UID |
| `k8s.container.name`, `container.id`, `container.image.name`, `container.image.tag` (deprecated, use `container.image.tags`), `container.image.tags`, `container.image.repo_digests` | a container identifier on the record (see [quirks](quirks.md)) |
| `service.namespace`, `service.name`, `service.version`, `service.instance.id` | calculated (see below) |

`service.name` is calculated by semantic-convention priority: `app.kubernetes.io/instance` label → `app.kubernetes.io/name` label → workload name → pod name.

## `filter`

```yaml
filter:
  node: node-1                        # only pods on this node (static name)
  node_from_env_var: KUBE_NODE_NAME   # DaemonSet: only this node's pods (set via downward API)
  namespace: production               # single namespace only
  labels:
    - {key: environment, value: production, op: equals}   # op: equals | not-equals | exists | does-not-exist
  fields:
    - {key: status.phase, value: Running, op: equals}     # op: equals | not-equals
```

`filter.node` / `filter.node_from_env_var` and `filter.namespace` are the two big memory levers — without them a DaemonSet caches every pod in the cluster.

## `pod_association`

Each association must be unique. v0.160.0 rejects duplicate source sets even when their sources are
listed in a different order.

Ordered list; the first association whose **every** source matches wins. Default order: `k8s.pod.ip` resource attr → `ip` resource attr → connection IP → `host.name` (if a valid IP).

```yaml
pod_association:
  - sources: [{from: resource_attribute, name: k8s.pod.uid}]
  - sources:                                       # both must match (AND)
      - {from: resource_attribute, name: k8s.pod.name}
      - {from: resource_attribute, name: k8s.namespace.name}
  - sources: [{from: connection}]                  # IP from the incoming connection
```

| `from` | `name` |
|--------|--------|
| `connection` | not used (uses the incoming connection IP) |
| `resource_attribute` | the attribute key to read |

Max **4** sources per association. Of the container-level attributes, only `container.id` is usable here; `service.version` and `service.instance.id` cannot be (they are computed at runtime).
