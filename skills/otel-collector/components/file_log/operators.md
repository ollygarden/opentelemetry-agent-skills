# `file_log` receiver: operators

The `operators` list is a **stanza** pipeline — an ordered chain of small, single-purpose
steps that run on each log entry as it leaves the file reader. This is where parsing,
timestamp/severity extraction, and entry combining happen. The full operator reference lives
upstream in [`pkg/stanza/docs/operators`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/stanza/docs/operators); this page is the catalog plus the wiring rules you need to author a pipeline.

## Pipeline rules

- Every operator has a **`type`**. Each can have a unique **`id`**; if you use the same `type`
  twice, you must give them distinct `id`s (otherwise `id` defaults to `type` and they collide).
- By default each operator passes its output to the **next** operator in the list; the last one
  emits from the receiver. Override with **`output: <id>`** to jump to a specific operator
  (used for branching after a `router`).
- For `file_log` the input is **implicit** — the receiver *is* the `file_input`. Do **not** add
  `file_input` or other input operators to `operators`; start with a parser or general-purpose
  operator. Only parsers and general-purpose operators should be used here.
- The raw line is in the entry **body**. Parsers read from `parse_from` (default `body`) and
  write to `parse_to` (default `attributes`).

## Parsers

| `type` | Parses |
|--------|--------|
| `container` | Docker / CRI-O / containerd container log formats — the standard first parser for Kubernetes pod logs (`/var/log/pods/...`). |
| `json_parser` | A JSON object into fields. |
| `json_array_parser` | A JSON array into a list of values. |
| `csv_parser` | CSV using a configured header. |
| `regex_parser` | Named capture groups of a regex into fields. |
| `key_value_parser` | `k=v` pairs into fields. |
| `uri_parser` | A URI string into its components. |
| `syslog_parser` | RFC 3164 / RFC 5424 syslog. |
| `severity_parser` | A field's value into `SeverityNumber`/`SeverityText`. |
| `time_parser` | A field's value into the record timestamp. |
| `trace_parser` | Trace/span IDs and flags into the record's trace context. |
| `scope_name_parser` | A field into the instrumentation scope name. |

### Embedded timestamp / severity / trace parsing

Most parsers can run a follow-up timestamp, severity, and/or trace parse inline instead of
needing a separate operator — the "complex parser" form. This is the idiomatic shape:

```yaml
operators:
  - type: regex_parser
    regex: '^(?P<ts>\S+) (?P<sev>[A-Z]+) (?P<msg>.*)$'
    timestamp:
      parse_from: attributes.ts
      layout: '%Y-%m-%dT%H:%M:%S%z'   # strptime by default; layout_type: gotime|epoch also supported
    severity:
      parse_from: attributes.sev       # maps INFO/WARN/ERROR/... to SeverityNumber
```

## General-purpose operators

| `type` | Does |
|--------|------|
| `add` | Add a field with a static value or expression. |
| `copy` | Copy a field to another location. |
| `move` | Move/rename a field. |
| `remove` | Delete a field. |
| `retain` | Keep only the listed fields, drop the rest. |
| `flatten` | Flatten a nested map one level. |
| `assign_keys` | Assign keys to a list, turning it into a map. |
| `filter` | Drop an entry that matches an [expression](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/stanza/docs/types/expression.md) (entry-level drop, distinct from the `filter` processor). |
| `router` | Branch entries to different downstream operators by expression (use with `output`). |
| `recombine` | Combine multiple consecutive entries into one (e.g. stack traces, multi-line app logs). |
| `regex_replace` | Regex find/replace on a field. |
| `unquote` | Unquote a quoted string field. |
| `sanitize_utf8` | Replace invalid UTF-8 sequences. |
| `noop` | Pass through unchanged (placeholder/testing). |

> `recombine` vs `multiline`: the top-level [`multiline`](advanced.md#multiline-entries) block
> splits the *byte stream* into entries before any operator runs (one regex, line start/end).
> The `recombine` operator merges *already-split* entries using an expression — more flexible
> (e.g. "combine until the next line starts with a timestamp"), at more cost. Prefer `multiline`
> for simple cases.

## Inputs (not used here)

`file_input`, `journald_input`, `syslog_input`, `tcp_input`, `udp_input`,
`windows_eventlog_input`, `stdin` exist in the stanza library but belong to their **own
receivers** (`journald`, `tcplog`, `udplog`, `windowseventlog`, …). Don't place them in
`file_log`'s `operators`.

## Worked example — JSON logs with timestamp and severity

```yaml
receivers:
  file_log:
    include: [ /var/log/myservice/*.json ]
    start_at: beginning
    operators:
      - type: json_parser
        timestamp:
          parse_from: attributes.time
          layout: '%Y-%m-%d %H:%M:%S'
        severity:
          parse_from: attributes.level
      - type: move                 # promote the parsed message into the body
        from: attributes.message
        to: body
```
