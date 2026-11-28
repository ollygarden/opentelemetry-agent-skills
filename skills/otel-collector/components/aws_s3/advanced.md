# `aws_s3` receiver: advanced use-cases

Release basis: [configuration.md](configuration.md).

## Exporter-to-receiver settings

The receiver reads the [`awss3` exporter](../aws_s3_exporter/README.md)'s `otlp_json` and `otlp_proto` objects, compressed or not, when these match:

| Exporter `s3uploader` / top level | Receiver `s3downloader` / top level |
|-----------------------------------|-------------------------------------|
| `s3_bucket` | `s3_bucket` |
| `s3_base_prefix` + `s3_prefix` | `s3_prefix: <base>/<prefix>` — the receiver has no base prefix |
| `resource_attrs_to_s3.s3_prefix` value | one receiver instance per value, `s3_prefix: <base>/<value>` |
| `s3_partition_format` | `s3_partition_format` (time-range mode; minute, hour, or day granularity) |
| `s3_partition_timezone` | `s3_partition_timezone` |
| `file_prefix` | `file_prefix`, with `file_prefix_include_telemetry_type: true` (default) |
| `marshaler: otlp_json` / `otlp_proto` | built in (`.json` / `.binpb`) |
| `compression: gzip` / `zstd` | automatic (`.gz` / `.zst`) |
| `encoding: <id>` + `encoding_file_extension: <ext>` | `encodings: [{extension: <id>, suffix: .<ext>}]` |
| `endpoint`, `s3_force_path_style` | `endpoint`, `s3_force_path_style` |
| `disable_ssl` | none — use an `http://` endpoint |
| `access_key` / `secret_key` / `role_arn` | none — default credential chain |

`sumo_ic` objects (`.json`, not OTLP) decode without error as an empty request, so nothing is ingested ([delete rules](quirks.md#sqs-delete-rules)). `body` objects (`.txt`) are skipped unless an encoding handles `.txt`; with `text_encoding` the bodies come back as log records without the original resource, attributes, or timestamps.

## SQS notification layout

SQS mode needs the bucket to publish `s3:ObjectCreated:*` events to the queue, directly or through an SNS topic. Setup, permissions, and prefix/suffix filters: [Amazon S3 event notifications](https://docs.aws.amazon.com/AmazonS3/latest/userguide/EventNotifications.html).

Give each signal its own exporter prefix, notification filter, queue, and receiver ID ([delete rules](quirks.md#sqs-delete-rules)):

```yaml
receivers:
  aws_s3/traces:
    s3downloader: { s3_bucket: telemetry, s3_prefix: traces/ }
    sqs: { queue_url: "https://sqs.eu-central-1.amazonaws.com/123456789012/telemetry-traces", region: eu-central-1 }
  aws_s3/logs:
    s3downloader: { s3_bucket: telemetry, s3_prefix: logs/ }
    sqs: { queue_url: "https://sqs.eu-central-1.amazonaws.com/123456789012/telemetry-logs", region: eu-central-1 }
service:
  pipelines:
    traces: { receivers: [aws_s3/traces], exporters: [otlp_grpc] }
    logs: { receivers: [aws_s3/logs], exporters: [otlp_grpc] }
```

The matching exporter side is in [the exporter's advanced.md](../aws_s3_exporter/advanced.md#one-prefix-per-signal-for-downstream-readers). A redrive policy moves an undeletable message to a [dead-letter queue](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-dead-letter-queues.html); an ingest that outlasts the visibility timeout is delivered again, possibly to another replica ([visibility timeout](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)).

## Tagging as a checkpoint

```yaml
receivers:
  aws_s3:
    starttime: "2026-11-01"
    endtime: "2026-11-08"
    s3downloader:
      s3_bucket: telemetry-archive
      s3_prefix: otel
      tag_object_after_ingestion: true
      skip_ingesting_tagged_objects: true
```

A restarted replay or a redelivered SQS message then skips already-ingested objects, including ones whose data was never delivered ([why](quirks.md#the-ingested-tag-marks-acceptance-not-delivery)). A failed `PutObjectTagging` is only logged, so that object is ingested again next time. The `otel-collector:status=ingested` tag can also drive an S3 lifecycle rule that expires or transitions ingested objects.

## OpAMP progress notifications

```yaml
extensions:
  opamp:
    server:
      ws:
        endpoint: wss://opamp.example.com/v1/opamp
receivers:
  aws_s3:
    starttime: "2026-11-01"
    endtime: "2026-11-02"
    s3downloader: { s3_bucket: telemetry-archive, s3_prefix: otel }
    notifications:
      opampextension: opamp
service:
  extensions: [opamp]
```

Time-range mode only. The receiver sends `TimeBasedIngestStatus` messages on the custom capability `org.opentelemetry.collector.receiver.awss3`, with `ingest_status` `ingesting` before each step, `completed` at the end, or `failed` on error. Payload fields: upstream [Notifications](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/receiver/awss3receiver#notifications).
