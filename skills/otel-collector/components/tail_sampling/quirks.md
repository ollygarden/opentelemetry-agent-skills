# `tail_sampling`: known quirks

## All spans of a trace must reach the SAME instance

The decision is made per-trace inside one process. If spans of one trace are spread across multiple tail-sampling collectors, each sees only a fragment and makes its own (wrong/partial) decision. Whenever you run more than one tail-sampling collector you **must** put a `loadbalancing` exporter layer in front that routes by `traceID`. A single instance needs no load balancer.

## Memory scales with `num_traces` and trace size

`num_traces` is the in-flight trace buffer: every trace awaiting a decision is held in memory. Rough estimate `num_traces * avg_spans_per_trace * ~1KB/span` (e.g. 50,000 traces × 20 spans ≈ 1 GB). Longer `decision_wait` means more traces resident at once. When the buffer fills, the oldest traces are evicted **before** their decision (surfacing as the `sampling_trace_dropped_too_early` metric) unless `block_on_overflow` is set. Size it as `traces_per_sec * decision_wait_seconds * safety_factor`.

## `decision_wait` adds latency and a fixed window

Sampled traces are only exported after `decision_wait` expires, so this delay is added before any trace leaves the processor. Set it long enough for distributed traces to assemble; too short and late spans miss the window. Spans that arrive **after** the decision are not retroactively folded into it — they either inherit the cached decision (if `decision_cache` is configured) or risk forming a separate/dropped partial trace.

## Stateful, single-process decision

Tail sampling is inherently stateful: the keep/drop choice is computed once, in one process, from whatever spans were buffered at decision time. There is no cross-instance coordination and no re-evaluation of a trace once decided. Place context-enriching processors (e.g. `k8sattributes`) **before** `tail_sampling` in the pipeline, since the processor re-batches spans and downstream context can be lost.
