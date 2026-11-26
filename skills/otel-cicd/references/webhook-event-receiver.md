# Webhook event receiver (`webhook_event`)

Facts for the contrib [`webhookeventreceiver`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.166.0/receiver/webhookeventreceiver)
at collector-contrib [v0.166.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/CHANGELOG.md),
for keeping raw GitHub or GitLab webhook payloads as logs next to the `github` or `gitlab`
receiver's traces. Values hold for the named releases; check the current release before relying on
them.

## Identity

| Fact | Value at v0.166.0 | Source |
|---|---|---|
| Component type | `webhook_event`; `webhookevent` is a deprecated alias | [metadata.yaml](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/webhookeventreceiver/metadata.yaml); renamed in [v0.154.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/CHANGELOG.md#L3953) |
| Signals and stability | logs, beta | metadata.yaml; beta since [v0.133.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/CHANGELOG.md#L7508) |
| Distribution | `otelcol-contrib` only | metadata.yaml |

It is a generic receiver: it accepts any POST body and does not know GitHub or GitLab event types.

## What it emits

Each request becomes one log record whose string body is the whole request body, unless
`split_logs_at_newline` or `split_logs_at_json_boundary` is set. The body is not parsed. Query
parameters become resource attributes. Headers matching `header_attribute_regex` become
`header.<Canonical-Name>` log attributes whose values are string arrays (for example
`header.X-Github-Event` = `["workflow_run"]`). `convert_headers_to_attributes` is accepted but
has no effect at v0.166.0. Records carry an observed timestamp but no trace context
([req_to_log.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/webhookeventreceiver/req_to_log.go)).

Relating payloads to traces is one-to-many, because this receiver logs every delivery while the
traces receivers keep only finished runs:

- GitHub: the `github` receiver sets `cicd.pipeline.run.id` from `workflow_run.id` and
  `cicd.pipeline.task.run.id` from `workflow_job.id`. Earlier deliveries (`requested`, `queued`,
  `in_progress`) and re-run attempts share the ID, so match on the ID plus `action: completed`
  and `workflow_run.run_attempt` ([github-receiver.md](github-receiver.md#traces-from-webhooks)).
- GitLab: the `gitlab` receiver sets `cicd.pipeline.run.id` from `object_attributes.id`, which
  repeats on every pipeline status event, so also match `object_attributes.status` and
  `finished_at` ([gitlab-receiver.md](gitlab-receiver.md#spans)).
- The body is a string: extract the fields in the backend, or parse it with OTTL `ParseJSON` in a
  `transform` processor (`otel-ottl` skill). Compare parsed numbers with the receivers' integer
  IDs.

## Configuration

```yaml
# webhook_event receiver: otelcol-contrib only; logs beta at v0.166.0
receivers:
  webhook_event/github:
    endpoint: 0.0.0.0:8088              # required, no default
    path: /events                       # default
    health_path: /health_check          # default
    header_attribute_regex: "^X-Github-Event$"
    hmac_signature:                     # validates GitHub's signature
      secret: ${env:GITHUB_WEBHOOK_SECRET}
      header: X-Hub-Signature-256
      prefix: "sha256="
exporters:
  debug: {}                             # replace with otlp_grpc or otlp_http once verified
service:
  pipelines:
    logs:
      receivers: [webhook_event/github]
      exporters: [debug]
```

Source: [README configuration](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/webhookeventreceiver/README.md#configuration),
[config.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/webhookeventreceiver/config.go),
[factory.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/webhookeventreceiver/factory.go).

- `hmac_signature` (since [v0.157.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/CHANGELOG.md#L3292))
  checks an HMAC-SHA256 hex digest of the raw body and rejects missing or invalid signatures with
  401.
- `required_header` (`key`, `value`) requires an exact header value. GitLab sends its secret token
  in `X-Gitlab-Token`, so `{key: X-Gitlab-Token, value: ${env:GITLAB_WEBHOOK_TOKEN}}` authenticates
  GitLab deliveries. From [v0.166.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/CHANGELOG.md#L93)
  the value is redacted in printed configuration and compared in constant time. The code does not
  exclude it from header attributes, so `header_attribute_regex` must not match the
  `required_header` key (use `^X-Gitlab-Event$`, not `^X-Gitlab-.*`), or the token is stored.
- `max_request_body_size` defaults to 100 KiB in this receiver (the README's confighttp default is
  wrong), and larger bodies are rejected; raise it for large payloads such as GitLab pipeline
  events with many jobs.
- `read_timeout` and `write_timeout` default to 500ms, with a 10s maximum.
- It listens on its own endpoint, so GitHub or GitLab must deliver each event to it separately from
  the traces receiver. A GitHub App has one webhook URL, so the second delivery comes from an
  organization or repository webhook, or a second App.

The payloads hold the personal data that the traces receivers emit or keep behind options, and a
`resource` processor does not touch a log body ([SKILL.md](../SKILL.md#security)). Store them only
where that data may be kept, or rewrite the body with a `transform` processor.
