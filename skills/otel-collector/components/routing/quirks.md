# `routing`: known quirks

## Unmatched telemetry is dropped without `default_pipelines`

If a piece of telemetry matches no route and `default_pipelines` is not set, it is **silently dropped** — no error, no log. This is the most common cause of "where did my data go?" with this connector. Always configure `default_pipelines` (even just a catch-all archive or a `debug` pipeline) unless dropping unmatched data is genuinely what you want.

## `error_mode: propagate` drops the payload on OTTL errors

With the default `error_mode: propagate`, any OTTL evaluation error — a missing attribute, a type mismatch — makes the connector return an error and the **whole payload is dropped** from the collector. In production use `error_mode: ignore`, which logs the error and sends the payload to `default_pipelines` instead — so `ignore` only actually rescues data when `default_pipelines` is set; without it the errored payload still has nowhere to go. Guard fragile conditions with nil checks (`resource.attributes["k"] != nil and resource.attributes["k"] == "v"`) so a missing key doesn't error in the first place. The default flips to `ignore` under the `connector.routing.defaultErrorModeIgnore` feature gate (alpha, since v0.155.0), so don't assume `propagate` — set `error_mode` explicitly.

## `action` defaults to `move`, so matched data skips `default_pipelines`

The default `action` is `move`: a matched item is removed from further route evaluation and does **not** also reach `default_pipelines` or later routes. Use `action: copy` when you want matched data to stay available for subsequent routes and the default (e.g. an archive-everything route placed first).

## `statement` and `condition` are mutually exclusive

Each route needs **exactly one** of `statement` or `condition`. Setting both, or neither, fails config validation. `statement` is the `route() where <expr>` form — or `<editor> where <expr>` (e.g. `delete_key(...) where …`) when you also want to mutate matched data in the same pass (you can't chain an editor onto `route()` with `and`); `condition` is a bare boolean expression. Pick one per route.

## `request` context is deprecated and has a limited grammar

The `request` context is **deprecated as of v0.156.0** (a warning is logged when it is used) — prefer the `otelcol.client.metadata["key"][0]` (HTTP) / `otelcol.grpc.metadata["key"][0]` (gRPC) paths, which are valid in every signal context and support the full OTTL grammar. The legacy `request` context supports only a single `==` or `!=` comparison — no `and`/`or`, and a `statement` is rejected (it must be `condition`). gRPC metadata keys are lowercased, so use lowercase keys (`request["x-tenant"]` / `otelcol.grpc.metadata["x-tenant"][0]`) for gRPC traffic. The receiver must actually propagate request metadata (the `otlp` receiver does; file-based receivers don't).

## Item-level contexts can split a Resource bundle across pipelines

`span`, `metric`, `datapoint`, and `log` contexts evaluate per item, so a single incoming ResourceSpans/ResourceMetrics/ResourceLogs bundle can be torn apart — some items to one pipeline, others to another (or to the default). This is usually intended, but it means downstream pipelines may receive partial resource bundles. Use `resource` context when you want whole bundles kept together.

## Validation errors → fixes

| Error | Cause | Fix |
|-------|-------|-----|
| `the routing table is empty` | `table` missing or empty | Add at least one route. |
| `no condition or statement provided` | route has neither | Add one of `statement`/`condition`. |
| `both condition and statement provided` | route has both | Remove one. |
| `no pipelines defined` | route missing `pipelines` | Add at least one pipeline ID. |
| `invalid Action string: <value>` (inside a `cannot unmarshal the configuration` error) | `action` set to something other than `move`/`copy` | Use `move` or `copy` (or omit for the `move` default). |
| `"request" context requires a 'condition'` | `request` route used a `statement` | Use `condition` for `request` context (or migrate to `otelcol.*` paths). |
| `invalid context: <name>` | unsupported/typo context | Use `resource`, `span`, `metric`, `datapoint`, `log`, `otelcol`, or the deprecated `request` (and one valid for the signal). |

## `match_once` was removed in v0.120.0

`match_once` (deprecated v0.116.0, removed v0.120.0) is gone — each item now matches **at most one route**. Configs using `match_once: false` no longer load. To fan out, list multiple pipelines in one route; to route on independent dimensions, use parallel named instances (see [advanced](advanced.md#migrating-from-match_once)).

## Stability caveats

All three signal pairs (traces, metrics, logs) are **Alpha**. Config surface and behavior can shift between releases — confirm against the upstream README for your exact collector version. The connector replaces the `routingprocessor`, which has been removed from contrib (v0.156.0 rejects it as an unknown type); existing configs must migrate to the connector form.
