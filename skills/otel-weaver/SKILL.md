---
name: otel-weaver
description: OpenTelemetry Weaver registry authoring, codegen, and CI enforcement. Use when adopting Weaver, authoring or reviewing a registry (manifest, attributes, metrics, spans, events), writing Jinja2 templates against the resolved schema, migrating hand-maintained telemetry constants, or wiring `weaver registry check`/`generate`/`diff` into CI.
---

# OpenTelemetry Weaver

Use this skill when an organization wants to define its own semantic conventions on top of upstream OTel and generate language bindings from them.

Usage:
- pair with `otel-semantic-conventions` to decide which attributes already exist upstream and should not be redeclared in the local registry
- use `otel-sdk-versions` only for SDK package selection; Weaver versions are tracked separately at <https://github.com/open-telemetry/weaver/releases>

If a companion skill is unavailable:
- do not stop
- do not rely on memory alone when the guidance can be checked from official sources
- consult the Weaver repo, `schemas/semconv-syntax.v2.md`, and `docs/usage.md` / `docs/validate.md`
- state which fallback you used and leave any unverified item unresolved

## Mental Model

Three moving parts:

1. **Registry** — directory of YAML files. `manifest.yaml` is required and names the registry, declares `semconv_version` and `schema_url`. The rest declare `attributes`, `metrics`, `spans`, `events`, `entities`. The local registry's `semconv_version` is yours to manage; bump it on changes.
2. **Templates** — directory of Jinja2 files plus a `weaver.yaml` per target language describing which templates to run, with what filter, in what `application_mode`, and with what output filename.
3. **Policies** — Rego rules. Built-in OTel policies are the floor; custom policies layer on org rules.

These three replace a hand-rolled `const.go` (or equivalent): const blocks become the registry, the act of writing them becomes codegen, and tribal knowledge becomes policies.

## Non-Negotiable Rules

- Install Weaver via one of the methods documented at <https://github.com/open-telemetry/weaver#install> (release binary, `otel/weaver:vX.Y.Z` Docker image, or the `setup-weaver` GitHub Action). Never `brew install weaver` — that resolves to an unrelated Scribd tool.
- Reference upstream semconv attributes by `ref` rather than redeclaring them. Boundary domains (`http`, `db`, `messaging`, `rpc`, `network`, `gen-ai`, ...) belong in upstream OTel semconv, not in a local registry. Use the language SDK's semconv package for those at runtime.
- Every entry needs `stability`. Weaver refuses to generate without it.
- Use a domain prefix (e.g. `ecommerce.`, `acme.`) for org-local attributes, metrics, and spans.
- Run the language formatter (`gofmt -w`, `prettier`, `ruff format`, ...) on generated output. Jinja whitespace produces multiple blank lines; without formatting, the diff check in CI will fail spuriously.
- Confirm the resolved schema shape before writing a template. Run `weaver registry resolve -r <reg> -f json -o /tmp/r.json` and inspect — input `key`/`name`/`type` fields are renamed at resolve time (see `references/template-authoring.md`).

## Workflow

1. **Install or locate Weaver.** Follow the upstream install instructions at <https://github.com/open-telemetry/weaver#install> — pick a pinned release binary, the `otel/weaver:vX.Y.Z` Docker image, or the `setup-weaver` GitHub Action. Use Docker for CI and reproducible local runs.
2. **Author the registry.** Required: `manifest.yaml` plus at least one of `attributes.yaml` / `metrics.yaml` / `spans.yaml` / `events.yaml`. See `references/registry-authoring.md`.
3. **Author templates.** One target dir per language under `templates/registry/<lang>/` with `weaver.yaml` plus `*.j2`. See `references/template-authoring.md`.
4. **Validate and generate.** `weaver registry check -r ./telemetry/registry/` for fast feedback. `weaver registry generate --registry ./telemetry/registry/ --templates ./telemetry/templates/ <lang> <output-dir>` for codegen. Run the language formatter on the output.
5. **Wire into CI.** Three gates: `check` (schema), `generate` + `git diff --exit-code` (checked-in code is current), `diff` against the base branch (surfaces breaking changes). See `references/ci-integration.md`.

