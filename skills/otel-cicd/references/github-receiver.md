# GitHub receiver (`github`)

Facts for the contrib [`githubreceiver`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.166.0/receiver/githubreceiver)
at collector-contrib [v0.166.0](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/CHANGELOG.md).
Values hold for the named releases; check the current release before relying on them.

- [Identity](#identity)
- [Traces from webhooks](#traces-from-webhooks)
- [Joining in-job telemetry to the receiver's trace](#joining-in-job-telemetry-to-the-receivers-trace)
- [Metrics from the scraper](#metrics-from-the-scraper)
- [Attribute differences from semantic conventions](#attribute-differences-from-semantic-conventions)

## Identity

| Fact | Value at v0.166.0 | Source |
|---|---|---|
| Component type | `github` | [metadata.yaml](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/metadata.yaml) |
| Signals and stability | traces alpha (since v0.151.0), metrics alpha | same |
| Distribution | `otelcol-contrib` only; not in `otelcol`, `otelcol-k8s` or `otelcol-otlp` | [otelcol-contrib manifest](https://github.com/open-telemetry/opentelemetry-collector-releases/blob/v0.166.0/distributions/otelcol-contrib/manifest.yaml) |
| Semantic conventions it emits | v1.43.0 (`sem_conv_version`) | metadata.yaml |

## Traces from webhooks

```yaml
# github receiver: otelcol-contrib only; traces and metrics alpha at v0.166.0
receivers:
  github:
    webhook:
      endpoint: 0.0.0.0:19418          # default localhost:8080 is loopback only
      path: /events                    # default
      health_path: /health             # default
      secret: ${env:GITHUB_WEBHOOK_SECRET}  # validates X-Hub-Signature-256
      service_name: ci                 # optional, see service.name below
    scrapers:                          # validation requires at least one scraper,
      scraper:                         # even for a traces-only receiver
        github_org: my-org
processors:
  resource/drop-personal:              # the receiver always emits these
    attributes:
      - {key: vcs.ref.head.revision.author.name, action: delete}
      - {key: vcs.ref.head.revision.author.email, action: delete}
      - {key: cicd.pipeline.run.sender.login, action: delete}
      - {key: cicd.pipeline.task.run.sender.login, action: delete}
exporters:
  debug:
    verbosity: detailed                # replace with otlp_grpc or otlp_http once verified
service:
  pipelines:
    traces:
      receivers: [github]
      processors: [resource/drop-personal]
      exporters: [debug]
```

Source: [README configuration](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/README.md#receiver-configuration),
[config.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/config.go),
[factory.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/factory.go).
The upstream README's `scraper/<name>` example keys are invalid: the only scraper key is `scraper`.

- `secret` validates `X-Hub-Signature-256` (SHA-1 `X-Hub-Signature` as fallback). Without a
  secret, deliveries are not validated.
- `read_timeout` and `write_timeout` default to 500ms; values above 10s are rejected.
- `required_headers` is checked before the signature. GitHub does not send custom headers, so it is
  for a proxy or WAF that adds one; it cannot override `X-GitHub-Event`, `X-GitHub-Delivery`,
  `X-GitHub-Hook-ID` or `X-Hub-Signature-256`.
- `include_span_events: true` adds the raw event JSON as span events (typically 5-50KB each). The
  payload includes the names and e-mail addresses that `resource/drop-personal` removes from the
  resource, and a resource processor does not filter span events, so keep it off unless that data
  may be stored.
- All `confighttp` server settings apply, including `tls`.
- One receiver instance covers one organization; use `github/<name>` instances for more
  ([named instances](../../otel-collector/SKILL.md#named-instances)).

GitHub side: the README documents a GitHub App subscribed to **Workflow runs** and **Workflow jobs**
(`workflow_run`, `workflow_job`), with the receiver URL and the shared secret. Deliveries must use
JSON or form content.

Event handling ([trace_receiver.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/trace_receiver.go)):
`workflow_run` and `workflow_job` produce spans only when `status` is `completed` (others return
204); `ping` returns 200; any other event returns 400. If `workflow_run` is not subscribed, job
spans point at a root span that never arrives.

Span structure ([trace_event_handling.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/trace_event_handling.go)):

| Span | Built from | Kind | Name | Time range |
|---|---|---|---|---|
| Run (root) | `workflow_run` | SERVER | workflow run name | `run_started_at` to `updated_at` |
| Job | `workflow_job` | INTERNAL | job name | `created_at` (includes queueing) to `completed_at` |
| Queue | `workflow_job` | INTERNAL | `queue-<job>` | `created_at` to `started_at` |
| Step | `workflow_job` steps | INTERNAL | step name (duplicates get `-n`) | step start to end |

Queue and step spans are siblings under the job span. A re-run (`run_attempt > 1`) is a new trace
with a link to the previous attempt's trace. Span status: `success` is OK, `failure` is ERROR,
other conclusions (cancelled, skipped, timed out) are UNSET.

`service.name` for traces comes from, in order: `webhook.service_name`, the repository custom
property `service_name`, the repository name, then `unknown_service`. Values are lowercased with
`_` replaced by `-`. Other repository custom properties become
`github.repository.custom_properties.<key>` resource attributes.

## Joining in-job telemetry to the receiver's trace

Trace and span IDs are derived deterministically, so a step can compute its own parent without an
API call ([README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/README.md#traces---getting-started)).
With the feature gate `receiver.githubreceiver.UseCheckRunID` (beta, enabled by default since
v0.151.0):

| Span | ID |
|---|---|
| Trace | `sha256("{run_id}{run_attempt}t")`, first 32 hex characters |
| Run (root) | `sha256("{run_id}{run_attempt}s")`, hex characters 17-32 |
| Job | `sha256("{check_run_id}-j")`, hex characters 17-32 |
| Step | `sha256("{check_run_id}-{step_name}-s")`, hex characters 17-32 |
| Queue | `sha256("{check_run_id}-q")`, hex characters 17-32 |

`check_run_id` is `${{ job.check_run_id }}`, which GitHub Enterprise Server does not provide, so
this helper does not work there. `step_name` is the step name the webhook reports, so
give the step an explicit `name:` (an unnamed step gets a generated one) and keep names unique
within the job, or steps share a span ID. `${{ }}` expressions are substituted only in workflow
YAML, so pass them to the shell through `env:`:

```yaml
- name: Integration tests
  env:
    CHECK_RUN_ID: ${{ job.check_run_id }}
    STEP_NAME: Integration tests            # must equal the step's name
  run: |
    trace_id=$(printf '%s' "${GITHUB_RUN_ID}${GITHUB_RUN_ATTEMPT}t" | sha256sum | cut -c1-32)
    span_id=$(printf '%s' "${CHECK_RUN_ID}-${STEP_NAME}-s" | sha256sum | cut -c17-32)
    export TRACEPARENT="00-${trace_id}-${span_id}-01"
    go test ./...
```

The process then extracts `TRACEPARENT` as its parent; if its SDK side is not set up yet, see
[context-propagation.md](context-propagation.md#sdk-carriers).
Disabling the gate (`--feature-gates=-receiver.githubreceiver.UseCheckRunID`) restores the legacy
scheme, which hashes `{run_id}{run_attempt}{job_name}…` and requires unique job names.

## Metrics from the scraper

The scraper polls GitHub's GraphQL and REST APIs; it needs no webhook.

```yaml
extensions:
  bearertokenauth/github:
    token: ${env:GH_PAT}
receivers:
  github:
    collection_interval: 300s
    scrapers:
      scraper:
        github_org: my-org
        search_query: "org:my-org topic:my-topic"   # optional
        auth:
          authenticator: bearertokenauth/github
exporters:
  debug: {}
service:
  extensions: [bearertokenauth/github]
  pipelines:
    metrics:
      receivers: [github]
      exporters: [debug]
```

Source: [README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/README.md),
[scraper factory](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/internal/scraper/githubscraper/factory.go),
[scraper README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/internal/scraper/githubscraper/README.md).

- `github_org` is required at scrape time; config validation does not check it.
- `search_query` defaults to `org:<github_org> archived:false` (or `user:` for a user account).
- `endpoint` is only for GitHub Enterprise Server: the server root, without `/api/graphql`.
- `collection_interval` defaults to 1m (the upstream README says 30s). GitHub's GraphQL limit is 5,000 points per hour per token
  and a scrape costs about 4 points per repository, so the scraper README gives
  `collection_interval (s) = 4n / (r / 3600)` for `n` repositories and `r` points per hour
  (40 repositories: about 115s), and suggests 300s as a sensible default. It also recommends at
  most 100 repositories per `search_query` and one token per receiver instance.
- `concurrency_limit` defaults to 50; `merged_pr_lookback_days` defaults to 30.
- Retries cover 429, 502, 503, 504, and 403 with `Retry-After`.
- The token needs read access to the target repositories; without it only `vcs.repository.count`
  is produced. Inside a workflow, `GITHUB_TOKEN` with `contents: read` and `pull-requests: read`
  can scrape its own repository (`search_query: repo:${{ github.repository }}`).
- Toggle metrics under `scrapers.scraper.metrics.<name>.enabled`.

| Metric | Unit | Default |
|---|---|---|
| `vcs.change.count` | `{change}` | on |
| `vcs.change.duration` | `s` | on (open changes only) |
| `vcs.change.time_to_approval` | `s` | on (only when approved changes exist) |
| `vcs.change.time_to_merge` | `s` | on (only when merged changes exist) |
| `vcs.ref.count` | `{ref}` | on |
| `vcs.ref.lines_delta` | `{line}` | on |
| `vcs.ref.revisions_delta` | `{revision}` | on |
| `vcs.ref.time` | `s` | on |
| `vcs.repository.count` | `{repository}` | on |
| `vcs.contributor.count` | `{contributor}` | **off** |

Source: [documentation.md](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/documentation.md).
All are gauges, each marked development in the receiver's per-metric metadata. Branch metrics skip the default branch
and branches without changes. The metric resource carries `vcs.provider.name` and `vcs.owner.name`
but no `service.name`.

## Attribute differences from semantic conventions

The receiver emits receiver-specific keys
([model.go](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.166.0/receiver/githubreceiver/model.go)).
Queries written against the released conventions ([semantic-conventions.md](semantic-conventions.md))
miss these:

| Receiver emits | Semantic conventions v1.45.0 |
|---|---|
| `cicd.pipeline.run.status` | `cicd.pipeline.result` |
| `cicd.pipeline.run.task.status` | `cicd.pipeline.task.run.result` |
| `cicd.pipeline.worker.*` | `cicd.worker.*` |
| `vcs.ref.head` (branch) | `vcs.ref.head.name` |
| run span named after the workflow run | `{action} {pipeline}` |
| `vcs.change.count`, `vcs.ref.count`, `vcs.repository.count` as gauges | updowncounters |

Run resources carry `cicd.pipeline.name` and `cicd.pipeline.run.id`. Job resources carry
`cicd.pipeline.name` set to the job name and `cicd.pipeline.task.run.id`, but no
`cicd.pipeline.run.id`; relate jobs to their run by trace ID.
