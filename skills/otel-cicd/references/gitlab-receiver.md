# GitLab receiver (`gitlab`)

Facts for the contrib [`gitlabreceiver`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.166.0/receiver/gitlabreceiver)
at collector-contrib [v0.166.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/CHANGELOG.md).
Values hold for the named releases; check the current release before relying on them.

## Identity

| Fact | Value at v0.166.0 | Source |
|---|---|---|
| Component type | `gitlab` | [metadata.yaml](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/gitlabreceiver/metadata.yaml) |
| Signals and stability | traces only, **no metrics**; alpha in metadata.yaml, but the factory registers traces as development, which is what the Collector reports | metadata.yaml; [factory.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/gitlabreceiver/factory.go) |
| Distribution | `otelcol-contrib` only | [otelcol-contrib manifest](https://github.com/open-telemetry/opentelemetry-collector-releases/blob/v0.166.0/distributions/otelcol-contrib/manifest.yaml) |
| Semantic conventions it emits | v1.43.0 (Go `semconv/v1.43.0` import) | [model.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/gitlabreceiver/model.go) |

## Configuration

```yaml
# gitlab receiver: otelcol-contrib only; traces alpha in metadata, development in the factory
receivers:
  gitlab:
    webhook:
      endpoint: 0.0.0.0:19418                # default localhost:8080 is loopback only
      path: /events                          # default
      health_path: /health                   # default
      secret: ${env:GITLAB_WEBHOOK_TOKEN}    # compared with the X-Gitlab-Token header
      include_user_attributes: false         # default
exporters:
  debug:
    verbosity: detailed                      # replace with otlp_grpc or otlp_http once verified
service:
  pipelines:
    traces:
      receivers: [gitlab]
      exporters: [debug]
```

Source: [README configuration](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/gitlabreceiver/README.md#configuration),
[config.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/gitlabreceiver/config.go).

- `secret` is compared for equality with the `X-Gitlab-Token` header (GitLab's webhook secret
  token); there is no HMAC signature. A mismatch returns 400.
- Only POST is accepted. Timeouts default to 500ms, with a 10s maximum.
- `required_headers` enforces extra headers only when listed (for self-managed GitLab, for example
  `X-Gitlab-Instance`); it cannot override `X-Gitlab-Webhook-UUID`, `X-Gitlab-Event`,
  `X-Gitlab-Event-UUID` or `Idempotency-Key`.
- `include_user_attributes: true` adds `vcs.ref.head.revision.author.name`, `.author.email` and
  `.message`, `cicd.pipeline.run.actor.id` and `.name`, and `gitlab.pipeline.run.actor.username`.
- `service.name` is always the project's `path_with_namespace`; there is no override option.

GitLab side: add a project or group webhook pointing at the receiver URL, set the secret token,
and enable **Pipeline events** only. Job events are rejected with 400; job spans come from the
`builds` listed in the pipeline event ([traces_receiver.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/gitlabreceiver/traces_receiver.go)).

## Spans

Pipeline events produce spans only for finished pipelines (`success`, `failed`, `canceled`,
`skipped`); running or pending ones return 204.

| Span | Kind | Name | Time range |
|---|---|---|---|
| Pipeline (root) | SERVER | pipeline name, else the commit title | `created_at` to `finished_at` |
| Stage | SERVER | stage name | earliest job start to latest job finish |
| Job | SERVER | job name | `started_at` to `finished_at` |

There are no step or queue spans. Status: `success` is OK, `failed` and `canceled` are ERROR,
others UNSET; a stage takes the status of its first job, not an aggregate.

Attributes use the convention keys with raw GitLab values: resources carry `cicd.pipeline.name`, `cicd.pipeline.run.id`, `cicd.pipeline.run.url.full`,
`cicd.pipeline.result` (for example `failed`, not the convention's `failure`),
`vcs.repository.url.full`, `vcs.ref.head.name`, `vcs.ref.head.revision`, and for merge-request
pipelines `vcs.change.id`, `vcs.change.state` and `vcs.ref.base.name`. Job spans carry
`cicd.pipeline.task.name`, `cicd.pipeline.task.run.id`, `cicd.pipeline.task.run.result` (raw job
status), `cicd.worker.id` and `cicd.worker.name`. `cicd.pipeline.run.id`,
`cicd.pipeline.task.run.id` and `cicd.worker.id` are integers; the conventions define strings.

Keys the registry does not define: `vcs.repository.visibility`, `vcs.repository.ref.default`,
`vcs.ref.head.revision.timestamp`, `cicd.pipeline.task.run.environment.name`, `cicd.worker.type`,
`cicd.worker.tags`, `cicd.worker.shared`, the `gitlab.*` keys, and the `include_user_attributes`
keys. Source: [traces_event_handling.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/gitlabreceiver/traces_event_handling.go).

## Limitations

From the [README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/gitlabreceiver/README.md#tracing-limitations):

- The trace ID and pipeline span ID hash the pipeline ID with the pipeline's `finished_at`; stage
  and job span IDs hash their IDs with start times. A running job cannot know `finished_at`, so
  in-job telemetry cannot join the receiver's trace; emit it as a separate trace that carries the
  pipeline's `cicd.pipeline.run.id`.
- Child and multi-project pipelines become separate traces.
