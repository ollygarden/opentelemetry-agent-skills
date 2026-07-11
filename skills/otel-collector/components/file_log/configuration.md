# `file_log` receiver: configuration

Every key below traces to the contrib v0.156.0 source (`receiver/filelogreceiver/README.md`, generated from the `fileconsumer` config). The receiver wraps the stanza `file_input` operator, so most keys are file-discovery and file-reading knobs; the parsing pipeline is `operators` (see [operators.md](operators.md)).

Only `include` is required.

## File discovery

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `include` | `[]string` | **required** | Glob patterns of file paths to read. |
| `exclude` | `[]string` | `[]` | Glob patterns to exclude, applied against the `include` matches. |
| `exclude_older_than` | duration | — | Exclude files whose modification time is older than this age. |
| `start_at` | `beginning` \| `end` | `end` | Where to start reading **newly discovered** files. Default `end` means a pre-existing, idle file yields nothing — see [quirks.md](quirks.md). |
| `poll_interval` | duration | `200ms` | Interval between filesystem polls for new data / new files. |
| `max_concurrent_files` | int | `1024` | Max files read concurrently; excess files are processed in batches. |
| `max_batches` | int | `0` | When batching is needed to respect `max_concurrent_files`, caps batches per poll. `0` = no limit. |

## File reading & sizing

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `encoding` | string | `utf-8` | File encoding. See [supported encodings](#supported-encodings). |
| `fingerprint_size` | bytes | `1000` | Bytes from the start of a file used to uniquely identify it. Minimum `16`. Decreasing it re-ingests files larger than the new size. |
| `initial_buffer_size` | bytes | `16KiB` | Initial read-buffer size for headers and logs; grows as needed. |
| `max_log_size` | bytes | `1MiB` | Max size of a single log entry. Behavior on overflow set by `max_log_size_behavior`. |
| `max_log_size_behavior` | `split` \| `truncate` | `split` | `split` emits the oversized entry as multiple records; `truncate` keeps the head and drops the rest. |
| `force_flush_period` | duration | `500ms` | Time since last new data after which a partial trailing line may be emitted. `0` disables (waits for a delimiter). |
| `preserve_leading_whitespaces` | bool | `false` | Keep leading whitespace on each entry. |
| `preserve_trailing_whitespaces` | bool | `false` | Keep trailing whitespace on each entry. |
| `compression` | `` \| `gzip` \| `auto` | `` | Read compressed files. `auto` auto-detects gzip by header. Appended (not rewritten) compressed files only. |

## Emitted file attributes

Each toggles a `log.file.*` attribute on every record from that file. `*_owner_*` and `*_permissions` are **not supported on Windows**.

| Key | Type | Default | Attribute added |
|-----|------|---------|-----------------|
| `include_file_name` | bool | `true` | `log.file.name` |
| `include_file_path` | bool | `false` | `log.file.path` |
| `include_file_name_resolved` | bool | `false` | `log.file.name_resolved` (after symlink resolution) |
| `include_file_path_resolved` | bool | `false` | `log.file.path_resolved` (after symlink resolution) |
| `include_file_owner_name` | bool | `false` | `log.file.owner.name` (Unix only) |
| `include_file_owner_group_name` | bool | `false` | `log.file.owner.group.name` (Unix only) |
| `include_file_permissions` | bool | `false` | `log.file.permissions`, 3-digit octal e.g. `755` (Unix only) |
| `include_file_record_number` | bool | `false` | `log.file.record_number` |
| `include_file_record_offset` | bool | `false` | `log.file.record_offset` |

## Static attributes & resource

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `attributes` | map | `{}` | `key: value` pairs added to each entry's attributes. Values are strings or stanza [expressions](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/stanza/docs/types/expression.md) returning a string. |
| `resource` | map | `{}` | `key: value` pairs added to each entry's resource. Same value rules as `attributes`. |
| `operators` | `[]operator` | `[]` | The stanza parsing pipeline. See [operators.md](operators.md). |
| `multiline` | block | — | Split entries on a pattern instead of newlines. See [advanced.md](advanced.md#multiline-entries). |

## Durability & rotation

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `storage` | component ID | none (in-memory) | A `file_storage` (or other storage) extension ID for persisting offsets across restarts. See [advanced.md](advanced.md#durable-offsets-with-storage). |
| `delete_after_read` | bool | `false` | Read then immediately delete each file. Requires feature gate `filelog.allowFileDeletion`; must be `false` when `start_at: end`. |
| `acquire_fs_lock` | bool | `false` | Acquire a filesystem lock before reading (Unix only). |
| `file_cache_advise` | bool | `false` | Hint the OS to release cached pages after read (Linux only); helps page-cache pressure on large sequential reads. |
| `on_truncate` | `ignore` \| `read_whole_file` \| `read_new` | `ignore` | Behavior when a same-fingerprint file shrinks (copytruncate rotation). See [advanced.md](advanced.md#log-rotation-and-on_truncate). |
| `polls_to_archive` | int | `0` | **Experimental.** With `storage` set, retain offsets of old readers on disk for this many poll cycles instead of purging after 3 generations. |

## Header metadata parsing

Requires feature gate `filelog.allowHeaderMetadataParsing` and `start_at: beginning`. See [advanced.md](advanced.md#header-metadata-parsing).

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `header.pattern` | regex | required (when `header` set) | Regex that every header line matches. |
| `header.metadata_operators` | `[]operator` | required (when `header` set) | Operators that parse attributes out of the header lines. Merged onto every record from the file; header lines themselves are not emitted. |

## Retry on downstream failure

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `retry_on_failure.enabled` | bool | `false` | Pause reading a file and resend the current batch on a downstream error. |
| `retry_on_failure.initial_interval` | duration | `1s` | Backoff before the first retry. |
| `retry_on_failure.max_interval` | duration | `30s` | Upper bound on backoff. |
| `retry_on_failure.max_elapsed_time` | duration | `5m` | Total time (incl. retries) before data is dropped. `0` = retry forever. |

## File ordering (`ordering_criteria`)

Tracks only the top-N files after grouping and sorting — useful when many rotated files match but you want a deterministic read order.

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `ordering_criteria.regex` | regex | — | Must contain named capture groups referenced by `sort_by.regex_key`. |
| `ordering_criteria.group_by` | regex | — | Named-capture regex for pre-sort grouping. |
| `ordering_criteria.top_n` | int | `1` | Number of files to track after ordering. |
| `ordering_criteria.sort_by.regex_key` | string | — | Named group (from `ordering_criteria.regex`) to sort on. |
| `ordering_criteria.sort_by.sort_type` | `numeric` \| `alphabetical` \| `timestamp` \| `mtime` | — | Sort strategy. `mtime` requires the `filelog.mtimeSortType` feature gate (Alpha, off by default). |
| `ordering_criteria.sort_by.location` | string | — | Timestamp location (when `sort_type: timestamp`). |
| `ordering_criteria.sort_by.layout` | strptime | — | Timestamp format, strptime (when `sort_type: timestamp`). ⚠️ The upstream README documents this key as `format`, but the actual field (matcher source + `config.schema.yaml`) is `layout` — `format:` is rejected as an invalid key (verified on v0.156.0). |
| `ordering_criteria.sort_by.ascending` | bool | — | Sort direction. |

## Supported encodings

`nop` (raw bytes, no validation), `utf-8` (default), `utf-8-raw` (no invalid-byte replacement), `utf-16le`, `utf-16be`, `ascii`, `big5`. Other IANA charsets are supported best-effort.

## Minimal config

```yaml
receivers:
  file_log:
    include: [ /var/log/myapp/*.log ]
    start_at: beginning            # default `end` would skip existing content
    operators:
      - type: json_parser          # parse each line as JSON into attributes
        timestamp:
          parse_from: attributes.time
          layout: '%Y-%m-%dT%H:%M:%S%z'
```
