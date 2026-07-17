# Compile-time instrumentation with `otelc`

`otelc` (`go.opentelemetry.io/otelc`, first stable release **v1.0.0**) instruments Go applications
with OpenTelemetry **at compile time**. It wraps the Go toolchain via the compiler's `-toolexec`
hook and injects trampoline hooks (linked with `//go:linkname`) into target functions, so
instrumentation is baked into the binary — **no source-code changes**, and it reaches third-party
dependencies and stdlib packages you don't own.

## When to use it (vs. the rest of this skill)

| Approach | Reach for it when |
|---|---|
| **`otelc`** (this file) | You can rebuild the app and want zero code changes, zero runtime overhead, and automatic coverage of dependencies/stdlib you don't control. |
| **Hand-written SDK + contrib libraries** ([`instrumentation-libraries.md`](instrumentation-libraries.md), [`api.md`](api.md)) | You need explicit, fine-grained control over spans/metrics, or you cannot change the build toolchain. |

`otelc` is **not** [`opentelemetry-go-instrumentation`](https://github.com/open-telemetry/opentelemetry-go-instrumentation),
which is a separate **eBPF/uprobe runtime** auto-instrumentation project. `otelc` is
compile-time / build-time.

## Requirements

- **Go 1.25+** toolchain (`go 1.25.0` in the project's `go.mod`).
- **Go 1.24+** for the tool-dependency workflow (`go get -tool` / `go tool otelc`).
- Operates on the module containing your `go.mod`.

## Usage modes

All three produce an instrumented binary; they differ in how `otelc` wires into the build.

```bash
# Mode 1 — wrap the build (simplest). Prefix the normal go command.
otelc go build -o bin/app .
otelc go test ./...

# Mode 2 — tool dependency (Go 1.24+, reproducible: version tracked in go.mod)
go get -tool go.opentelemetry.io/otelc/tool/cmd/otelc
go tool otelc go build -o bin/app .

# Mode 3 — toolexec drop-in via GOFLAGS (build command owned by a Makefile/CI)
otelc setup                                             # run once; re-run when deps change
export GOFLAGS="${GOFLAGS} '-toolexec=otelc toolexec'"  # single quotes are required
go build -o myapp .
```

Obtain the binary by building from source (`make build` in the upstream repo), `go install
go.opentelemetry.io/otelc/tool/cmd/otelc`, or the tool-dependency mode above.

**Zero-config:** with no instrumentation file present, `otelc go build` analyzes the dependency
graph, generates a temporary config for the build, and cleans up. Convenient, but **not
reproducible** — upgrading `otelc` can silently change what gets instrumented. For auditable builds,
pin instrumentations explicitly (see below).

## Subcommands

| Subcommand | Purpose |
|---|---|
| `otelc go …` | Instrument and run the go toolchain; everything after `go` is forwarded verbatim. |
| `otelc setup` | Analyze the module, generate instrumentation sources, write the matched rule set to `.otelc-build/` (needed before Mode 3). Also runs the interactive wizard. |
| `otelc pin` | Create/update `otel.instrumentation.go` to pin instrumentation packages. Flags: `--prune` (default on), `--validate`, `--generate`. |
| `otelc cleanup` | Delete `.otelc-build/` and generated files. |
| `otelc version` | Print the tool version (`--verbose` adds the Go runtime version). |
| `otelc toolexec …` | Hidden interceptor invoked by `-toolexec=otelc toolexec`; never call it directly. |

**Flag ordering:** global flags (`--rules`, `--debug`/`-d`, `--work-dir`/`-w`) must come **before**
the `go` subcommand. `otelc go` and `otelc toolexec` skip flag parsing, so anything after them goes
straight to the Go toolchain — `otelc go build --rules …` sends `--rules` to `go build` and fails.

## Supported libraries (v1)

Verified against the upstream `instrumentation/` tree. The active set is `otelc`'s embedded bundle;
confirm against the tree when a version changes.

| Library | Signals |
|---|---|
| `net/http` (client & server) | HTTP spans |
| `google.golang.org/grpc` (client & server) | RPC spans |
| `database/sql` | DB client spans |
| `github.com/gin-gonic/gin` | HTTP server spans |
| `github.com/redis/go-redis/v9` | Redis DB spans |
| `go.mongodb.org/mongo-driver` | MongoDB DB spans |
| `k8s.io/client-go` | K8s resource spans |
| `github.com/openai/openai-go` (v1/v2/v3) | GenAI spans |
| `github.com/segmentio/kafka-go` (consumer & producer) | Kafka messaging spans |
| `log/slog` | Log records |
| `github.com/sirupsen/logrus` | Log records |
| Go runtime | Runtime metrics |

## Selecting & configuring instrumentation

**Rule sources, highest priority first — there is NO merging; each source entirely replaces the
ones below it:**

| Priority | Source |
|---|---|
| 1 | `OTELC_RULES` env var (file / dir / comma-separated list) |
| 2 | `--rules` flag (same format; used only when `OTELC_RULES` is unset) |
| 3 | Tool file `otel.instrumentation.go` (alias `otelc.tool.go`) |
| 4 | Embedded default bundle |

This is the top cause of "nothing got instrumented": an `OTELC_RULES`/`--rules` override silently
masks the tool file and the embedded bundle. `--rules`/`OTELC_RULES` are for dev/debugging;
pin via the tool file for production.

**Explicit pinning** uses the standard Go `tools.go` blank-import pattern in a module-scoped file
next to `go.mod`:

```go
//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc/instrumentation/net/http/server"
	_ "go.opentelemetry.io/otelc/instrumentation/google.golang.org/grpc"
)
```

Only blank (`_`) imports are allowed; run `go mod tidy` after editing, or let `otelc pin` manage it.

**Runtime tuning:** instrumented binaries embed an auto-initialized SDK that reads the standard
[OTel SDK env vars](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/)
(`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_SAMPLER`, `OTEL_SERVICE_NAME`, …). There is no
`otelc`-specific exporter/sampler config. The one `otelc`-specific knob is `OTEL_GLS_MAX_SPANS`
(goroutine-local-storage span-stack depth, for instrumentation that doesn't thread `context.Context`).

**Verify what matched:** `.otelc-build/matched.json` lists every rule that matched; `[]` means
nothing matched (and `otelc` prints `Warning: no instrumentation will be applied`). Enable
`--debug`/`OTELC_DEBUG=1` for `.otelc-build/debug.log`.

## Rule schema (authoring)

Custom rules and new-library instrumentation use YAML rules with `target` (import path; globs and
`$root` supported), optional `version`, `where`/`where.file` predicates, and a `do` action. The
seven rule types: `inject_hooks` (function hook), `add_struct_fields`, `inject_code`, `wrap_call`,
`expand_directive`, `add_file`, `assign_value`. See the fetch table for the full grammar.

## Sources of truth

`otelc` ships stable and evolves; fetch upstream docs rather than trusting a snapshot. Repo:
`open-telemetry/opentelemetry-go-compile-instrumentation` (module `go.opentelemetry.io/otelc`).

| Fact / task | Fetch |
|---|---|
| Latest `otelc` release tag | `gh api repos/open-telemetry/opentelemetry-go-compile-instrumentation/releases/latest -q '.tag_name'` |
| Install, usage modes, managing instrumentations | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/getting-started.md` |
| Full rule schema (all 7 types, glob grammar, predicates) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/rules.md` |
| Scope, filtering, precedence, runtime tuning, verification | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/configuration.md` |
| Declaring instrumentations via `otel.instrumentation.go` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/external-configuration.md` |
| Add instrumentation for a new library; semconv | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/instrument-guide.md` |
| Internals (`-toolexec`, trampolines, `//go:linkname`, GLS) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/implementation.md` |
| Diagnose why instrumentation was not applied | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/troubleshooting.md` |
| Current supported-library set | List the `instrumentation/` tree in the repo (each leaf ships its own rules). |
