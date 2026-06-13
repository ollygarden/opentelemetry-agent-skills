# `file_log` receiver

| | |
|-|-|
| Kind | receiver |
| Type | `file_log` (formerly `filelog`) |
| Signals | logs (Beta) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/filelogreceiver> |
| Rename | `filelog` → `file_log` in **v0.149.0**; `filelog` kept as a deprecated alias (runtime warns `"filelog" alias is deprecated; use "file_log" instead`). |

## Description

Tails log files from disk and turns each line (or multi-line entry) into an OTLP log record. You give it a list of glob patterns (`include`), and it discovers matching files, follows them as they grow, and tracks how far it has read into each one. File identity is established by a content **fingerprint** (the first `fingerprint_size` bytes) plus inode, so the receiver keeps reading correctly across both common rotation strategies — move/create and copy/truncate — even when the rotated file no longer matches `include`. Read offsets live in memory by default; point `storage` at a `file_storage` extension to make them survive a collector restart.

The raw line lands in the log record `Body` and a few file attributes (`log.file.name` by default) are attached. Everything else — extracting fields, parsing timestamps and severity, combining stack traces into one record — is done by an ordered pipeline of **stanza operators** under `operators:` (`json_parser`, `regex_parser`, `severity_parser`, `recombine`, …). The operator framework, not the receiver's top-level keys, is where most real-world configuration lives.

The single most common surprise: **`start_at` defaults to `end`**, so a file that already exists and is not being actively appended to yields **no records**. Set `start_at: beginning` to read existing content. See [quirks.md](quirks.md).

## Main use-cases

Use when:
- You collect logs that an application writes to files (container stdout/stderr on a node, app log files, system logs) rather than sending over OTLP.
- You need to parse unstructured or semi-structured text into structured records — JSON, regex-delimited, syslog, key/value — with timestamp and severity extraction.
- You need durable offset tracking so a collector restart resumes where it left off (pair with a `file_storage` extension via `storage`).
- You tail Kubernetes pod logs (`/var/log/pods/...`) — the canonical node-level log-collection path, typically followed by `k8s_attributes` for metadata enrichment.

Avoid when:
- The source already speaks OTLP — use the [`otlp`](../otlp/README.md) receiver.
- You only need journald, Windows Event Log, TCP/UDP syslog, or stdin — use the dedicated receiver (`journald`, `windowseventlog`, `tcplog`/`udplog`, …) rather than wiring those stanza inputs by hand.

## Related components

- [`otlp` receiver](../otlp/README.md) — the OTLP-native ingress; `file_log` is its file-tailing counterpart for logs.
- [`k8s_attributes`](../k8s_attributes/README.md) — enrich tailed pod logs with Kubernetes metadata (namespace, pod, labels).
- [`transform`](../transform/README.md) — OTTL-based post-processing once records are in the pipeline; the stanza `operators` do parsing at ingest, `transform` does mutation downstream.
- [`memory_limiter`](../memory_limiter/README.md) — belongs **first** in the logs pipeline fed by this receiver.
- `file_storage` extension — set `storage:` to it so read offsets survive collector restarts (not yet a page in this skill).

## Details

- [Configuration](configuration.md) — every top-level config key (include/exclude, `start_at`, fingerprinting, sizing, rotation, `storage`, `header`, `ordering_criteria`) with defaults and validation.
- [Operators](operators.md) — the stanza operator pipeline: the available inputs/parsers/general-purpose operators, the `id`/`output` wiring rules, and embedded timestamp/severity parsing.
- [Verification](verification.md) — a file-based recipe (write lines to a mounted file → `file_log` → `debug`), verified on contrib v0.154.0. Notes why `telemetrygen` cannot drive this receiver.
- [Advanced use-cases](advanced.md) — multiline entries, encodings, durable offsets via `storage`, header metadata parsing, log-rotation handling (`on_truncate`), file ordering, and `compression`.
- [Known quirks](quirks.md) — the `start_at: end` default, the rename/alias, feature-gated options, `delete_after_read` constraints, fingerprint re-ingestion, and stability.
