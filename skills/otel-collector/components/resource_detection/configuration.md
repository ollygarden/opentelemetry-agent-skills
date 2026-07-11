# `resource_detection` — configuration

## Top-level keys

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `detectors` | `[]string` | `[env]` | Ordered list of detectors to run. Valid values: `env`, `system`, `docker`, `heroku`, `gcp`, `ec2`, `ecs`, `elastic_beanstalk`, `eks`, `lambda`, `azure`, `aks`, `consul`, `kubeadm`, `oraclecloud`, `k8s_api`, `k8snode` (deprecated → `k8s_api`), `openshift`, `dynatrace`, `hetzner`, `akamai`, `scaleway`, `upcloud`, `vultr`, `digitalocean`, `nova`, `alibaba_ecs`, `tencent_cvm`, `ibmcloud_vpc`, `ibmcloud_classic`. |
| `override` | `bool` | `true` | Whether detected attributes overwrite resource attributes already present on incoming telemetry. `true` overwrites; `false` keeps existing values and only adds missing ones. |
| `refresh_interval` | `duration` | `0` | If `> 0`, re-runs all detectors on this interval. `0` (default) means detect once at startup and cache. |
| `timeout` | `duration` | `5s` | HTTP client timeout for detectors that call a metadata service. Inherited from the embedded `confighttp.ClientConfig`. |

The component embeds the standard `confighttp.ClientConfig`, so other HTTP client knobs (proxy, TLS, headers) are available for the metadata-service detectors; `timeout` is the one you will usually touch.

> Defaults verified against `factory.go` (`createDefaultConfig`) and `config.go` on contrib v0.156.0: `Detectors: [env]`, `Override: true`, `RefreshInterval: 0`, client `Timeout: 5s`.

## Per-detector configuration

Each detector has its own config block, keyed by the detector name, squashed into the top level:

```yaml
processors:
  resource_detection:
    detectors: [system, ec2]
    system:
      hostname_sources: ["os"]
    ec2:
      tags:
        - ^team$
```

### Selecting which attributes a detector emits — `resource_attributes`

Most detectors expose a `resource_attributes` map that enables/disables individual attributes. This is how you both **opt in** to attributes that are off by default and **route** an attribute to a specific detector when several can produce it:

```yaml
resource_detection:
  detectors: [system, ec2]
  system:
    resource_attributes:
      host.name:
        enabled: true
      host.id:
        enabled: false   # let ec2 own host.id instead
  ec2:
    resource_attributes:
      host.name:
        enabled: false
      host.id:
        enabled: true
```

The set of attributes and their default-enabled state are per detector — consult the detector's `documentation.md` in the upstream source (e.g. `internal/system/documentation.md`). For the `system` detector, only `host.name` and `os.type` are enabled by default; `host.id`, `host.arch`, `os.description`, the `host.cpu.*` family, `host.ip`/`host.mac`/`host.interface`, and others are opt-in.

## Detector catalog

Every detector reports `cloud.provider`/`cloud.platform` plus a platform-specific set. The most-used ones:

| Detector | Source | Key attributes |
|----------|--------|----------------|
| `env` | `OTEL_RESOURCE_ATTRIBUTES` env var (falls back to deprecated `OTEL_RESOURCE`), `k=v,k=v` format | whatever you put in the variable |
| `system` | host machine | `host.name`, `os.type` (default); `host.id`, `host.arch`, `host.cpu.*`, `os.description`, … (opt-in). `hostname_sources` (`["dns","os"]` default; also `cname`, `lookup`) controls how `host.name` is resolved |
| `docker` | Docker daemon (mount the socket) | `host.name`, `os.type`. Use instead of `system` when the Collector runs as a container; **does not work on macOS** |
| `ec2` | EC2 IMDS | `cloud.*`, `host.id`, `host.name`, `host.type`. Optional `tags` (regex list; needs `ec2:DescribeTags` IAM, or `tags_from_imds: true`). `fail_on_missing_metadata`, `max_attempts`, `max_backoff` |
| `ecs` | ECS Task Metadata Endpoint (V4/V3) | `cloud.*`, `aws.ecs.*` |
| `eks` | EC2 IMDS + k8s/EC2 API fallback | `cloud.*`; `k8s.cluster.name` opt-in (needs `EC2:DescribeInstances`). `node_from_env_var` |
| `lambda` | Lambda runtime env vars | `cloud.*`, `faas.*` |
| `gcp` | GCP metadata server | `cloud.*`, `host.*`, `k8s.cluster.name`, `faas.*` per platform (GCE/GKE/Cloud Run/Functions/App Engine). Optional `labels` (regex list; needs `roles/compute.viewer`) |
| `azure` | Azure IMDS | `cloud.*`, `host.*`. Optional `tags` (regex → `azure.tags.<name>`) |
| `aks` | Azure IMDS | `cloud.*`; `k8s.cluster.name` opt-in |
| `k8s_api` | k8s API server | node/cluster attrs; requires `node_from_env_var` (default `K8S_NODE_NAME`) and `nodes` RBAC. `auth_type` (`serviceAccount` default / `none` / `kubeConfig`). `k8snode` is the deprecated alias |
| `kubeadm` | k8s API (`kubeadm-config` ConfigMap) | `k8s.cluster.name`, `k8s.cluster.uid`. `auth_type` |
| `openshift` | OpenShift/k8s API | `cloud.*`, `k8s.cluster.name`. `address`, `token`, `tls` |
| `heroku` | Heroku dyno metadata env vars | `service.name`, `service.version`, `service.instance.id`, `heroku.*` |
| `dynatrace` | `dt_host_metadata.properties` file | `dt.entity.host`, `host.name`, `dt.smartscape.host` |
| `consul` | Consul agent | node + exploded `_node_meta` |

Additional metadata-service detectors follow the same shape (a `fail_on_missing_metadata` flag, sometimes a `labels`/`tags` regex list): `hetzner`, `akamai`, `scaleway`, `upcloud`, `vultr`, `digitalocean`, `nova` (OpenStack), `alibaba_ecs`, `tencent_cvm`, `ibmcloud_vpc` (`protocol: http|https`), `ibmcloud_classic`, `oraclecloud`, `elastic_beanstalk`, `consul`.

For the exact attribute list any detector emits, read its `internal/<detector>/documentation.md` in the upstream source — do not assume.
