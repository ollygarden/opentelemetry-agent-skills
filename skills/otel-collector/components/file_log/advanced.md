# `file_log` receiver: advanced use-cases

## Multiline entries

By default an entry is one line. The `multiline` block makes the **file reader** split the byte
stream on a pattern instead — for stack traces, pretty-printed payloads, or any record that spans
lines. It must contain **exactly one** of `line_start_pattern` or `line_end_pattern` (regexes);
`omit_pattern: true` strips the matched delimiter from each entry.

```yaml
receivers:
  file_log:
    include: [ /var/log/example/multiline.log ]
    multiline:
      line_start_pattern: ^Exception     # a new entry begins at each "Exception" line
```

Everything between two `^Exception` lines becomes a single record. For rules that can't be
expressed as one start/end regex (e.g. "combine until the next line has a timestamp"), use the
[`recombine`](operators.md#general-purpose-operators) operator instead — more flexible, more cost.

## Durable offsets with `storage`

By default read offsets live only in memory, so a collector restart re-reads from `start_at`
(typically losing position or re-ingesting). Point `storage` at a `file_storage` extension to
persist them:

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/file_storage

receivers:
  file_log:
    include: [ /var/log/myapp/*.log ]
    storage: file_storage          # persists Fingerprint + Offset + FileAttributes per file

service:
  extensions: [file_storage]
  pipelines:
    logs:
      receivers: [file_log]
      exporters: [debug]
```

Stored per tracked file: the fingerprint (first bytes), the byte offset, and the file attributes.
Note this guarantees the *file* is read accurately; records can still be lost further downstream —
for full durability also configure exporter persistent queues. With `polls_to_archive` (>0,
experimental) plus `storage`, offsets of readers older than three poll cycles are archived to disk
rather than purged, which helps avoid re-ingestion when used with `exclude_older_than`.

## Log rotation and `on_truncate`

The receiver follows files across rotation by inode + fingerprint, so it keeps reading even when a
rotated file no longer matches `include`. Two strategies are handled:

- **move/create** — handled transparently; reading continues on the renamed file.
- **copy/truncate** — the file shrinks in place. `on_truncate` controls the response when a
  same-fingerprint file is seen smaller than the stored offset:
  - `ignore` (default) — keep the offset; read nothing until the file grows past it. Avoids
    duplicates.
  - `read_whole_file` — reset offset to 0 and re-read everything. No data loss, possible
    duplicates.
  - `read_new` — set offset to the current (post-truncate) size; read only new appends.

```yaml
receivers:
  file_log:
    include: [ /var/log/myapp/*.log ]
    on_truncate: read_whole_file
```

**File-attribute behavior during rotation:** if the rotated name no longer matches `include`, the
*original* `log.file.name`/`log.file.path` are preserved; if it still matches, the *new* rotated
name is reported (prevents duplicate per-file metrics).

## Header metadata parsing

Read attributes from a file's header lines and stamp them onto every record from that file.
Requires the `filelog.allowHeaderMetadataParsing` feature gate and `start_at: beginning`.

```yaml
receivers:
  file_log:
    include: [ /var/log/myapp/*.log ]
    start_at: beginning
    header:
      pattern: '^#.*'                       # every header line matches this
      metadata_operators:
        - type: regex_parser
          regex: '^#Service: (?P<service>.*)$'
```

Each header line is run through `metadata_operators`; the resulting attributes are merged (upsert)
and attached to every emitted record. Header lines themselves are not emitted.

## Compression

Read gzip-compressed files directly:

```yaml
receivers:
  file_log:
    include: [ /var/log/example/*.log.gz ]
    compression: gzip              # or `auto` to auto-detect gzip by header
```

`auto` is useful for a mix of compressed and plain files in one receiver. Compressed inputs must
be **appended** to (not rewritten) if they keep growing.

## File ordering

When many rotated files match `include` but you only want the newest tracked, `ordering_criteria`
groups and sorts matches and keeps the top N:

```yaml
receivers:
  file_log:
    include: [ /var/log/myapp/app-*.log ]
    ordering_criteria:
      regex: 'app-(?P<num>\d+)\.log'
      top_n: 1
      sort_by:
        - sort_type: numeric
          regex_key: num
          ascending: false         # track only the highest-numbered file
```

## Kubernetes pod logs

The canonical node-level collection path: tail the kubelet's per-pod log files and parse the
container runtime's wrapper format with the [`container`](operators.md#parsers) operator, which
handles `docker`, `cri-o`, and `containerd` and lifts the runtime metadata (stream, timestamp)
out of the line. Pair with [`k8s_attributes`](../k8s_attributes/README.md) downstream for
namespace/pod/label enrichment.

```yaml
receivers:
  file_log:
    include: [ /var/log/pods/*/*/*.log ]
    start_at: end                  # live tail; don't backfill node history on restart
    include_file_path: true        # k8s_attributes can associate by log.file.path
    operators:
      - type: container            # auto-detects docker/cri-o/containerd
```

## Named instances

Run the receiver more than once (e.g. different parsing per log source) with the `type/name` form:

```yaml
receivers:
  file_log/app:
    include: [ /var/log/app/*.log ]
    operators: [ { type: json_parser } ]
  file_log/access:
    include: [ /var/log/nginx/access.log ]
    operators: [ { type: regex_parser, regex: '...' } ]

service:
  pipelines:
    logs:
      receivers: [file_log/app, file_log/access]
      exporters: [debug]
```
