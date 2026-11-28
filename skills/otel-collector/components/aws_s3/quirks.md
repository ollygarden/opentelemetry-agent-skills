# `aws_s3` receiver: known quirks

Release basis: [configuration.md](configuration.md).

## SQS mode ignores the signal type

This factory does not use `sharedcomponent`, so one `aws_s3` ID in a `traces` and a `logs` pipeline is two SQS pollers on one queue ([named instances](../../SKILL.md#named-instances)). In SQS mode a poller takes **any** `ObjectCreated` record under `s3_prefix`, whatever signal the object holds. In a shared-queue run, the traces poller decoded a logs object as `"spans": 0` and deleted the message, so those logs were lost (observed). Time-range mode filters by signal only through the `<signal>_` file-name segment, so only with `file_prefix_include_telemetry_type: true` (default). Queue layout: [advanced.md](advanced.md#sqs-notification-layout).

## SQS delete rules

A message is deleted unless one of its records fails — and skipping is not failing. Handled, so deleted:

- records that are not `aws:s3` `ObjectCreated:*`, name another bucket, or fall outside `s3_prefix`, and `s3:TestEvent` messages (observed);
- `NoSuchKey` (`Failed to get S3 object`, observed), an unsupported suffix, an already-tagged object;
- an object that decodes as an empty request: one of another signal (observed), or a `sumo_ic` object.

Failed, so the **whole** message is kept and redelivered after the queue's visibility timeout, re-ingesting its records that had succeeded unless tagging skips them: decode errors (`Message not deleted due to processing failures, will be retried after visibility timeout`, observed), and fetch, tag-check, and pipeline errors. A body that is not JSON (`Failed to parse message as SNS notification`, observed), JSON that is neither an S3 event nor an SNS `Notification` (`Message is not a valid S3 notification`), or an SNS message whose `Message` does not parse (`Failed to parse S3 event from SNS message`) is never deleted by the receiver and returns every visibility timeout. Delete failures are only logged.

SQS hands each message to whichever poller receives it, so every poller on a queue must be able to ingest every object the queue announces: identical replicas are safe; pollers that differ in signal, bucket, or `s3_prefix` delete each other's data.

## A silent, one-shot reader

`Start` runs the reader once in a goroutine and discards its error, and the receiver reports no component status. Time-range mode returns on its first error, and the rest of the range is never read until a restart, which walks the whole range again (re-ingesting it unless `skip_ingesting_tagged_objects` is on). That error can be:

- an `s3_partition_format` coarser than a day (`"%Y/%m"`): nothing is logged — not even `Start reading telemetry` — and nothing is read (observed); day granularity works since v0.162.0;
- a scheme-less `endpoint` (`host:8333`): ``Custom endpoint `host:8333` was not a valid URI`` under `Error reading telemetry` (observed);
- any list, fetch, tag-check, decode, or pipeline error. This receiver does not retry refused data, so a [`memory_limiter`](../memory_limiter/quirks.md#refusal-is-non-permanent--receivers-must-retry) refusal also ends the walk.

SQS mode never returns: errors keep the message (see above), and a failed receive is retried every 5s.

## Trailing slashes in time-range `s3_prefix`

The time-range listing is `<s3_prefix>/<partition>/…` with a literal `/`, so `s3_prefix: otel/` lists `otel//year=…`, which no exporter key matches. Use `otel`. In SQS mode `s3_prefix` is a plain string prefix, where `otel/` is the stricter choice.

## Whole objects in memory

`GetObject` bodies are read fully with `io.ReadAll`, decompressed fully, then decoded. Peak memory per object is roughly compressed plus decompressed size plus the decoded pdata; object size is set on the writer side.

## Log sampling hides per-step lines

Each time-range step logs `Reading telemetry`, plus `No telemetry found for time` when empty, at `info`, so the Collector's default [log sampling](../../SKILL.md#verification-harness) drops most of them (observed).
