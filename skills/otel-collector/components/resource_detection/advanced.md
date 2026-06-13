# `resource_detection` — advanced use-cases

## Detector ordering — first detector to set an attribute wins

Detectors run in list order, and the **first** detector to set a given attribute keeps it; later detectors do not overwrite a value already produced earlier in the same run. This matters when several detectors can emit the same key (e.g. `cloud.platform`, `host.id`). The upstream-recommended AWS order, most-specific first, is:

```yaml
processors:
  resource_detection:
    detectors: [lambda, elastic_beanstalk, eks, ecs, ec2]
```

With `detectors: [eks, ec2]`, `cloud.platform` ends up `aws_eks`; reverse them and you would wrongly get `aws_ec2`. Put the most specific platform detector before the more general one. (`override` is a separate axis — it governs detected-vs-incoming, not detector-vs-detector; see [quirks](quirks.md).)

## Combining `env` with a platform detector

A common pattern: pull deploy-time context from the environment **and** auto-detect the platform.

```yaml
processors:
  resource_detection:
    detectors: [env, ec2]
    timeout: 2s
    override: false
```

`env` reads `OTEL_RESOURCE_ATTRIBUTES` (e.g. `service.name`, `service.version`, `deployment.environment.name` injected by your deploy tooling); `ec2` adds the infra identity. `override: false` keeps anything the SDKs already set.

## Routing an attribute to a specific detector

When two detectors can both produce an attribute, disable it on the one you don't want and enable it on the one you do, rather than relying on order:

```yaml
processors:
  resource_detection:
    detectors: [system, ec2]
    system:
      resource_attributes:
        host.name: { enabled: true }
        host.id:   { enabled: false }
    ec2:
      resource_attributes:
        host.name: { enabled: false }
        host.id:   { enabled: true }
```

Here `host.name` comes from `system` and `host.id` from `ec2`, regardless of list order.

## Chaining with the `resource` processor

`resource_detection` discovers; [`resource`](../resource/README.md) normalizes. Run detection first, then clean up what it found — rename legacy keys, drop high-cardinality ones, or hash identifiers:

```yaml
processors:
  resource_detection:
    detectors: [env, system, ec2]
  resource:
    attributes:
      - key: host.id          # drop a high-cardinality detected key
        action: delete
      - key: deployment.environment.name
        value: production      # force a literal regardless of what env said
        action: upsert

service:
  pipelines:
    traces:
      processors: [resource_detection, resource]
```

## Capturing cloud tags / labels

Several cloud detectors can fold instance tags/labels into resource attributes via a regex allow-list:

```yaml
processors:
  resource_detection:
    detectors: [ec2]
    ec2:
      tags:
        - ^team$
        - ^cost-center$
        - ^env.*$
```

`gcp` and `nova` use `labels:` with the same regex semantics; `azure` uses `tags:` and prefixes matches as `azure.tags.<name>`. EC2 tag retrieval needs the `ec2:DescribeTags` IAM permission (or `tags_from_imds: true` with instance metadata tags enabled).

## Periodic refresh

By default detection runs once at startup. Set `refresh_interval` to re-detect for environments where resource attributes change over the Collector's lifetime (cloud instance tags, k8s labels):

```yaml
processors:
  resource_detection:
    detectors: [ec2]
    refresh_interval: 5m
```

Use this sparingly — see the cardinality and performance caveats in [quirks](quirks.md). The upstream guidance is that a single startup detection suffices for most deployments, and intervals below 1 minute are strongly discouraged.
