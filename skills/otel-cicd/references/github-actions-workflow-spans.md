# GitHub Actions spans from the workflow itself

Facts for emitting pipeline-run and task spans from inside a GitHub Actions workflow, for when GitHub
cannot reach a Collector `github` receiver webhook. GitHub Actions has no built-in OTLP export, and
no OpenTelemetry-organization action exports workflow telemetry. Values hold for the named releases;
check the current release before relying on them.

- **Ordering.** A task span needs its parent's ID before the run span exists, so the trace ID and
  run span ID must be computed up front; the receiver's scheme
  ([github-receiver.md](github-receiver.md#joining-in-job-telemetry-to-the-receivers-trace)) is one
  option and keeps IDs compatible with a later move to the receiver. The run span (kind SERVER) can
  only end once the other jobs have finished, which a job with `needs:` on all of them and
  `if: always()` can observe. That job is still running when it ends the span, so its own remaining
  steps and result fall outside the run span; only an observer outside the run, such as the
  receiver's `workflow_run` event, sees the final conclusion.
- **Identity.** Task spans (kind INTERNAL): `cicd.pipeline.task.run.id` is `${{ job.check_run_id }}`
  and `cicd.pipeline.task.run.url.full` is
  `${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}/job/${{ job.check_run_id }}`.
  The run is `${{ github.run_id }}` and `.../actions/runs/${{ github.run_id }}`. These are string
  attributes. `job.check_run_id` is not available on GitHub Enterprise Server
  ([contexts reference](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts#job-context));
  there, take each job's `id` from the jobs API below.
- **Results.** `job.status` (inside a job) and `needs.<job>.result` (in a later job) map to the
  convention enum as `success` to `success`, `failure` to `failure`, `cancelled` to
  `cancellation`, `skipped` to `skip`. `error.type` on `failure` or `error` takes a low-cardinality
  class, or the conventions' fallback `_OTHER` when no class is known
  ([error registry](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/registry/attributes/error.md)).
- **Timing.** The run's `run_started_at` comes from `GET /repos/{owner}/{repo}/actions/runs/{run_id}`,
  and every job's `id`, `started_at`, `completed_at`, conclusion and steps from
  `GET /repos/{owner}/{repo}/actions/runs/{run_id}/attempts/{attempt_number}/jobs`, which returns
  30 jobs per page by default (`per_page` up to 100), so follow the pagination links. Both need
  `permissions: actions: read`. A timestamp taken in a job's first step misses runner setup and
  service containers.
- **Resource.** The pipeline-run resource (`cicd.pipeline.name`, `cicd.pipeline.run.id`,
  `cicd.pipeline.run.url.full`, `vcs.*`) belongs on every span of the run, including in-job
  processes, for example through `OTEL_RESOURCE_ATTRIBUTES`. Its `,` and `=` characters MUST be
  percent-encoded and others MAY be
  ([resource SDK](https://github.com/open-telemetry/opentelemetry-specification/blob/v1.63.0/specification/resource/sdk.md#specifying-resource-information-via-an-environment-variable)),
  but Ruby's `opentelemetry-sdk` 1.13.1 splits on `,` and `=` without percent-decoding
  ([resource.rb](https://github.com/open-telemetry/opentelemetry-ruby/blob/opentelemetry-sdk/v1.13.1/sdk/lib/opentelemetry/sdk/resources/resource.rb)),
  so values free of `,`, `=` and `%` work everywhere (spaces are fine).
- **A Collector inside a job.** It must be listening before senders start (poll its OTLP port or
  health check). An in-memory `sending_queue` gives data still queued at shutdown one send attempt and loses it if the Collector is killed; the exporter's
  `wait_for_result` makes senders wait for the export result instead (its limits are in the
  `otel-collector` skill's [exporter queue reference](../../otel-collector/references/exporterhelper.md#sending_queue)).
- **Tools.** Spans can come from an SDK, from OTLP/JSON posted with `curl`, or from `otel-cli`
  ([context-propagation.md](context-propagation.md#command-line-spans) covers its flags and the
  integer-typing trap).
