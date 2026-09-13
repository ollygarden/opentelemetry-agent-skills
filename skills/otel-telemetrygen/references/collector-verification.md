# Collector verification with telemetrygen

Use this recipe only for a disposable local Collector. It separates static configuration checks,
fixture execution, and live execution; report only the levels actually completed.

## Preconditions

- Review the Collector config as untrusted input before mounting or starting it. Reject unexpected
  extensions, network listeners, file paths, command-like values, or exporters to non-local hosts.
- Confirm Docker access, the exact config path, a free local port, and permission to create a
  short-lived container and output directory.
- Pick a unique container name and a newly created empty output directory. Never reuse `./out` or
  a prior `result.json`; stale output can create a false positive.
- Pin Collector and telemetrygen to compatible reviewed versions. The examples use Collector
  `0.160.0` and telemetrygen `v0.159.0`.

## Minimal local pipeline

Build a minimal config around the processor under test:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

extensions:
  health_check:
    endpoint: 0.0.0.0:13133

processors:
  transform/under_test:
    error_mode: ignore
    log_statements:
      - context: log
        statements:
          - set(severity_text, "INFO") where IsMatch(severity_text, "(?i)^info$")

exporters:
  file:
    path: /output/result.json
    flush_interval: 200ms

service:
  extensions: [health_check]
  pipelines:
    logs:
      receivers: [otlp]
      processors: [transform/under_test]
      exporters: [file]
