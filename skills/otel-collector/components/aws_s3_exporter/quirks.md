# `awss3` exporter: known quirks

Release basis: [configuration.md](configuration.md).

## Partitions are upload time, not telemetry time

The partition path is `s3_partition_format` applied to the wall clock at upload, in `s3_partition_timezone` (default: the process's local zone, UTC in the official image). Late, buffered, or retried data lands in the minute it was uploaded, so a time-range replay with the [`aws_s3` receiver](../aws_s3/README.md) selects by upload time (observed).

## One object per request

Each request reaching the exporter becomes one object (observed), uploaded with one `PutObject` below the S3 transfer manager's 16 MiB multipart threshold and as a multipart upload at or above it; a request that marshals to zero bytes uploads nothing and still counts as sent. Object size and count follow upstream batching (`batch` processor, or queue batching). All objects one prefix writes in one partition window share a key prefix and differ only in the file name; S3 applies request-rate limits per prefix ([Amazon S3 performance guidelines](https://docs.aws.amazon.com/AmazonS3/latest/userguide/optimizing-performance.html)). `s3_base_prefix`, `s3_prefix`, and `resource_attrs_to_s3.s3_prefix` are the levers that change the key prefix.

## Routing reads the first resource of each request

Each upload takes its bucket and prefix from the request's **first** resource. Only `resource_attrs_to_s3.s3_prefix` splits requests per value, in a wrapper in front of the exporterhelper queue. Queue batching ([`sending_queue.batch`](../otlp_exporter/configuration.md#sending_queue)) runs after that split and partitions only by client metadata, which this exporter does not inject, so it merges the split requests again. Observed:

- `s3_prefix` routing with `sending_queue.batch`: requests for `tenant-a`, `tenant-b`, and an untagged one became one object under `…/tenant-a/…`.
- `s3_prefix` routing behind a `batch` processor: one object per tenant (that batching happens before the split).
- `s3_bucket` routing alone: a request with resources for two buckets was written whole to the first resource's bucket.

## No error is permanent

The exporter wraps no marshaling, compression, or upload error as permanent, so [`retry_on_failure`](../otlp_exporter/configuration.md#retry_on_failure) retries every failed export until its `max_elapsed_time`, then drops the data. Inside each attempt the S3 client retries on its own (`retry_mode`, `retry_max_attempts`, `retry_max_backoff`), bounded by the per-attempt `timeout`; `retry_mode: nop` leaves retries to `retry_on_failure`. Cases that never succeed and are still retried:

- A scheme-less `endpoint` (`host:9000`): ``Custom endpoint `host:9000` was not a valid URI`` (observed). `disable_ssl: true` only rewrites `https://` to `http://`; it does not make a scheme-less endpoint valid.
- Denied access, a missing bucket, or expired credentials.

## A marshaler must cover every signal it receives

The exporter calls the marshaler for its pipeline's signal without checking that one exists, and `otelcol validate` accepts the config:

- `marshaler: body` sets only a logs marshaler, so the first traces or metrics export dereferences a nil marshaler and the Collector panics. `sumo_ic` in those pipelines fails at build instead ([validation](configuration.md#validation)).
- `encoding:` uses whichever marshaler interfaces the extension implements. With `text_encoding` (logs only) in a traces pipeline, the first trace export crashed the Collector with `panic: runtime error: invalid memory address or nil pointer dereference` in `(*s3Marshaler).MarshalTraces` (observed).

## Compression and `Content-Encoding`

Every format except `sumo_ic` is uploaded with `Content-Encoding: gzip` (or `zstd`), so HTTP clients that honor the header may decompress on download (observed); `sumo_ic` objects are stored compressed without it. The `aws_s3` receiver decompresses by key suffix either way.