## Gotchas

These cost time and are not obvious from the upstream docs:

1. `brew install weaver` installs the wrong tool. Use GitHub releases or Docker.
2. Generated output is not formatter-clean. Always run the language formatter after `weaver registry generate`.
3. Resolved field names differ from input. Attribute input `key` → resolved `name`. Metric input `name` → resolved `metric_name`. Span input `type` → resolved `id` of the form `span.<type>`, with a flat `name` string equal to the input `name.note`. Always resolve and inspect before writing a template.
4. The `comment` Jinja filter takes a keyword argument: `attr.brief | comment(format="go")`. It already emits the `// ` prefix; do not add another.
5. There is no prebuilt jq filter for spans. Use:
   ```
   semconv_signal("span"; {}) | group_by(.root_namespace) | map({root_namespace: .[0].root_namespace, spans: . | sort_by(.id)})
   ```
   It **must be single-quoted** in `weaver.yaml`; the colons in jq syntax collide with YAML mapping rules otherwise.
6. `weaver registry check` emits "File format `definition/2` is not yet stable" on every run as of 0.22.1. This is normal; do not treat it as a failure.
7. `--future` is opt-in but breaks today on `definition/2`. Note this in CI guidance and re-enable once the format goes stable.
8. CLI argument ordering for `generate`: target directory name is positional **after** `--registry` and `--templates`; the output directory follows. `--templates` points at the **parent** that contains target dirs, not at the language-specific subdir.
9. Span name in registry vs. runtime: required schema fields are `type`, `kind` (`client`/`server`/`producer`/`consumer`/`internal`), `brief`, `stability`, and a structured `name: { note: "..." }`. For internal business spans, putting the dotted type identifier in `name.note` and using the resolved `span.name` string at runtime is clean.
10. **What does NOT belong in your local registry.** DB, HTTP, messaging, RPC, network, GenAI, and similar boundary spans/attributes follow upstream OTel semconv. Until upstream is pulled in as a manifest dependency, instrumentation for those should reference the language SDK's semconv package directly. This is the most common modeling mistake.
11. Drop the `.total` suffix from counter names — OTel naming has moved away from it.
12. Use seconds (`s`) for duration histograms, not milliseconds. Migrating from `ms` is a natural step when authoring the schema; flag it.

## References To Load On Demand

- registry YAML field reference: `references/registry-authoring.md`
- Jinja2 patterns, jq filters, resolved-shape cheat sheet: `references/template-authoring.md`
- ready-to-lift GitHub Actions example: `references/ci-integration.md`
- hand-maintained-constants → registry walkthrough: `references/migration-playbook.md`
- semantic conventions skill: `otel-semantic-conventions`
- manual instrumentation skill: `manual-instrumentation`

## Out Of Scope

These are natural follow-ups but not part of this skill:
- publishing the registry as a versioned artifact for downstream consumers
- declaring upstream semantic-conventions as a manifest dependency
- `weaver registry live-check` against a local collector
- custom Rego policies beyond the built-ins
- helper-function codegen (`MyMetricName(meter)` wrappers)

## Verification Contract

If you authored or modified a Weaver registry, templates, or CI integration:
- re-open the changed files before finishing
- run `weaver registry check` against the registry and capture the result
- run `weaver registry generate` and the language formatter, then verify `git diff --exit-code` is clean
- confirm each applicable item with codebase evidence

Report the final check with:
- `[x]` completed
- `[~]` not applicable, with a reason
- `[ ]` unresolved

Use these items:
- registry has `manifest.yaml` with `name`, `semconv_version`, `schema_url`
- every entry has `stability`
- org-local attributes/metrics/spans use a domain prefix
- no boundary-domain (http/db/messaging/rpc/network/gen-ai) entries duplicated locally
- counter names have no `.total` suffix
- duration histograms use `s` (seconds)
- templates use jq filters that match the resolved schema, with the spans filter single-quoted
- generated output is formatter-clean
- CI runs `check`, `generate` + `git diff --exit-code`, and `diff` against the base branch
- changed files were re-read
- remaining risks or gaps are stated
