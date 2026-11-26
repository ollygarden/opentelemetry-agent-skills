# CI/CD systems with built-in OpenTelemetry export

Open-source CI/CD systems and build tools that export OTLP themselves. Values hold for the named
releases; check the current release before relying on them. None of them follows the `cicd.*` span
conventions at these releases, so map their attributes before building queries on the released
conventions ([semantic-conventions.md](semantic-conventions.md)).

| System | Release | Exports | Conventions | Configuration | Injects `TRACEPARENT` into steps |
|---|---|---|---|---|---|
| Jenkins [`opentelemetry-plugin`](https://github.com/jenkinsci/opentelemetry-plugin/blob/3.1605.v37c3e29857e5/docs/job-traces.md) | 3.1605.v37c3e29857e5 | traces (pipeline, stage, step spans), metrics, build logs | legacy `ci.pipeline.*` span attributes; metrics use `cicd.pipeline.run.duration`, `cicd.pipeline.run.active`, `cicd.worker.count`, `cicd.pipeline.run.errors`, `cicd.system.errors` ([CicdMetrics.java](https://github.com/jenkinsci/opentelemetry-plugin/blob/3.1605.v37c3e29857e5/src/main/java/io/jenkins/plugins/opentelemetry/semconv/CicdMetrics.java)) | plugin settings or JCasC, plus `OTEL_*` autoconfiguration ([setup](https://github.com/jenkinsci/opentelemetry-plugin/blob/3.1605.v37c3e29857e5/docs/setup-and-configuration.md)); can pass `OTEL_EXPORTER_OTLP_*` to builds | yes, in `sh`, `bat` and `powershell` steps (`TRACEPARENT`, `TRACESTATE`, plus `TRACE_ID`, `SPAN_ID`) |
| [Argo Workflows](https://github.com/argoproj/argo-workflows/blob/v4.1.6/docs/tracing.md) | v4.1.6 | traces (workflow, node, container; Beta since v4.1.0), metrics (since v3.6.0) | own attributes | standard `OTEL_EXPORTER_OTLP_*` ([telemetry configuration](https://github.com/argoproj/argo-workflows/blob/v4.1.6/docs/telemetry-configuration.md)) | yes, through the contrib `envcar` carrier |
| [Buildkite agent](https://github.com/buildkite/agent/releases/tag/v4.12.0) | v4.12.0 | job traces, agent metrics, job logs | `buildkite.*` attributes | `--opentelemetry-tracing`, `--opentelemetry-metrics`, `--job-logs-otlp`, plus `OTEL_EXPORTER_OTLP_*` | yes, normalized per the environment-carrier specification |
| [Concourse](https://github.com/concourse/concourse/releases/tag/v8.3.4) | v8.3.4 | traces only (experimental), OTLP gRPC | own attributes | `CONCOURSE_TRACING_OTLP_ADDRESS`, `CONCOURSE_TRACING_OTLP_HEADERS`, `CONCOURSE_TRACING_OTLP_USE_TLS` | yes, in `run` commands |
| [Dagger](https://github.com/dagger/dagger/releases/tag/v0.21.12) | v0.21.12 | traces, logs, metrics | own attributes | standard `OTEL_EXPORTER_OTLP_*` | yes; it also extracts `TRACEPARENT` from its own environment, so a Dagger call joins the calling pipeline's trace |
| [Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/v1.18.0/docs/developers/tracing.md) | v1.18.0 | traces of the PipelineRun and TaskRun reconcilers (OTLP/HTTP), OpenTelemetry metrics | `pipelinerun`, `taskrun`, `namespace` attributes | `config-tracing` ConfigMap (`enabled`, `endpoint`, `credentialsSecret`) | no; context moves between runs through `tekton.dev/*SpanContext` annotations |
| [Maven extension](https://github.com/open-telemetry/opentelemetry-java-contrib/blob/v1.62.1/maven-extension/README.md) (java-contrib) | v1.62.1 | traces of Maven goal executions | own attributes | `OTEL_TRACES_EXPORTER=otlp` is required (the default `none` exports nothing), plus `OTEL_EXPORTER_OTLP_*` | extracts `TRACEPARENT` from the environment, so a build joins the pipeline's trace |

Not OTLP to an arbitrary backend:

- **GitLab:** GitLab's built-in CI/CD observability (`GITLAB_OBSERVABILITY_EXPORT`, Beta) sends
  only to GitLab Observability ([docs](https://docs.gitlab.com/operations/observability/ci_cd/)).
  For a Collector, use the `gitlab` receiver ([gitlab-receiver.md](gitlab-receiver.md)).
- **GitHub Actions:** no built-in export; use the `github` receiver ([github-receiver.md](github-receiver.md))
  or emit spans from the workflow ([github-actions-workflow-spans.md](github-actions-workflow-spans.md)).
