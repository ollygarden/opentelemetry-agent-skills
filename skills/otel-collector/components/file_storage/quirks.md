# `file_storage`: known quirks

## The directory must already exist

By default `file_storage` does **not** create its data directory — if `directory` is missing, config validation fails with `directory must exist: … You can enable the create_directory option to automatically create it`. The fix is exactly what the error says: set `create_directory: true` (and, when compaction is enabled, the compaction directory must exist too, or be auto-created the same way).

## Validation errors → fix

All from `config.go` `Validate()`:

| Condition | Error | Fix |
|-----------|-------|-----|
| `directory` (or `compaction.directory` when `on_start`/`on_rebound` is set) does not exist, and `create_directory: false` | `directory must exist: … You can enable the create_directory option to automatically create it` | Create the dir, or set `create_directory: true`. |
| Path exists but is not a directory | `<path> is not a directory` | Point `directory` at an actual directory. |
| `compaction.max_transaction_size` < 0 | `max transaction size for compaction cannot be less than 0` | Use `0` (ignore sizes) or a positive value. |
| `compaction.on_rebound: true` with `check_interval <= 0` | `compaction check interval must be positive when rebound compaction is set` | Set `compaction.check_interval` to a positive duration (default `5s`). |
| `create_directory: true` with non-octal `directory_permissions` | `directory_permissions value must be a valid octal representation` | Use a valid octal string, e.g. `"0750"`. |
| `directory_permissions` with bits outside `0777` | `directory_permissions contain invalid bits for file access` | Keep permissions within `0777` (file-access bits only). |

## One bbolt file per consumer, with name escaping

The extension writes a separate bbolt file per consuming component, named `<kind>_<type>_<name>` inside `directory`. Only `A-Za-z0-9._-` are safe; any other character (and `~` itself) is escaped as `~<hex unicode>`. Names too long for the OS filename limit are truncated, and the component→filename mapping is logged. So the file set in your storage dir is keyed to your component instances — renaming a receiver/exporter (or its instance name) creates a **new** file and orphans the old state.

## The on-disk name is `receiver_filelog_`, not `receiver_file_log_`

A `file_log` receiver with no instance name produces a bbolt file named **`receiver_filelog_`** — using the deprecated `filelog` type spelling and a trailing `_` for the empty name — even though your config says `file_log`. This is expected; don't go looking for `receiver_file_log_`.

## bbolt corruption and `recreate` can lose or duplicate state

bbolt is generally durable, but on certain corruption panics the extension can either crash the Collector or — with `recreate: true` — rename the corrupted db to `{filename}.{ISO 8601 timestamp}.backup` and start a fresh, empty db so the Collector keeps running (contrib issue #35899). The trade-off: the new db has **no** prior state, so offsets reset (data re-read) or a persistent queue is lost; a receiver may then re-emit already-processed data (duplicates). Treat `recreate` as availability-over-correctness.

## mmap space is not reclaimed until compaction

bbolt is mmap-backed: after a usage spike (e.g. a persistent queue filling during an outage) the file's allocated size grows and does **not** shrink when the data is removed. Disk/mmap footprint stays high until a compaction runs. Enable `compaction.on_start` and/or `compaction.on_rebound` to reclaim it — see [advanced.md](advanced.md).

## `fsync` is a durability-vs-throughput trade-off

`fsync: false` (the default) leaves bbolt's `DB.NoSync` behavior, which is faster but can lose the very latest writes on an unclean shutdown. `fsync: true` forces an `fsync` after every DB write for maximum integrity at a real throughput cost. Only enable it where losing the last few writes is unacceptable.

## Container volume / permission gotchas

The contrib image runs as a **non-root** uid. A mounted host directory (or a Kubernetes volume) must be **writable by that uid**, or the extension can't create/open its bbolt files. In the verification recipe this is why the scratch tree is `chmod -R 777`. In production, set ownership/`fsGroup` appropriately rather than world-writable. `directory_permissions` only affects directories the extension itself creates (with `create_directory: true`); it does not fix a non-writable mount.

## Reading the data for troubleshooting

The bbolt file's contents are readable with the `strings` utility, which is handy for confirming a consumer actually wrote state. This is an implementation detail and may change if the extension moves off bbolt — don't build tooling on it.

## Stability

The extension is **beta**. It is widely used and stable in practice, but the config surface can still change between releases — confirm keys/defaults against the exact contrib version you run.
