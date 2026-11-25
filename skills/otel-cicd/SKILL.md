---
name: otel-cicd
description: Observe CI/CD pipelines with OpenTelemetry. Use when turning GitHub Actions or GitLab CI/CD runs, jobs and steps into traces with the Collector `github` or `gitlab` receiver (workflow_run and pipeline webhooks, the GitHub scraper); collecting pull-request, branch and repository (VCS) or DORA metrics such as lead time for changes; joining build or test processes to the pipeline trace through `TRACEPARENT` or otel-cli; exporting OTLP from Jenkins, Argo Workflows, Tekton, Buildkite, Concourse or Dagger; or applying the `cicd.*` and `vcs.*` semantic conventions. Triggers on "trace our GitHub Actions workflows", "GitLab pipeline traces in the Collector", "make test spans join the CI trace", and "instrument our pipeline runner". Not for testing OpenTelemetry changes in CI, dependency-update automation, or deployment verification.
---

# OpenTelemetry for CI/CD pipelines

Use the `otel-collector` skill for other components and Collector-wide rules,
`otel-semantic-conventions` for full attribute definitions, and the language skills (`otel-go`,
`otel-python`, ...) for SDK setup.

## Choose the collection path

| What to observe | Path | Signals | Read |
|---|---|---|---|
| GitHub Actions runs, jobs, steps | Collector `github` receiver, webhook from a GitHub App | traces | [github-receiver.md](references/github-receiver.md) |
| GitHub pull requests, branches, repositories | Collector `github` receiver, scraper | metrics | [github-receiver.md](references/github-receiver.md#metrics-from-the-scraper) |
| Queue time, runner or worker load | `github` receiver queue spans, or `cicd.*` metrics | traces, metrics | [github-receiver.md](references/github-receiver.md#traces-from-webhooks), [semantic-conventions.md](references/semantic-conventions.md#metrics) |
| GitLab CI/CD pipelines, stages, jobs | Collector `gitlab` receiver, webhook | traces only | [gitlab-receiver.md](references/gitlab-receiver.md) |
| Commands inside a job (tests, builds, scripts) | SDK or CLI in the job, parent from `TRACEPARENT` | traces, logs, metrics | [context-propagation.md](references/context-propagation.md) |
| GitHub Actions runs when no webhook endpoint is reachable | spans emitted by the workflow itself | traces | [github-actions-workflow-spans.md](references/github-actions-workflow-spans.md) |
| CI/CD systems with built-in OTLP export (Jenkins, Argo Workflows, ...) | the tool's exporter | varies | [native-integrations.md](references/native-integrations.md) |
| A custom pipeline runner, or spans and metrics you emit yourself | an SDK following the conventions | traces, metrics | [semantic-conventions.md](references/semantic-conventions.md) |

Facts that most often go wrong, at collector-contrib v0.166.0 and semantic-conventions v1.45.0
(values hold for the named releases; check the current release before relying on them):

- Both receivers ship only in `otelcol-contrib`. `github` is alpha; `gitlab` is alpha in its
  metadata, but the v0.166.0 factory registers it as development. Webhooks are push-only: GitHub
  or GitLab must reach the Collector, and the default `localhost:8080` endpoint is loopback only.
- The `github` receiver needs a `scrapers` entry even for traces only.
- The `gitlab` receiver has no metrics, accepts only pipeline events, and compares `secret` with
  the `X-Gitlab-Token` header. No contrib receiver collects GitLab VCS metrics.
- The `github` receiver derives trace and span IDs deterministically, so a step can compute its own
  `TRACEPARENT` and join the receiver's trace. In-job telemetry cannot join the `gitlab`
  receiver's trace.
- The `github` receiver emits receiver-specific keys (for example `cicd.pipeline.run.status`
  instead of `cicd.pipeline.result`); the `gitlab` receiver uses the convention keys with raw
  GitLab values. Map them before writing queries against the released conventions.
- No SDK reads `TRACEPARENT` on its own: the child process extracts it with an environment
  carrier or a map built from the environment.
- The CI/CD conventions are Release Candidate, not Stable. Pipeline name and run ID are resource
  attributes, not span attributes.

## Workflow

1. **Pick the path.** Match the question to a row of the table above and read only the references
   it names.
2. **Configure the receiver or exporter.** Start from the reference's minimal configuration, and
   tell the user the component's stability and distribution, because alpha and development
   components can change between releases ([stability levels](../otel-collector/SKILL.md#stability-levels)). A custom distribution needs the receiver module and,
   for the GitHub scraper, a client auth extension such as `bearertokenauthextension`
   (`otel-collector-builder` skill).
3. **If in-job processes must join the trace,** read
   [context-propagation.md](references/context-propagation.md).
4. **If you emit spans or metrics yourself,** follow
   [semantic-conventions.md](references/semantic-conventions.md); run the `otel-semantic-conventions`
   script only for an attribute its tables do not list.
5. **Verify.** Send one finished pipeline through a `debug` exporter first and check the span tree
   (run, then job or stage, then step), the span statuses, the resource attributes, and that in-job
   spans share the run's trace ID.

## Security

- Webhook endpoints face the internet: set `secret`, put a WAF or equivalent in front as the
  `github` receiver README asks, and keep tokens and secrets in environment variables or a secret
  store.
- The `github` receiver always records the head commit's committer name and e-mail, under the
  `author` keys (`vcs.ref.head.revision.author.name`, `.email`), and the triggering user's login
  (`cicd.pipeline.run.sender.login`, `cicd.pipeline.task.run.sender.login`). Drop them with a
  `resource` or `attributes` processor if they must not be stored.
- Off by default: the `github` receiver's `include_span_events` copies whole event payloads, and
  the `gitlab` receiver's `include_user_attributes` adds author and actor names and e-mails.
- Environment variables are visible to other processes and logs; keep secrets out of `BAGGAGE`.
