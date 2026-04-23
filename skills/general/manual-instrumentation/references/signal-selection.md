# Signal Selection

Choose the signal before adding code.

## Use A Span When

Use a span for:
- request or workflow flow through a boundary
- outbound dependency calls
- operations where latency and parent-child relationships matter
- business operations that benefit from trace correlation

## Use A Metric When

Use a metric for:
- throughput, rates, and counts
- latency distributions
- utilization or saturation
- queue depth, active jobs, retries, or resource pressure

Prefer metrics when the question is aggregate and recurring rather than transaction-specific.

## Use A Log Or Event When

Use a log or span event for:
- a discrete diagnostic fact
- retry and fallback annotations
- state transitions
- contextual evidence that should sit on a trace

Do not convert every successful step into a log or event.

## Do Nothing When

Skip telemetry when:
- it adds cost without helping runtime understanding
- it duplicates better existing telemetry
- it produces unbounded or noisy dimensions
- it only mirrors obvious local control flow