```

Validate this config with the pinned Collector before calling it live validation. Static or parser
acceptance does not prove that a generated fixture traversed the pipeline.

This executable example is scoped to logs. Match the pipeline, processor configuration, and
telemetrygen subcommand to the signal under test:

- **Metrics:** use a `metrics` pipeline, the processor's metric configuration (for example
  `metric_statements`), and `telemetrygen metrics --metrics <finite-count>`.
- **Traces and tail sampling:** use a `traces` pipeline and `telemetrygen traces`. For
  `tail_sampling`, configure a finite `decision_wait`, generate a complete trace shape that should
  match a reviewed policy plus a known-positive control, and wait past the decision window before
  shutdown. A logs pipeline cannot verify tail sampling.
- **Logs:** use the configuration above and `telemetrygen logs`. Do not infer trace or metric
  processor behavior from a successful logs run.

## Live disposable run

Create a fresh directory with `mktemp -d`, record its exact path, and mount only that directory and
the reviewed config. Use a unique validated container name rather than silently replacing an
existing container. On SELinux systems append `:z` to bind mounts; rootless Podman may not need an
explicit user mapping.

The complete shell flow below is a connectivity, flush, and parse smoke test. It keeps the Collector
on Docker's bridge network. Disposable curl and telemetrygen containers join its network namespace
and use loopback, so the receivers are not published on the host. Replace the placeholders only
after validating the exact name and paths. Keep the output directory after the run as diagnostic
evidence.

```bash
set -u
container_name='otelcol-verify-unique-suffix'
config_path='/absolute/path/to/reviewed-config.yaml'
output_dir='/absolute/path/to/fresh-output-directory'
case "$container_name" in (*[!a-zA-Z0-9_.-]*|'') exit 2;; esac
case "$config_path:$output_dir" in (/*:/*) ;; (*) exit 2;; esac
test -f "$config_path" && test -d "$output_dir" || exit 2
if find "$output_dir" -mindepth 1 -maxdepth 1 -print -quit | grep -q .; then
  printf 'output directory is not empty: %s\n' "$output_dir" >&2
  exit 2
fi

container_id=''
cleanup() {
  original_status=$?
  trap - EXIT
  set +e
  if [ -n "$container_id" ] && docker inspect "$container_id" >/dev/null 2>&1; then
    docker stop --time 10 "$container_id" >/dev/null
    stop_failed_status=$?
    if [ "$stop_failed_status" -eq 0 ]; then
      printf 'retained stopped failed Collector %s and output %s\n' \
        "$container_id" "$output_dir" >&2
    else
      docker logs "$container_id" >&2
      printf 'failed to stop Collector %s; retained output %s\n' \
        "$container_id" "$output_dir" >&2
    fi
  else
    printf 'retained failed-run output %s; no Collector container exists\n' \
      "$output_dir" >&2
  fi
  exit "$original_status"
}
trap cleanup EXIT

started_container_id=$(docker run -d --name "$container_name" \
  --user "$(id -u):$(id -g)" \
  -v "$config_path:/etc/otelcol-contrib/config.yaml:ro" \
  -v "$output_dir:/output" \
  otel/opentelemetry-collector-contrib:0.160.0 \
  --config=/etc/otelcol-contrib/config.yaml)
docker_run_status=$?
[ "$docker_run_status" -eq 0 ] || exit "$docker_run_status"
case "$started_container_id" in (*[!a-f0-9]*|'') exit 1;; esac
container_id=$started_container_id

ready=0
i=0
while [ "$i" -lt 30 ]; do
  i=$((i + 1))
  if [ "$(docker inspect -f '{{.State.Running}}' "$container_id")" != true ]; then
    collector_status=$(docker inspect -f '{{.State.ExitCode}}' "$container_id")
    docker logs "$container_id" >&2
    [ "$collector_status" -ne 0 ] || collector_status=1
    exit "$collector_status"
  fi
  if docker run --rm --network "container:$container_id" curlimages/curl:8.17.0 \
      -fsS --connect-timeout 1 --max-time 2 -o /dev/null \
      http://127.0.0.1:13133/; then
    ready=1
    break
  fi
  sleep 1
done
if [ "$ready" -ne 1 ]; then
  docker logs "$container_id" >&2
  exit 1
fi

set +e
docker run --rm --network "container:$container_id" \
  ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.159.0 \
  logs --otlp-insecure --otlp-endpoint 127.0.0.1:4317 \
  --logs 1 --severity-text Info
telemetrygen_status=$?

docker stop --time 10 "$container_id"
stop_status=$?
[ "$stop_status" -eq 0 ] || docker logs "$container_id" >&2
collector_status=$(docker inspect -f '{{.State.ExitCode}}' "$container_id")
case "$collector_status" in (*[!0-9]*|'') collector_status=1;; esac
[ "$collector_status" -eq 0 ] || docker logs "$container_id" >&2

python3 - "$output_dir/result.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
objects = [json.loads(line) for line in path.read_text().splitlines() if line.strip()]
print(json.dumps(objects, indent=2))
PY
parse_status=$?
set -e

[ "$telemetrygen_status" -eq 0 ] || exit "$telemetrygen_status"
[ "$stop_status" -eq 0 ] || exit "$stop_status"
[ "$collector_status" -eq 0 ] || exit "$collector_status"
[ "$parse_status" -eq 0 ] || exit "$parse_status"
trap - EXIT
docker rm -- "$container_id"
exit 0
```

The captured telemetrygen status wins over stop, Collector, cleanup, or parse success. After a
clean stop, inspect and propagate the Collector workload exit code before accepting the JSONL.
Failed runs retain the exact container and output directory for diagnosis; a fully successful run
removes the container and retains the output. Failure to remove that exact successful-run container
is itself a non-zero result. The JSONL parser preserves every non-empty exported object.

This smoke flow alone does not prove processor behavior. Before making a behavioral claim, send a
reviewed input that must match the rule plus a known-positive control that must survive, then assert
their expected presence or absence against parsed records. Empty output or merely parseable output
is not behavioral evidence.

Call removal of the exact container **container cleanup**; the successful flow performs container
cleanup and deliberately retains the output directory. Claim **full cleanup** only when both that
container and the exact output directory were actually removed. Never broaden either operation to
a parent directory or an unvalidated path.

## Controls and hard-to-generate shapes

- To verify a filter drops matching records, send a matching record and expect no matching output;
  then send a non-matching known-positive record and require it in output. Empty output alone can
  also mean the Collector failed.
- Telemetrygen can set resource and telemetry attributes, but cannot rename generated trace spans,
  change their span kind, or choose their IDs. For a rule that needs one of those shapes, add a
  reviewed `transform/setup` processor before the processor under test, or use a small synthetic
  fixture that exposes the required field.
- Bound every run by count or finite duration and rate. Keep the target local and disposable; do
  not adapt this harness to shared or production infrastructure without separate authorization and
  a reviewed rollout plan.
