# `awss3` exporter: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

This recipe writes to a local S3-compatible store ([SeaweedFS](https://github.com/seaweedfs/seaweedfs) `weed mini`, which creates a user from `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` and pre-creates the bucket), with dummy credentials, and proves the key layout and one object per export request. Commands are bash.

Config (`s3-export.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
exporters:
  awss3:
    s3uploader:
      s3_bucket: telemetry
      s3_prefix: otel
      s3_partition_timezone: UTC
      endpoint: http://s3-store:8333   # scheme required
      s3_force_path_style: true
      compression: gzip
service:
  telemetry:
    logs:
      level: debug                     # logs each key as "uploading object"
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [awss3]
    logs:
      receivers: [otlp]
      exporters: [awss3]
```

Start the store and the Collector, then wait for both (`waitlog` gives up after 60s and prints the log tail):

```bash
docker network create s3verify
IMG=otel/opentelemetry-collector-contrib:0.165.0
docker run -d --name s3-store --network s3verify \
  -e AWS_ACCESS_KEY_ID=verify-key -e AWS_SECRET_ACCESS_KEY=verify-secret \
  chrislusf/seaweedfs:4.55 mini -dir=/data -bucket=telemetry
docker run -d --name s3-export --network s3verify \
  -e AWS_ACCESS_KEY_ID=verify-key -e AWS_SECRET_ACCESS_KEY=verify-secret \
  -v "$PWD/s3-export.yaml:/etc/otelcol-contrib/config.yaml" $IMG
waitlog() { for _ in $(seq 60); do docker logs "$1" 2>&1 | grep -q "$2" && return; sleep 1; done
  docker logs "$1" 2>&1 | tail -5; return 1; }
waitlog s3-store 'created bucket telemetry' && waitlog s3-export 'Everything is ready'
```

Send 5 traces and 7 logs (flags: the `otel-telemetrygen` skill's `references/flags.md`), then save the uploaded keys:

```bash
TG=ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.165.1
docker run --rm --network s3verify $TG traces --otlp-insecure \
  --otlp-endpoint s3-export:4317 --traces 5 --workers 1 --rate 1
docker run --rm --network s3verify $TG logs --otlp-insecure \
  --otlp-endpoint s3-export:4317 --logs 7 --workers 1 --rate 1 --body "s3 round trip"
waitlog s3-export '"key": "[^"]*logs_'
docker logs s3-export 2>&1 | grep -o '"key": "[^"]*"' | cut -d'"' -f4 | tee s3-keys.txt
docker logs s3-export 2>&1 | grep -c 'Exporting failed'
```

**What proves it worked:** one key per export request in the documented layout, and `0` failed exports. At `--rate 1` the traces usually went out one request each (a trace split across flushes adds a key) and the 7 logs in one:

```text
otel/year=2026/month=11/day=28/hour=05/minute=34/traces_632022536.json.gz
… four more traces_<n>.json.gz keys …
otel/year=2026/month=11/day=28/hour=05/minute=34/logs_712869168.json.gz
0
```

`uploading object` is logged before the upload; the [`aws_s3` receiver verification](../aws_s3/verification.md) proves the objects exist by reading them back from `s3-store` with `s3-keys.txt`. To continue there, keep everything running — its teardown removes it all. Otherwise:

```bash
docker rm -f s3-export s3-store
docker network rm s3verify
```
