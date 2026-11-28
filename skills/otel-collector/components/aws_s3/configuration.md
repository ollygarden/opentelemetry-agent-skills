# `aws_s3` receiver: configuration

Facts trace to contrib **v0.165.1** [`receiver/awss3receiver`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/receiver/awss3receiver). Version notes come from the [contrib v0.165.1 changelog](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/CHANGELOG.md). Values hold for the named releases; check the current release before relying on them.

## Typical configs

Time range:

```yaml
receivers:
  aws_s3:
    starttime: "2026-11-01 00:00"
    endtime: "2026-11-02 00:00"
    s3downloader:
      region: eu-central-1
      s3_bucket: telemetry-archive
      s3_prefix: otel
      s3_partition_timezone: UTC
```

SQS:

```yaml
receivers:
  aws_s3/logs:
    s3downloader:
      region: eu-central-1
      s3_bucket: telemetry-archive
      s3_prefix: logs/
    sqs:
      queue_url: https://sqs.eu-central-1.amazonaws.com/123456789012/archive-logs
      region: eu-central-1
```

## Top-level keys

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `starttime` | string | — | Time-range mode. RFC3339, `YYYY-MM-DD HH:MM`, or `YYYY-MM-DD`; without an offset the value is UTC. Inclusive. |
| `endtime` | string | — | Same formats; exclusive (the walk runs while the step time is before it). |
| `sqs` | object | — | SQS mode; see [`sqs`](#sqs). Mutually exclusive with `starttime`/`endtime`. |
| `encodings` | list | — | `{extension: <id>, suffix: <string>}` entries; first entry whose suffix matches the key wins. See [Decoding](#decoding). |
| `notifications.opampextension` | component ID | — | OpAMP extension that receives time-range progress; see [advanced.md](advanced.md#opamp-progress-notifications). |

## `s3downloader`

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `s3_bucket` | string | — | Required. In SQS mode, records for any other bucket are skipped. |
| `region` | string | `us-east-1` | S3 client region. |
| `s3_prefix` | string | — | Time-range: first key segment, joined to the partition with one `/` (no trailing slash). SQS: plain string-prefix filter on the decoded key. The upstream README marks it required; `Validate()` does not. |
| `s3_partition_format` | string | `year=%Y/month=%m/day=%d/hour=%H/minute=%M` | Time-range only. Must change every minute, hour, or day; the step is the finest of those. |
| `s3_partition_timezone` | string | Local | IANA name used to format the partition. |
| `file_prefix` | string | — | Time-range only; listed after the partition. |
| `file_prefix_include_telemetry_type` | bool | `true` | Time-range only; appends `<signal>_` to `file_prefix`, matching the exporter's file names. |
| `endpoint` | string | — | Custom S3 API base URL; needs a scheme, e.g. `http://host:8333`. |
| `endpoint_partition_id` | string | `aws` | Parsed and defaulted; no code reads it at v0.165.1. |
| `s3_force_path_style` | bool | `false` | Path-style addressing. |
| `tag_object_after_ingestion` | bool | `false` | After a successful ingest, best-effort `PutObjectTagging` with `otel-collector:status=ingested`, replacing the object's existing tags; needs `s3:PutObjectTagging`. |
| `skip_ingesting_tagged_objects` | bool | `false` | `GetObjectTagging` before each fetch; skips objects with that tag. Needs `s3:GetObjectTagging`. |

Credentials always come from the AWS SDK default chain (environment, shared config, web identity, instance metadata); there are no key, role, or `disable_ssl` options.

## `sqs`

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `queue_url` | string | — | Required in SQS mode. |
| `region` | string | — | Required in SQS mode. |
| `endpoint` | string | — | Custom SQS API base URL. |
| `wait_time_seconds` | int | `20` | Long-poll wait, `0`–`20`. |
| `max_number_of_messages` | int | `10` | Messages per receive, `1`–`10`. |

The receiver never sets a visibility timeout; the queue's own setting decides when an undeleted message reappears. Message handling is in [quirks.md](quirks.md#sqs-delete-rules).

## Time-range walk

For each step from `starttime` while before `endtime`, the receiver lists:

```text
<s3_prefix>/<strftime(s3_partition_format, step in s3_partition_timezone)>/<file_prefix>[<signal>_]
```

and fetches every listed object in order. An empty `s3_prefix` drops the first segment. The window selects partitions by key, never by record timestamp; for the `awss3` exporter the partition is upload time ([why](../aws_s3_exporter/quirks.md#partitions-are-upload-time-not-telemetry-time)), so late data sits in later partitions. One `ListObjectsV2` call (plus pagination) runs per step and signal: a day of minute partitions is 1,440 list calls per signal.

## Decoding

1. A key ending in `.gz` is gunzipped and `.zst` is zstd-decoded, and that suffix is stripped.
2. The first `encodings` entry whose `suffix` ends the key decodes it, if its extension implements the signal's unmarshaler; later entries are not tried.
3. Otherwise `.json` → OTLP JSON, `.binpb` → OTLP Protobuf.
4. Otherwise the object is skipped with `Unsupported file format`.

## Validation

| Error | Cause |
|-------|-------|
| `bucket is required` | No `s3downloader.s3_bucket`. |
| `either starttime/endtime or sqs configuration must be provided` | Neither mode configured. |
| `when starttime is specified, endtime is required` (and the reverse) | Only one bound. |
| `starttime/endtime and sqs configuration cannot be used together` | Both modes. |
| `unable to parse starttime (…), accepted formats: …` | Bad time string. |
| `sqs.queue_url is required` / `sqs.region is required` | Incomplete `sqs`. |
| `sqs.wait_time_seconds must be between 0 and 20` | Out of range. |
| `sqs.max_number_of_messages must be between 1 and 10` | Out of range. |
| `invalid S3 partition timezone: …` | Unknown `s3_partition_timezone` in time-range mode, reported when the receiver is created. |
| `extension "<id>" not found` / `extension "<id>" does not exist` | An `encodings` / `notifications.opampextension` ID not running, reported at start. |
