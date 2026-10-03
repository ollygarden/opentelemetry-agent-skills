# `tail_sampling`: known quirks

## All spans of a trace must reach the SAME instance

The decision is made per-trace inside one process. If spans of one trace are spread across multiple tail-sampling collectors, each sees only a fragment and makes its own (wrong/partial) decision. Whenever you run more than one tail-sampling collector you **must** put a `load_balancing` exporter layer in front that routes by `traceID`. A single instance needs no load balancer.

`num_shards` does not change this rule. Sharding is internal to one processor and routes a complete
trace to one event loop; it is not coordination among Collector instances.

## Memory scales with `num_traces` and trace size

`num_traces` is the in-flight trace buffer: every trace awaiting a decision is held in memory. Rough estimate `num_traces * avg_spans_per_trace * ~1KB/span` (e.g. 50,000 traces × 20 spans ≈ 1 GB). Longer `decision_wait` means more traces resident at once. When the buffer fills, the oldest traces are evicted **before** their decision (surfacing as the `sampling_trace_dropped_too_early` metric) unless `block_on_overflow` is set. Size it as `traces_per_sec * decision_wait_seconds * safety_factor`.

## `decision_wait` adds latency and a fixed window

Sampled traces are only exported after `decision_wait` expires, so this delay is added before any trace leaves the processor. Set it long enough for distributed traces to assemble; too short and late spans miss the window. Spans that arrive **after** the decision are not retroactively folded into it — they either inherit the cached decision (if `decision_cache` is configured) or risk forming a separate/dropped partial trace.

## Stateful, single-process decision

Tail sampling is inherently stateful: the keep/drop choice is computed once, in one process, from whatever spans were buffered at decision time. There is no cross-instance coordination and no re-evaluation of a trace once decided. Place context-enriching processors (e.g. `k8s_attributes`) **before** `tail_sampling` in the pipeline, since the processor re-batches spans and downstream context can be lost.

## Tracestate rewriting is opt-in and parse-sensitive

The processor only consumes and rewrites OpenTelemetry probability sampling fields when the alpha `processor.tailsamplingprocessor.usetracestate` gate is enabled. If a span's W3C `tracestate` cannot be parsed, its outgoing `th` is not rewritten; monitor `otelcol_processor_tail_sampling_count_spans_with_unparseable_tracestate` (Development stability) for this case. The probabilistic decision still falls back to the legacy trace-ID hash when the trace carries no probability sampling information. Do not combine the gate with `sample_on_first_match`, which can stop before the least-strict matched threshold is known. Before v0.162.0, late-arriving spans of an already-sampled trace could be rewritten with a different threshold than the original decision; v0.162.0 reuses the original decision's threshold.
