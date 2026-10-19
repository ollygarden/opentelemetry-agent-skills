# Compile-time instrumentation with `otelc`

`otelc` (`go.opentelemetry.io/otelc`, current stable release **v1.2.0**, verified 2026-10-19) instruments Go applications
with OpenTelemetry **at compile time**. It wraps the Go toolchain via the compiler's `-toolexec`
hook and injects trampoline hooks (linked with `//go:linkname`) into target functions, so
instrumentation is baked into the binary — **no source-code changes**, and it reaches third-party
dependencies and stdlib packages you don't own. v1.0.0 is retracted because `otelc pin` generated
the pre-v1 module paths in consumer `go.mod` files; use v1.0.1 or later.

## When to use it (vs. the rest of this skill)

| Approach | Reach for it when |
|---|---|
| **`otelc`** (this file) | You can rebuild the app and want zero source changes and automatic coverage of dependencies/stdlib you don't control. The injected OpenTelemetry SDK and instrumentation still have runtime cost. |
| **Hand-written SDK + contrib libraries** ([`instrumentation-libraries.md`](instrumentation-libraries.md), [`api.md`](api.md)) | You need explicit, fine-grained control over spans/metrics, or you cannot change the build toolchain. |

`otelc` is **not** [`opentelemetry-go-instrumentation`](https://github.com/open-telemetry/opentelemetry-go-instrumentation),
which is a separate **eBPF/uprobe runtime** auto-instrumentation project. `otelc` is
compile-time / build-time.

## Requirements

- **Go 1.26+** toolchain for otelc v1.2.0+ (`go 1.26.0` in its [`go.mod`](https://proxy.golang.org/go.opentelemetry.io/otelc/@v/v1.2.0.mod)); v1.1.0 and earlier need Go 1.25+.
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
go.opentelemetry.io/otelc/tool/cmd/otelc@latest`, or the tool-dependency mode above.

**Zero-config:** with no instrumentation file present, `otelc go build` analyzes the dependency
graph, generates a temporary config for the build, and cleans up. Upgrading `otelc` can change what
gets instrumented, so pin the `otelc` version and inspect `.otelc-build/matched.json` in auditable
builds. Committing `otelc pin` output is not yet supported (see below).

## Subcommands

| Subcommand | Purpose |
|---|---|
| `otelc go …` | Instrument and run the go toolchain; everything after `go` is forwarded verbatim. |
| `otelc setup` | Analyze the module, generate instrumentation sources, and write the matched rule set to `.otelc-build/` (needed before Mode 3). |
| `otelc pin` | Create/update the local-workflow `otel.instrumentation.go`. Flags: `--prune` (default on), `--validate`, `--generate`. |
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
| `go.mongodb.org/mongo-driver` (v1/v2) | MongoDB DB spans |
| `k8s.io/client-go` | K8s resource spans |
| `github.com/openai/openai-go` (v1/v2/v3) | GenAI spans |
| `github.com/anthropics/anthropic-sdk-go` | GenAI spans |
| `github.com/segmentio/kafka-go` (consumer & producer) | Kafka messaging spans |
| `github.com/aws/aws-sdk-go-v2` | AWS SDK client spans |
| `github.com/linode/linodego/v2` | HTTP client spans and metrics |
| `log`, `log/slog`, `github.com/sirupsen/logrus` | Trace/span ID log correlation |
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
masks the tool file and the embedded bundle. `--rules`/`OTELC_RULES` are for dev/debugging. The
tool file is the intended explicit model, but `otelc` has not published the instrumentation
submodules: `otelc pin` adds local `replace` directives into `.otelc-build/`. Treat its output as
a local workflow, not a source-controlled production configuration.

**Local explicit selection** uses the standard Go `tools.go` blank-import pattern in a
module-scoped file next to `go.mod`. Let `otelc pin` generate and manage this file:

```go
//go:build tools

package tools

import (
	_ "go.opentelemetry.io/otelc/instrumentation/net/http/server"
	_ "go.opentelemetry.io/otelc/instrumentation/google.golang.org/grpc/client"
	_ "go.opentelemetry.io/otelc/instrumentation/google.golang.org/grpc/server"
)
```

Only blank (`_`) imports are allowed. Do not hand-copy the parent
`instrumentation/google.golang.org/grpc` import shown in an upstream protocol example: it
is not a Go package; the client and server are separate instrumentation modules.

**Runtime tuning:** instrumented binaries embed an auto-initialized SDK that reads the standard
[OTel SDK env vars](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/)
(`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_SAMPLER`, `OTEL_SERVICE_NAME`, …). There is no
`otelc`-specific exporter or sampler destination config. `OTEL_SDK_DISABLED=true` (case-insensitive)
disables the injected SDK. `otelc` exposes these runtime controls:

| Variable | Behavior |
|---|---|
| `OTEL_GO_ENABLED_INSTRUMENTATIONS` | Comma-separated, case-insensitive allowlist of compiled instrumentation names. |
| `OTEL_GO_DISABLED_INSTRUMENTATIONS` | Comma-separated denylist, applied after the allowlist. |
| `OTEL_GLS_MAX_SPANS` | Per-goroutine live span-stack limit; default `1000`. |
| `OTEL_GLS_MAX_SPAN_STATES` | Shared span-lifecycle map limit; default `100000`, evicting the oldest state when full. |
| `OTEL_LOG_LEVEL` | `debug`, `info` (default), `warn`, or `error` for `otelc` runtime logs. |
| `OTEL_GO_SIMPLE_SPAN_PROCESSOR` | Exactly lowercase `true` selects `SimpleSpanProcessor`; other spellings are ignored. |

The enable/disable lists only gate instrumentation already compiled into the binary; build-time
selection still comes from the rule sources above.

**Verify what matched:** `.otelc-build/matched.json` lists every rule that matched; `[]` means
nothing matched (and `otelc` prints `Warning: no instrumentation will be applied`). Enable
`--debug`/`OTELC_DEBUG=1` for `.otelc-build/debug.log`.

## Rule schema (authoring)

Custom rules and new-library instrumentation use YAML rules with `target` (import path; globs and
`$root` supported), optional `version`, `where`/`where.file` predicates, and a `do` action. The
eight rule types: `inject_hooks` (function hook), `add_struct_fields`, `inject_code`, `wrap_call`,
`expand_directive`, `add_file`, `assign_value`, and `set_fields` (composite literals). See the
fetch table for the full grammar.

**v1.1.0 custom-rule migration:** function template variables are fields on Go's `text/template`
dot. Use `{{ .FuncName }}`, `{{ .FuncArgument 0 }}`, and corresponding dotted forms; the pre-v1.1
`{{ FuncName }}` spelling is no longer valid.

## Sources of truth

`otelc` ships stable and evolves; fetch upstream docs rather than trusting a snapshot. Repo:
`open-telemetry/opentelemetry-go-compile-instrumentation` (module `go.opentelemetry.io/otelc`).

| Fact / task | Fetch |
|---|---|
| Latest `otelc` release tag | `gh api repos/open-telemetry/opentelemetry-go-compile-instrumentation/releases/latest -q '.tag_name'` |
| Install, usage modes, managing instrumentations | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/getting-started.md` |
| Full rule schema, glob grammar, and predicates | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/rules.md` |
| Scope, filtering, precedence, runtime tuning, verification | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/configuration.md` |
| Declaring instrumentations via `otel.instrumentation.go` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/external-configuration.md` |
| Add instrumentation for a new library; semconv | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/instrument-guide.md` |
| Internals (`-toolexec`, trampolines, `//go:linkname`, GLS) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/implementation.md` |
| Diagnose why instrumentation was not applied | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-compile-instrumentation/main/docs/troubleshooting.md` |
| Current supported-library set | List the `instrumentation/` tree in the repo (each leaf ships its own rules). |
