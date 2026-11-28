# `awss3` exporter: advanced use-cases

Release basis: [configuration.md](configuration.md).

## Per-tenant prefixes from a resource attribute

```yaml
processors:
  batch:
exporters:
  awss3:
    s3uploader:
      s3_bucket: telemetry-archive
      s3_base_prefix: env/prod
      s3_prefix: unassigned        # used when the attribute is absent
    resource_attrs_to_s3:
      s3_prefix: tenant.id
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [awss3]
```

Each object holds one tenant: `env/prod/<tenant.id>/year=…/logs_<n>.json`, as long as queue batching stays off and routing uses `s3_prefix`, not `s3_bucket`; `tenant.id` must be a string without `/` or `..` ([routing limits](quirks.md#routing-reads-the-first-resource-of-each-request)).

## S3-compatible stores

```yaml
exporters:
  awss3:
    s3uploader:
      region: us-east-1              # still signs requests
      s3_bucket: telemetry
      endpoint: http://object-store.internal:9000
      s3_force_path_style: true
      access_key: ${env:S3_ACCESS_KEY}
      secret_key: ${env:S3_SECRET_KEY}
```

`endpoint` needs a scheme ([quirks](quirks.md#no-error-is-permanent)). Over `http://` (or `disable_ssl: true`) requests are still signed, but the payload travels unencrypted.

## Encoding extensions

```yaml
extensions:
  text_encoding:
exporters:
  awss3:
    s3uploader:
      s3_bucket: telemetry
    encoding: text_encoding
    encoding_file_extension: txt
service:
  extensions: [text_encoding]
  pipelines:
    logs:
      receivers: [otlp]
      exporters: [awss3]
```

Keys end in `.txt`. Wire an encoding only into pipelines of signals it marshals — `text_encoding` marshals logs only.

## One prefix per signal for downstream readers

The signal name sits in the file name, not the prefix. A reader that filters by prefix — S3 event-notification filters feeding the [`aws_s3` receiver's SQS mode](../aws_s3/quirks.md#sqs-mode-ignores-the-signal-type) — needs separate instances:

```yaml
exporters:
  awss3/traces:
    s3uploader: { s3_bucket: telemetry, s3_prefix: traces }
  awss3/logs:
    s3uploader: { s3_bucket: telemetry, s3_prefix: logs }
service:
  pipelines:
    traces: { receivers: [otlp], exporters: [awss3/traces] }
    logs: { receivers: [otlp], exporters: [awss3/logs] }
```

## Many replicas writing one prefix

Key uniqueness rests on the [`<unique>`](configuration.md#object-key-layout) part, by default a random integer. Two uploads that draw the same integer under the same partition, signal, and `file_prefix` write the same key; without bucket versioning the later object replaces the earlier. `unique_key_func_name: uuidv7` uses time-ordered UUIDs instead; a per-replica `file_prefix` (for example `${env:HOSTNAME}-`) also separates writers and shows each one in the key.
