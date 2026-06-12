# `probabilistic_sampler`: known quirks

## `fail_closed` drops zero/invalid-randomness items

With `fail_closed: true` (the default), any item whose randomness source is missing or zero — for example a span with no usable TraceID, or a log without the configured `from_attribute` — is **dropped**, because the sampler cannot make a valid decision. If you see fewer items than the configured percentage would predict, check for items lacking randomness before assuming the rate is wrong. Setting `fail_closed: false` lets these items through at the cost of a less precise effective rate.

## `hash_seed` must match across collectors

In `hash_seed` mode the decision is `FNV(source, hash_seed) vs threshold`. Two collectors with **different** seeds make different keep/drop choices for the same TraceID, so chaining them (or load-balancing across them) yields an unpredictable combined rate. Every collector in one sampling tier must share the same `hash_seed`. The 56-bit `proportional`/`equalizing` modes avoid this by using W3C randomness instead of a seed.

## Precision is capped per mode

`sampling_precision` (1–14 hex digits) controls how finely the threshold is encoded, but each mode has a practical ceiling: roughly 4 for `hash_seed` (it is only 14-bit), about 5 for `equalizing`, and higher for `proportional` though still bounded by float32 rounding of `sampling_percentage`. Asking for more precision than a mode supports does not improve accuracy.

## Threshold persists downstream via tracestate

For traces, the sampling threshold is written into the W3C `tracestate` (`ot=th:...`); for logs it is written into `sampling.threshold` / `sampling.randomness` log attributes. This persistence is intentional — it lets a downstream sampler stay consistent — but it means the threshold travels with the data and a later component can read or act on it. Stripping `tracestate` between tiers breaks `proportional`/`equalizing` consistency.

## Child spans/logs are sampled with probability ≥ parent

To preserve trace completeness, a child item is kept with probability at least that of its parent — the sampler will not drop a child of a kept parent purely by chance. This reduces (but does not fully eliminate) orphaned-child situations; combined with consistent per-TraceID hashing, whole traces tend to be kept or dropped together.

## Counts are statistical, not exact

The kept fraction only **approaches** `sampling_percentage`% as the number of items grows. For small N the observed ratio is noisy and varies run to run; never assert an exact count from a single run.

## Stability caveats

Traces are Beta and logs are Alpha — log-side config keys (`attribute_source`, `from_attribute`, `sampling_priority`) and behavior can still change between releases. Confirm against the upstream README for the exact collector version before relying on log sampling in production.
