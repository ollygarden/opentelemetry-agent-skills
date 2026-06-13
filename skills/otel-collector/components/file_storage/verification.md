# `file_storage`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`file_storage` is an **extension**, so it does not sit in a pipeline and there is no "telemetrygen → component → debug" path to exercise it directly. The behavior worth proving is **state durability across a Collector restart**. This recipe drives the extension through a [`file_log`](../file_log/README.md) receiver and shows that the receiver's read offset survives a restart: after a restart, already-read lines are **not** re-read even with `start_at: beginning`. No `telemetrygen` is involved — the input is a file on disk (like the `file_log` page). Verified end-to-end on `otel/opentelemetry-collector-contrib:0.154.0`.

## Setup

In a scratch dir (`/tmp/fsverify`), create a `storage/` dir and a `logs/` dir. The contrib image runs as a non-root uid, so a mounted host dir must be writable by it — make the scratch tree world-writable for the repro:

```bash
mkdir -p /tmp/fsverify/storage /tmp/fsverify/logs
printf 'line one\nline two\nline three\n' > /tmp/fsverify/logs/app.log
chmod -R 777 /tmp/fsverify
cd /tmp/fsverify
```

Config `fs.yaml` (minimal repro — omits `memory_limiter` etc. on purpose, to isolate the extension):

```yaml
extensions:
  file_storage:
    directory: /storage          # mounted host dir; exists, so create_directory not needed
receivers:
  file_log:
    include: [/logs/app.log]
    start_at: beginning
    storage: file_storage        # <- receiver persists its read offset here
exporters:
  debug:
    verbosity: detailed          # basic verbosity prints no per-record lines
service:
  extensions: [file_storage]     # extensions must be listed here to load
  pipelines:
    logs:
      receivers: [file_log]
      exporters: [debug]
```

## Run 1 — first read

Start the Collector, mounting **both** the storage and logs dirs:

```bash
IMG=otel/opentelemetry-collector-contrib:0.154.0
docker run -d --name fsrun1 \
  -v "$PWD/fs.yaml:/etc/otelcol-contrib/config.yaml" \
  -v "$PWD/storage:/storage" -v "$PWD/logs:/logs" $IMG
sleep 5
docker logs fsrun1 2>&1 | grep -c '^LogRecord #'        # => 3
docker logs fsrun1 2>&1 | grep 'Body: Str(line'         # => line one/two/three
ls /tmp/fsverify/storage                                # => receiver_filelog_  (the bbolt file)
docker rm -f fsrun1
```

**Verified result:** 3 records (`line one`, `line two`, `line three`), and a bbolt file **`receiver_filelog_`** (32768 bytes) appears in the storage dir. Note the on-disk name is `receiver_filelog_` (the deprecated `filelog` type spelling, trailing `_` for the empty instance name) even though the config uses `file_log` — see [quirks.md](quirks.md).

## Run 2 — restart resumes from the persisted offset

Append 2 lines and restart with the **same** storage dir:

```bash
printf 'line four\nline five\n' >> /tmp/fsverify/logs/app.log   # now 5 lines
docker run -d --name fsrun2 \
  -v "$PWD/fs.yaml:/etc/otelcol-contrib/config.yaml" \
  -v "$PWD/storage:/storage" -v "$PWD/logs:/logs" $IMG
sleep 5
docker logs fsrun2 2>&1 | grep -c '^LogRecord #'        # => 2
docker logs fsrun2 2>&1 | grep 'Body: Str(line'         # => line four/line five ONLY
docker rm -f fsrun2
```

**Verified result:** only **2** records (`line four`, `line five`). The persisted offset means the first 3 lines are **not** re-read, even though `start_at: beginning`.

## Control — without `storage`, everything is re-read

Same restart, but with the `storage:` line removed from the receiver (`fs_nostore.yaml` = `fs.yaml` minus that line):

```bash
docker run -d --name fsctl \
  -v "$PWD/fs_nostore.yaml:/etc/otelcol-contrib/config.yaml" \
  -v "$PWD/storage:/storage" -v "$PWD/logs:/logs" $IMG
sleep 5
docker logs fsctl 2>&1 | grep -c '^LogRecord #'         # => 5  (re-reads everything)
docker rm -f fsctl
```

**Verified result:** **5** records — without `storage`, the restart re-reads all 5 lines (no offset memory). This contrast proves the `file_storage` extension is what preserved the offset.

## Tear down

```bash
docker rm -f fsrun1 fsrun2 fsctl 2>/dev/null
rm -rf /tmp/fsverify
```
