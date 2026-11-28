# `aws_s3` receiver

| | |
|-|-|
| Kind | receiver |
| Type | `aws_s3` |
| Signals | traces, metrics, logs (Alpha) |
| Distributions | contrib |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awss3receiver` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/receiver/awss3receiver> |
| Rename | `awss3` → `aws_s3` in v0.162.0; the old name still works as a deprecated alias ([renames](../../SKILL.md#recent-renames)). |

## Description

Reads telemetry objects from Amazon S3 or an S3-compatible store into a pipeline, in exactly one of two modes: **time range** (`starttime` + `endtime`) walks the partition paths the [`awss3` exporter](../aws_s3_exporter/README.md) writes, once, then idles; **SQS** (`sqs:`) long-polls S3 `ObjectCreated:*` notifications (direct or SNS-wrapped) and fetches each referenced object. Objects decode by key suffix after `.gz` / `.zst` decompression ([decoding](configuration.md#decoding)).

## Main use-cases

Use when:
- You backfill or replay archived telemetry for a past window into a pipeline.
- You ingest telemetry continuously as producers drop objects into a bucket, driven by S3 → SQS notifications.

Avoid when:
- One SQS queue would feed pollers that differ in signal, bucket, or prefix — each deletes messages meant for the others ([quirks](quirks.md#sqs-delete-rules)).
- Objects exceed what the Collector can hold in memory ([quirks](quirks.md#whole-objects-in-memory)).
- You need checkpointed, resumable time-range replay — a restart re-reads the range unless objects are tagged.

## Related components

- [`awss3` exporter](../aws_s3_exporter/README.md) — the writer; [advanced.md](advanced.md#exporter-to-receiver-settings) maps the settings that must match.

## Details

- [Configuration](configuration.md) — keys per mode, defaults, time-range walk, decoding order, validation.
- [Verification](verification.md) — time-range replay and SQS ingestion against local stores.
- [Advanced use-cases](advanced.md) — exporter-to-receiver settings, SQS notification layout, tagging, OpAMP progress.
- [Known quirks](quirks.md) — shared-queue data loss, SQS delete rules, tagging before delivery, a silent one-shot reader, prefix slashes, memory, log sampling.
