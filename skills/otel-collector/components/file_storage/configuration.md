# `file_storage`: configuration

All keys live under the extension instance (e.g. `extensions: { file_storage: { … } }`), and the extension must be listed under `service.extensions:` to load. Facts below are traced to the `v0.154.0` contrib source (`config.go`, `factory.go`, `default_others.go`).

## Top-level keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `directory` | string | `/var/lib/otelcol/file_storage` (non-Windows) / `%ProgramData%\Otelcol\FileStorage` (Windows) | Dedicated data directory holding the per-consumer bbolt files. **Must already exist** unless `create_directory: true`. |
| `timeout` | duration | `1s` | Max time to wait for a file lock before failing the operation. Rarely needs changing. |
| `fsync` | bool | `false` | When `true`, force an `fsync` after each DB write (integrity over performance; flips bbolt's `DB.NoSync`). |
| `create_directory` | bool | `false` | When `true`, create the data (and compaction) directory if missing instead of failing validation. |
| `directory_permissions` | string (octal) | `"0750"` (rwxr-x---) | Permissions used when creating directories, minus the process umask. Only relevant when `create_directory: true`. |
| `recreate` | bool | `false` | When `true`, on certain bbolt corruption panics the extension renames the corrupted db to `{filename}.{ISO 8601 timestamp}.backup` and starts a fresh db so the Collector keeps running. State is lost / may be duplicated (contrib issue #35899). |

The platform-specific `directory` defaults come from `default_others.go` (and its Windows counterpart).

## `compaction`

bbolt is mmap-backed: when usage spikes, the file's allocated size grows and does **not** shrink on its own. Compaction rewrites the db to reclaim that freed-but-still-allocated space.

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `compaction.on_start` | bool | `false` | Compact once at Collector start. |
| `compaction.on_rebound` | bool | `false` | Online compaction triggered when rebound thresholds are met (see below). |
| `compaction.directory` | string | same as the default data `directory` | Temp directory used as a midstep during compaction. |
| `compaction.max_transaction_size` | int | `65536` | Max number of items in a single compaction transaction; `0` ignores transaction sizes. **Must be ≥ 0.** |
| `compaction.rebound_needed_threshold_mib` | int | `100` | When allocated size exceeds this, the "compaction needed" flag is armed. |
| `compaction.rebound_trigger_threshold_mib` | int | `10` | When the flag is armed and allocated size drops below this, compaction begins and the flag clears. |
| `compaction.check_interval` | duration | `5s` | How often rebound conditions are checked. **Must be > 0 when `on_rebound: true`.** |
| `compaction.cleanup_on_start` | bool | `false` | Remove leftover `tempdb*` files from the compaction directory at start (left behind if a prior process was killed mid-compaction). |

**Rebound (online) compaction** reclaims space after a transient spike drains: the flag arms only once allocated size has exceeded `rebound_needed_threshold_mib`, then compaction fires when usage drops back below `rebound_trigger_threshold_mib`. See [advanced.md](advanced.md) for tuning.

## Per-consumer files and naming

The extension creates **one bbolt file per consuming component**, named `<kind>_<type>_<name>` inside `directory`:

- A `file_log` receiver with no instance name → `receiver_filelog_` (note the on-disk name uses the deprecated `filelog` type spelling and a trailing `_` for the empty name; see [quirks.md](quirks.md)).
- A named `journald/myservice` receiver → `receiver_journald_myservice`.

Only `A-Za-z0-9._-` are safe in the name; any other character (including `~` itself) is escaped as `~<hex unicode>`. Over-long names are truncated to stay within OS filename limits, and the component→filename mapping is written to the Collector logs.

## Directory permissions and umask

`directory_permissions` (default `"0750"`) is applied **only** when `create_directory: true` creates a directory, and the effective mode is the configured value minus the process umask. The value must be valid octal with no bits outside `0777` — see the validation table in [quirks.md](quirks.md).

## Minimal example

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/storage
    create_directory: true        # create it on start if missing
service:
  extensions: [file_storage]      # must be listed here to load
```
