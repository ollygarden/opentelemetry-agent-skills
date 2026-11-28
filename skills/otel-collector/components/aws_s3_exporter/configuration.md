# `awss3` exporter: configuration

Facts trace to contrib **v0.165.1** [`exporter/awss3exporter`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/exporter/awss3exporter) and core **v0.165.0** [exporterhelper](https://github.com/open-telemetry/opentelemetry-collector/tree/v0.165.0/exporter/exporterhelper/internal) defaults. Version notes come from the [contrib v0.165.1 changelog](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/CHANGELOG.md). Upload facts come from `feature/s3/transfermanager` v0.5.2, the version that release pins. Values hold for the named releases; check the current release before relying on them.

## Typical config

```yaml
exporters:
  awss3:
    s3uploader:
      region: eu-central-1
      s3_bucket: telemetry-archive
      s3_prefix: otel
      compression: gzip
    marshaler: otlp_proto
```

## `s3uploader` keys

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `region` | string | `us-east-1` | Must be non-empty. Also used to sign requests to an S3-compatible `endpoint`. |
| `s3_bucket` | string | — | Required unless `endpoint` is set. |
| `s3_base_prefix` | string | — | First key segment; never overridden. |
| `s3_prefix` | string | — | Second key segment. |
| `s3_partition_format` | string | `year=%Y/month=%m/day=%d/hour=%H/minute=%M` | strftime layout ([timefmt-go](https://github.com/itchyny/timefmt-go)) applied to the upload time. |
| `s3_partition_timezone` | string | Local (`time.Local`) | IANA name, e.g. `UTC`. An unknown name fails exporter start with `invalid S3 partition timezone`. |
| `file_prefix` | string | — | Prepended to the file name, before the signal name. |
| `endpoint` | string | — | Custom S3 API base URL (SDK `BaseEndpoint`); needs a scheme, e.g. `http://host:9000`. |
| `s3_force_path_style` | bool | `false` | Path-style addressing (`endpoint/bucket/key`); most S3-compatible stores need `true`. |
| `disable_ssl` | bool | `false` | Rewrites an `https://` request URL to `http://`. |
| `compression` | string | none | `gzip` or `zstd`; other values fail ([validation](#validation)). Adds `.gz` / `.zst` to the key. |
| `storage_class` | string | `STANDARD` | Validated against the AWS SDK `StorageClass` values ([S3 storage classes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/storage-class-intro.html)). |
| `acl` | string | unset | Canned ACL, validated against the SDK `ObjectCannedACL` values; not sent when empty. |
| `access_key` / `secret_key` | opaque string | — | Static credentials (v0.164.0+). Both or neither; not combinable with `role_arn`. |
| `role_arn` | string | — | Role assumed through STS; credentials are cached (v0.165.0+). |
| `retry_mode` | string | `standard` | SDK retryer: `standard`, `adaptive`, or `nop` (no SDK retries). |
| `retry_max_attempts` | int | `3` | SDK attempts per upload; `0` retries retryable errors without limit. |
| `retry_max_backoff` | duration | `20s` | SDK max backoff between attempts. |
| `unique_key_func_name` | string | random integer | `uuidv7` is the only other value; falls back to the random integer if UUID generation fails. |

Without `access_key`/`secret_key` or `role_arn`, the exporter uses the AWS SDK default credential chain (environment, shared config, web identity, instance metadata).

## Top-level keys

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `marshaler` | string | `otlp_json` | See [Marshalers](#marshalers). Unknown names fail start with `unknown marshaler`. |
| `encoding` | component ID | — | Encoding extension; overrides `marshaler`. |
| `encoding_file_extension` | string | — | Key suffix (without the dot) used with `encoding`; empty adds no suffix. |
| `resource_attrs_to_s3.s3_bucket` | string | — | Resource attribute whose value replaces `s3_bucket`. |
| `resource_attrs_to_s3.s3_prefix` | string | — | Resource attribute whose value replaces `s3_prefix`. Both read the request's first resource ([routing](quirks.md#routing-reads-the-first-resource-of-each-request)). |
| `timeout` | duration | `5s` | Per export attempt (`exporterhelper.TimeoutConfig`, squashed). |
| `sending_queue` | object | enabled (the upstream README's `disabled` is stale) | Core v0.165.0 `NewDefaultQueueConfig`. Fields: [`sending_queue`](../otlp_exporter/configuration.md#sending_queue). |
| `retry_on_failure` | object | enabled | Fields: [`retry_on_failure`](../otlp_exporter/configuration.md#retry_on_failure). |

## Marshalers

| `marshaler` | Key extension | Signals | Content |
|-------------|---------------|---------|---------|
| `otlp_json` | `.json` | traces, metrics, logs | One OTLP JSON request per object. |
| `otlp_proto` | `.binpb` | traces, metrics, logs | One OTLP Protobuf message per object. |
| `sumo_ic` | `.json` | logs | Sumo Logic archive lines; needs `_sourceCategory`, `_sourceHost`, `_sourceName` resource attributes. |
| `body` | `.txt` | logs | One log body per line (`AsString()`). In a traces or metrics pipeline it [crashes the Collector](quirks.md#a-marshaler-must-cover-every-signal-it-receives). |
| `encoding: <id>` | `encoding_file_extension` | whatever the extension marshals | See [encoding extensions](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/extension/encoding). A signal the extension cannot marshal [crashes the Collector](quirks.md#a-marshaler-must-cover-every-signal-it-receives). |

## Object key layout

Segments are joined with `path.Join`:

```text
[s3_base_prefix/][prefix/]<strftime(s3_partition_format, now in s3_partition_timezone)>/<file_prefix><signal>_<unique><.ext><.gz|.zst>
```

- `prefix` is the `resource_attrs_to_s3.s3_prefix` attribute value when present, else `s3_prefix`.
- `now` is the upload wall-clock time (`clock.Now`), not any telemetry timestamp.
- `<signal>` is `traces`, `metrics`, or `logs`.
- `<unique>` is a random integer in `[100000000, 999999999)` or, with `unique_key_func_name: uuidv7`, a UUIDv7.

Example with `s3_base_prefix: env/prod`, `s3_prefix: otel`, `file_prefix: node-a-`, `marshaler: otlp_proto`, `compression: zstd`, defaults otherwise:

```text
env/prod/otel/year=2026/month=11/day=28/hour=05/minute=20/node-a-logs_731555892.binpb.zst
```

## Validation

| Error | Cause |
|-------|-------|
| `region is required` | `region: ""`. |
| `bucket or endpoint is required` | Neither `s3_bucket` nor `endpoint` set. |
| `invalid StorageClass` | Value outside the SDK list, including `storage_class: ""`. |
| `invalid ACL` | Non-empty value outside the SDK canned-ACL list. |
| `unknown compression type` | `zlib`, `deflate`, `snappy`, `x-snappy-framed`, or `lz4`. |
| `unsupported compression type "<value>"` | Unknown name, such as `br`; reported while decoding the config. |
| `invalid retry mode, must be either 'standard', 'adaptive' or 'nop'` | Bad `retry_mode`. |
| `invalid UniqueKeyFuncName` | Non-empty value other than `uuidv7`. |
| `both access_key and secret_key must either be set together or left empty` | Only one static key. |
| `cannot use both static credentials (access_key/secret_key) and role_arn` | Static keys plus `role_arn`. |
| `traces are not supported by sumo_ic output format` (also `metrics`) | `sumo_ic` in a traces or metrics pipeline; reported at pipeline build, which `otelcol validate` runs. |
