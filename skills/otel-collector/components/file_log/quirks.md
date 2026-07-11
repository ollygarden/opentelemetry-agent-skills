# `file_log` receiver: known quirks

## `start_at` defaults to `end` — existing files look empty

The single most common "the receiver reads nothing" problem. `start_at` defaults to `end`, so for
a **newly discovered** file the receiver begins at the *current* end and only emits lines appended
**after** it starts watching. A file that already exists and is not being actively written to
produces **zero records**.

```yaml
receivers:
  file_log:
    include: [ /var/log/myapp/*.log ]
    start_at: beginning            # read existing content; default `end` skips it
```

Use `beginning` for batch/static files and first-run backfills; `end` is the right default for
high-volume live logs where you don't want to re-read history on every restart (pair with
`storage` so restarts resume mid-file).

## The type was renamed `filelog` → `file_log` in v0.149.0

New configs should use `file_log`. The old `filelog` name still works as a deprecated alias but
logs a startup warning (verified on 0.154.0):

```
warn  "filelog" alias is deprecated; use "file_log" instead  {"otelcol.component.id": "filelog", ...}
```

When reviewing an existing config, `filelog:` is **not** broken — it's the legacy spelling.

## Feature-gated options

Some keys do nothing (or error) unless a feature gate is enabled at startup
(`--feature-gates=<gate>`):

| Option | Required gate | State (v0.156.0) |
|--------|---------------|-------|
| `delete_after_read` | `filelog.allowFileDeletion` | Alpha (off by default) — pass `--feature-gates=filelog.allowFileDeletion` |
| `header` parsing | `filelog.allowHeaderMetadataParsing` | Beta (on by default) — no flag needed |
| `ordering_criteria.sort_by.sort_type: mtime` | `filelog.mtimeSortType` | Alpha (off by default) |
| include/exclude case-insensitive globbing (Windows) | `filelog.windows.caseInsensitive` | Alpha (off by default) |
| protobuf checkpoint encoding | `filelog.protobufCheckpointEncoding` | Beta (**on by default** since v0.156.0; Alpha off in v0.148.0–v0.155.0) — ~7× faster decode, ~31% smaller; reads both formats either way |

## `delete_after_read` conflicts with `start_at: end`

`delete_after_read: true` reads a file then deletes it; it must be paired with
`start_at: beginning` (it is invalid with `start_at: end`) and needs the
`filelog.allowFileDeletion` gate.

## Fingerprinting and re-ingestion

Files are identified by the first `fingerprint_size` bytes (default `1000`, minimum `16`) plus
inode. Two consequences:

- **Decreasing `fingerprint_size`** changes identity for files larger than the new size →
  they are treated as new and **re-ingested**.
- Files whose first `fingerprint_size` bytes are **identical** (e.g. a fixed boilerplate header)
  can be mistaken for the same file. Raise `fingerprint_size` past the common prefix, or ensure
  early bytes differ.

## Don't put input operators in `operators`

The receiver *is* the file input. Adding `file_input` (or `tcp_input`, `journald_input`, …) to the
`operators` list is wrong — only parsers and general-purpose operators belong there. Use the
dedicated receiver for other inputs.

## `multiline` needs exactly one pattern

The `multiline` block must specify **exactly one** of `line_start_pattern` /
`line_end_pattern` — setting both, or neither (while present), is a config error.

## Owner/permission attributes are Unix-only

`include_file_owner_name`, `include_file_owner_group_name`, and `include_file_permissions` are not
supported on Windows.

## Stability

`logs` is **Beta** (production-viable; breaking config changes rare). Distributed in `contrib` and
`k8s` — **not** in the `core` or `otlp` distributions, so a minimal/core collector won't have it.
