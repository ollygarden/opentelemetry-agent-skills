#!/usr/bin/env bash

set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo=${OTEL_SEMCONV_REPO:-}

if [[ -z "$repo" ]]; then
  printf 'OTEL_SEMCONV_REPO must point to a semantic-conventions checkout\n' >&2
  exit 2
fi

query=(env OTEL_SEMCONV_REPO="$repo" OTEL_SEMCONV_TAG=v1.44.0 "$script_dir/query-otel-semantic-conventions.sh")

messaging_spans=$("${query[@]}" messaging spans)
grep -q '^span[.]messaging[.]aws[.]sqs[.]send[.]producer' <<<"$messaging_spans"

messaging_common=$("${query[@]}" messaging common)
grep -q '^messaging[.]kafka[.]attributes[.]common' <<<"$messaging_common"
grep -q '^messaging[.]rabbitmq[.]attributes[.]destination' <<<"$messaging_common"

exact_entry=$("${query[@]}" messaging span.messaging.aws.sqs.send.producer)
grep -q '^source: .*semantic-conventions/blob/v1[.]44[.]0/model/messaging/aws[.]yaml#L' <<<"$exact_entry"

client_registry=$("${query[@]}" client registry)
grep -q '^client[.]address' <<<"$client_registry"

client_exact=$("${query[@]}" client client.address)
grep -q '^source: .*semantic-conventions/blob/v1[.]44[.]0/model/client/registry[.]yaml#L' <<<"$client_exact"

hardware_metrics=$("${query[@]}" hardware metrics)
grep -q '^hw[.]battery[.]charge ' <<<"$hardware_metrics"

hardware_exact=$("${query[@]}" hardware hw.battery.charge)
grep -q '^source: .*semantic-conventions/blob/v1[.]44[.]0/model/hardware/battery-metrics[.]yaml#L' <<<"$hardware_exact"

legacy_http=$(env OTEL_SEMCONV_REPO="$repo" OTEL_SEMCONV_TAG=v1.43.0 "$script_dir/query-otel-semantic-conventions.sh" http spans)
grep -q '^span[.]http[.]client' <<<"$legacy_http"

legacy_messaging=$(env OTEL_SEMCONV_REPO="$repo" OTEL_SEMCONV_TAG=v1.43.0 "$script_dir/query-otel-semantic-conventions.sh" messaging spans)
grep -q '^messaging[.]kafka' <<<"$legacy_messaging"

query_v145=(env OTEL_SEMCONV_REPO="$repo" OTEL_SEMCONV_TAG=v1.45.0 "$script_dir/query-otel-semantic-conventions.sh")

rpc_spans=$("${query_v145[@]}" rpc spans)
grep -q '^span[.]rpc[.]grpc[.]call[.]client' <<<"$rpc_spans"

rpc_metrics=$("${query_v145[@]}" rpc metrics)
grep -q '^metric[.]rpc[.]grpc[.]client[.]call[.]duration' <<<"$rpc_metrics"

redis_exact=$("${query_v145[@]}" db span.db.redis.client)
grep -q '^source: .*semantic-conventions/blob/v1[.]45[.]0/model/db/redis[.]yaml#L' <<<"$redis_exact"
