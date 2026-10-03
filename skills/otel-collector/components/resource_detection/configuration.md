# `resource_detection` — configuration

## Top-level keys

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `detectors` | `[]string` | `[env]` | Ordered list of detectors to run. Valid values: `env`, `system`, `docker`, `heroku`, `gcp`, `ec2`, `ecs`, `elastic_beanstalk`, `eks`, `lambda`, `azure`, `aks`, `azureappservice`, `azurecontainerapps`, `consul`, `kubeadm`, `oraclecloud`, `k8s_api`, `k8snode` (deprecated → `k8s_api`), `openshift`, `dynatrace`, `hetzner`, `akamai`, `scaleway`, `upcloud`, `vultr`, `digitalocean`, `nova`, `alibaba_ecs`, `tencent_cvm`, `ibmcloud_vpc`, `ibmcloud_classic`. |
| `override` | `bool` | `true` | Whether detected attributes overwrite resource attributes already present on incoming telemetry. `true` overwrites; `false` keeps existing values and only adds missing ones. |
| `refresh_interval` | `duration` | `0` | If `> 0`, re-runs all detectors on this interval. `0` (default) means detect once at startup and cache. |
| `fail_on_missing_metadata` | `bool` | `false` | For supported network metadata detectors, make an unreachable metadata service a hard error that participates in processor retry instead of producing an empty resource. Added in v0.158.0. |
| `timeout` | `duration` | `5s` | HTTP client timeout for detectors that call a metadata service. Inherited from the embedded `confighttp.ClientConfig`. |
| `retry` | object | enabled (see below) | Global exponential-backoff policy applied to every detector attempt, including periodic refreshes. Added in v0.159.0. |

The component embeds the standard `confighttp.ClientConfig`, so other HTTP client knobs (proxy, TLS, headers) are available for the metadata-service detectors; `timeout` is the one you will usually touch.

> Defaults verified against `factory.go` (`createDefaultConfig`) and `config.go` on contrib v0.160.0: `Detectors: [env]`, `Override: true`, `RefreshInterval: 0`, `FailOnMissingMetadata: false`, client `Timeout: 5s`, and retry enabled with the values below.

## Global retry

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `retry.enabled` | bool | `true` | `false` makes one detector attempt with no retry. |
| `retry.initial_interval` | duration | `1s` | Delay before the first retry. |
| `retry.randomization_factor` | float | `0.5` | Jitter applied to retry intervals. |
| `retry.multiplier` | float | `2` | Backoff growth factor. |
| `retry.max_interval` | duration | `30s` | Maximum delay between attempts. |
| `retry.max_elapsed_time` | duration | `0` | Overall retry budget. `0` uses `timeout` as the overall session bound; a positive value becomes the session bound and `timeout` limits each attempt. |

With retry enabled, setting both `timeout: 0` and `retry.max_elapsed_time: 0` is rejected because
the detector could block startup indefinitely.

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
| `ec2` | EC2 IMDS | `cloud.*`, `host.id`, `host.name`, `host.type`. Optional `tags` (regex list; needs `ec2:DescribeTags` IAM, or `tags_from_imds: true`). Deprecated per-detector `fail_on_missing_metadata` (use the top-level key), plus `max_attempts`, `max_backoff` |
| `ecs` | ECS Task Metadata Endpoint (V4/V3) | `cloud.*`, `aws.ecs.*` |
| `eks` | EC2 IMDS + k8s/EC2 API fallback | `cloud.*`; `k8s.cluster.name` opt-in (needs `EC2:DescribeInstances`). `node_from_env_var` |
| `lambda` | Lambda runtime env vars | `cloud.*`, `faas.*` |
| `gcp` | GCP metadata server | `cloud.*`, `host.*`, `k8s.cluster.name`, `faas.*` per platform (GCE/GKE/Cloud Run/Functions/App Engine). Optional `labels` (regex list; needs `roles/compute.viewer`). Since v0.161.0, `host.type` on GKE is fetched via the Compute API (needs `compute.instances.get`; disable `host.type` to skip the call) |
| `azure` | Azure IMDS | `cloud.*`, `host.*`. Optional `tags` (regex → `azure.tags.<name>`) |
| `aks` | Azure IMDS | `cloud.*`; `k8s.cluster.name` opt-in |
| `azureappservice` | Azure App Service environment variables | `azure.app_service.instance.id`, `azure.resource_group.name`, `cloud.*`, `deployment.environment.name`, `service.name` |
| `azurefunctions` | Azure Functions environment variables (`FUNCTIONS_WORKER_RUNTIME` / `FUNCTIONS_EXTENSION_VERSION`); v0.162.0+ | `azure.resource_group.name`, `cloud.*` (incl. `cloud.resource_id`), `faas.instance`, `service.name`; `deployment.environment.name` opt-in. Flex Consumption plans lack `azure.resource_group.name` / `cloud.resource_id` |
| `azurecontainerapps` | Azure Container Apps environment variables | `azure.container_app.instance.id`, `cloud.platform`, `cloud.provider`, `service.name` |
| `k8s_api` | k8s API server | node/cluster attrs; requires `node_from_env_var` (default `K8S_NODE_NAME`) and `nodes` RBAC. `auth_type` (`serviceAccount` default / `none` / `kubeConfig`). `k8snode` is the deprecated alias |
| `kubeadm` | k8s API (`kubeadm-config` ConfigMap) | `k8s.cluster.name`, `k8s.cluster.uid`. `auth_type` |
| `openshift` | OpenShift/k8s API | `cloud.*`, `k8s.cluster.name`. `address`, `token`, `tls` |
| `heroku` | Heroku dyno metadata env vars | `service.name`, `service.version`, `service.instance.id`, `heroku.*` |
| `dynatrace` | `dt_host_metadata.properties` file | `dt.entity.host`, `host.name`, `dt.smartscape.host` |
| `consul` | Consul agent | node + exploded `_node_meta` |

Additional metadata-service detectors have detector-specific settings, sometimes including a `labels`/`tags` regex list: `hetzner`, `akamai`, `scaleway`, `upcloud`, `vultr`, `digitalocean`, `nova` (OpenStack), `alibaba_ecs`, `tencent_cvm`, `ibmcloud_vpc` (`protocol: http|https`), `ibmcloud_classic`, `oraclecloud`, `elastic_beanstalk`, `consul`. The per-detector `fail_on_missing_metadata` fields on `upcloud`, `vultr`, `nova`, `alibaba_ecs`, and `tencent_cvm` are deprecated; use the top-level key instead.

For the exact attribute list any detector emits, read its `internal/<detector>/documentation.md` in the upstream source — do not assume.

## Detector migration gates

- `processor.resourcedetection.elasticbeanstalk.EmitV1DeploymentConventions` (Alpha, off by
  default) adds `deployment.environment.name` and `deployment.id` alongside the legacy attributes.
- `processor.resourcedetection.elasticbeanstalk.DontEmitV0DeploymentConventions` (Alpha, off by
  default) removes deprecated `deployment.environment` and `service.instance.id`; it is rejected
  unless the emit-v1 gate is also enabled.
- `processor.resourcedetection.consul.prefixMetaAttributes` (Alpha, off by default) emits Consul
  node metadata as `consul.meta.<key>` instead of unprefixed `<key>`.
