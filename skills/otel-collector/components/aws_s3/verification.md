# `aws_s3` receiver: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`telemetrygen` cannot feed this receiver directly — it reads objects, not OTLP requests. Both recipes read the objects written by the [`awss3` exporter verification](../aws_s3_exporter/verification.md): run that first, in the same bash shell, and keep its `s3verify` network, `s3-store`, `s3-keys.txt`, `IMG`, and `waitlog`. Credentials are the same dummy values, passed through the environment.

## Time-range replay

Config (`s3-replay.yaml`):

```yaml
receivers:
  aws_s3:
    starttime: "${env:START}"
    endtime: "${env:END}"
    s3downloader:
      s3_bucket: telemetry
      s3_prefix: otel                  # no trailing slash
      s3_partition_timezone: UTC       # same as the exporter
      endpoint: http://s3-store:8333
      s3_force_path_style: true
exporters:
  debug:
    verbosity: basic
service:
  pipelines:
    traces:
      receivers: [aws_s3]
      exporters: [debug]
    logs:
      receivers: [aws_s3]
      exporters: [debug]
```

Replay the last 30 minutes (GNU `date`; on BSD use `-v-30M` / `-v+1M`) and wait until both signals end their walk:

```bash
START=$(date -u -d '-30 min' '+%Y-%m-%d %H:%M')
END=$(date -u -d '+1 min' '+%Y-%m-%d %H:%M')
docker run -d --name s3-replay --network s3verify -e START="$START" -e END="$END" \
  -e AWS_ACCESS_KEY_ID=verify-key -e AWS_SECRET_ACCESS_KEY=verify-secret \
  -v "$PWD/s3-replay.yaml:/etc/otelcol-contrib/config.yaml" $IMG
for _ in $(seq 60); do
  [ "$(docker logs s3-replay 2>&1 | grep -c -E '(Finished|Error) reading telemetry')" -ge 2 ] && break; sleep 1
done
docker logs s3-replay 2>&1 | grep -o -E '"(spans|log records)": [0-9]+' |
  awk -F': ' '{n[$1]+=$2} END {for (k in n) print k, n[k]}' | sort
docker logs s3-replay 2>&1 | grep -c 'Error reading telemetry'
```

**What proves it worked:** every exported span and log record comes back, and no walk failed:

```text
"log records" 7
"spans" 10
0
```

Afterwards the receiver idles.

## SQS-driven ingestion

This uses [ElasticMQ](https://github.com/softwaremill/elasticmq), an SQS-compatible server, with a hand-sent notification in the shape S3 publishes ([event message structure](https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-content-structure.html)); the local store does not publish S3 events itself, so bucket-to-queue wiring is not exercised.

Config (`s3-sqs.yaml`) — one signal only, see [quirks](quirks.md#sqs-mode-ignores-the-signal-type):

```yaml
receivers:
  aws_s3:
    s3downloader:
      s3_bucket: telemetry
      s3_prefix: otel/
      endpoint: http://s3-store:8333
      s3_force_path_style: true
    sqs:
      queue_url: http://sqs:9324/000000000000/s3-events
      region: us-east-1
      endpoint: http://sqs:9324
exporters:
  debug:
    verbosity: basic
service:
  pipelines:
    logs:
      receivers: [aws_s3]
      exporters: [debug]
```

Create a queue with a 5s visibility timeout, start the receiver, and notify about the logs object (S3 URL-encodes keys, so `=` becomes `%3D`):

```bash
docker run -d --name sqs --network s3verify -p 127.0.0.1:9324:9324 softwaremill/elasticmq-native:1.7.1
for _ in $(seq 30); do curl -sf 'http://127.0.0.1:9324/?Action=CreateQueue&QueueName=s3-events&Attribute.1.Name=VisibilityTimeout&Attribute.1.Value=5' >/dev/null && break; sleep 1; done
docker run -d --name s3-sqs --network s3verify \
  -e AWS_ACCESS_KEY_ID=verify-key -e AWS_SECRET_ACCESS_KEY=verify-secret \
  -v "$PWD/s3-sqs.yaml:/etc/otelcol-contrib/config.yaml" $IMG
Q=http://127.0.0.1:9324/000000000000/s3-events
send() { curl -s -o /dev/null "$Q" --data-urlencode Action=SendMessage --data-urlencode "MessageBody=$1"; }
depth() { curl -s "$Q?Action=GetQueueAttributes&AttributeName.1=All" | tr -d ' \n' |
  grep -o -E 'ApproximateNumberOfMessages(NotVisible)?</Name><Value>[0-9]+'; }
KEY=$(grep -m1 logs_ s3-keys.txt)
send '{"Records":[{"eventSource":"aws:s3","eventName":"ObjectCreated:Put","s3":{"bucket":{"name":"telemetry"},"object":{"key":"'"${KEY//=/%3D}"'"}}}]}'
waitlog s3-sqs '"log records": 7' && sleep 1 && depth
```

**What proves it worked:** the object is decoded and the message deleted:

```text
ApproximateNumberOfMessagesNotVisible</Name><Value>0
ApproximateNumberOfMessages</Name><Value>0
```

Then send a message that is not an S3 event; it is never deleted and comes back after each visibility timeout:

```bash
send 'not an S3 event'
for _ in $(seq 30); do
  [ "$(docker logs s3-sqs 2>&1 | grep -c 'Failed to parse message as SNS notification')" -ge 2 ] && break; sleep 1
done; depth
```

Expect `ApproximateNumberOfMessagesNotVisible` at `1`: the message was received twice and is in flight again.

Tear down everything from both pages:

```bash
docker rm -f s3-sqs sqs s3-replay s3-export s3-store
docker network rm s3verify
```
