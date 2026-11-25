# CI/CD semantic conventions

Facts at semantic-conventions
[v1.45.0](https://github.com/open-telemetry/semantic-conventions/releases/tag/v1.45.0).
Values hold for the named releases; check the current release before relying on them. For an
attribute these tables do not list, query the same release with the `otel-semantic-conventions`
skill (the tables cover the common questions without network access):

```bash
# from skills/otel-semantic-conventions/
OTEL_SEMCONV_TAG=v1.45.0 ./scripts/query-otel-semantic-conventions.sh cicd spans
OTEL_SEMCONV_TAG=v1.45.0 ./scripts/query-otel-semantic-conventions.sh vcs metrics
```

## Stability

The `cicd` and `vcs` conventions are **Release Candidate** at v1.45.0 (promoted from Development in
v1.43.0), except `vcs.change.time_to_approval`, `vcs.change.time_to_merge` and the `vcs.ref`
entity, which are Development. The `test`, `artifact` and `deployment` attributes are Development,
except `deployment.environment.name`, which is Stable.

## Spans

From [cicd-spans.md](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/cicd/cicd-spans.md):

| Span | Kind | Name | Required | Conditionally required | Opt-in |
|---|---|---|---|---|---|
| Pipeline run | SERVER | `{action} {pipeline}`, or `{action}` when no low-cardinality pipeline name exists | `cicd.pipeline.result` | `error.type` if the result is `failure` or `error` | `cicd.pipeline.action.name` |
| Pipeline task run | INTERNAL | no naming rule | `cicd.pipeline.task.name`, `cicd.pipeline.task.run.id`, `cicd.pipeline.task.run.url.full`, `cicd.pipeline.task.run.result` | `error.type` if the result is `failure` or `error` | none |

- `{action}` is `cicd.pipeline.action.name` (`BUILD`, `RUN`, `SYNC`) and `{pipeline}` is
  `cicd.pipeline.name`, for example `RUN Build and Test`.
- Set the task name, run ID and run URL at span creation; they are sampling-relevant.
- Both result attributes take `success`, `failure` (for example a test or compile failure),
  `error` (a CI/CD system fault), `timeout`, `cancellation` or `skip`.
- Span status follows the general recording-errors guidance.

## Resources

Pipeline and worker identity is resource data, not span attributes
([resource/cicd.md](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/resource/cicd.md)):

| Entity | Identifying | Descriptive |
|---|---|---|
| `cicd.pipeline` | `cicd.pipeline.name` | none |
| `cicd.pipeline.run` | `cicd.pipeline.run.id` | `cicd.pipeline.run.url.full` |
| `cicd.worker` | `cicd.worker.id` | `cicd.worker.name`, `cicd.worker.url.full` |
| `vcs.repository` | `vcs.repository.url.full` | `vcs.repository.name` |
| `vcs.ref` (Development) | `vcs.ref.head.revision` | `vcs.ref.head.name`, `vcs.ref.type` |

The conventions set the pipeline-run resource on all spans of a run, which means a TracerProvider
per pipeline run. Using the pipeline-run resource on metrics MUST be opt-in.

## Metrics

From [cicd-metrics.md](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/cicd/cicd-metrics.md):

| Metric | Instrument | Unit | Required attributes |
|---|---|---|---|
| `cicd.pipeline.run.duration` | histogram | `s` | `cicd.pipeline.name`, `cicd.pipeline.run.state`; `cicd.pipeline.result` once set; `error.type` on failure |
| `cicd.pipeline.run.active` | updowncounter | `{run}` | `cicd.pipeline.name`, `cicd.pipeline.run.state` |
| `cicd.pipeline.run.errors` | counter | `{error}` | `cicd.pipeline.name`, `error.type` |
| `cicd.worker.count` | updowncounter | `{worker}` | `cicd.worker.state` |
| `cicd.system.errors` | counter | `{error}` | `cicd.system.component`, `error.type` |

`cicd.pipeline.run.state` is `pending`, `executing` or `finalizing`; `cicd.worker.state` is
`available`, `busy` or `offline`. `cicd.system.errors` excludes pipeline-run errors such as test
failures. The ten `vcs.*` metrics all require `vcs.repository.url.full` except
`vcs.repository.count`; the `github` receiver's scraper emits them, all but `vcs.contributor.count`
by default ([github-receiver.md](github-receiver.md#metrics-from-the-scraper)).

## Related registries

- [VCS](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/registry/attributes/vcs.md):
  `vcs.repository.url.full` should not include `.git`; the deprecated `vcs.repository.ref.*` and
  `vcs.repository.change.*` keys were renamed to `vcs.ref.head.*` and `vcs.change.*`.
- [Test](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/registry/attributes/test.md):
  `test.suite.name`, `test.suite.run.status`, `test.case.name`, `test.case.result.status`.
- [Artifact](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/registry/attributes/artifact.md):
  `artifact.*` names, versions, hashes, purls and attestations.
- [Deployment](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/registry/attributes/deployment.md):
  `deployment.name`, `deployment.id`, `deployment.status`, `deployment.environment.name`.

Logs from pipelines follow the general log conventions and, when a trace context exists, should be
correlated with it ([cicd-logs.md](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/cicd/cicd-logs.md)).
