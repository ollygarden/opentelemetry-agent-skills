# `awss3` exporter

| | |
|-|-|
| Kind | exporter |
| Type | `awss3` — not renamed at contrib v0.165.1, so `exporters: { aws_s3: … }` is an unknown type (the receiver is [`aws_s3`](../aws_s3/README.md)) |
| Signals | traces, metrics, logs (Alpha) |
| Distributions | contrib |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/exporter/awss3exporter> |

## Description

Writes telemetry to Amazon S3 or an S3-compatible store, **one object per export request**. The request is marshaled (`otlp_json` by default, `otlp_proto`, the logs-only `sumo_ic` and `body`, or an encoding extension), optionally `gzip`/`zstd`-compressed, and uploaded under a key partitioned by **upload** time ([layout](configuration.md#object-key-layout)). The bucket and the `s3_prefix` segment can come per request from resource attributes (`resource_attrs_to_s3`).

## Main use-cases

Use when:
- You archive raw telemetry in object storage for audit, compliance, or cheap long-term retention.
- You stage telemetry for later replay through the [`aws_s3` receiver](../aws_s3/README.md) (time-range backfill or SQS-driven ingestion).
- You feed data-lake tooling that reads OTLP JSON or Protobuf objects in time-partitioned (`year=/month=/…`) paths.

Avoid when:
- You need query-ready storage — this writes opaque objects with no indexing.

## Related components

- [`aws_s3` receiver](../aws_s3/README.md) — reads these objects back; [advanced.md](../aws_s3/advanced.md#exporter-to-receiver-settings) maps the settings that must match.
- `batch` processor / exporter `sending_queue.batch` — set object size and count, since each request is one object.
- [`file_storage`](../file_storage/README.md) — persistent `sending_queue` across restarts.

## Details

- [Configuration](configuration.md) — `s3uploader` and top-level keys, marshalers, object-key layout, validation.
- [Verification](verification.md) — local S3-compatible store plus `telemetrygen`; proves the key layout and one object per request.
- [Advanced use-cases](advanced.md) — per-tenant prefixes, S3-compatible endpoints, encoding extensions, one prefix per signal, many replicas.
- [Known quirks](quirks.md) — upload-time partitions, request-rate per prefix, routing by first resource, no permanent errors, marshaler crashes, compression headers.
