# OTel Collector Component Skill Pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Standardize the `otel-collector` skill's component pages on a canonical 8-section template, refactor the two existing pages, add four new cost/quality component pages, and produce a prioritized backlog for the next wave.

**Architecture:** Each Collector component is one self-contained markdown page under `skills/otel-collector/components/<type>.md`, indexed from `skills/otel-collector/SKILL.md`. Content is sourced from authoritative references (the `telemetrydrops/ai-context` mirror, current; or upstream contrib READMEs via WebFetch) — never invented from memory. Work is split into three Linear-tracked phases, each on its own Linear-suggested branch.

**Tech Stack:** Markdown docs; OpenTelemetry Collector (contrib); `telemetrygen` + `debug`/`file` exporter for verification recipes; Linear (Engineering team) for issue/branch tracking.

**Spec:** `docs/superpowers/specs/2026-06-09-otel-collector-component-skills-design.md`

---

## The canonical template (reference for every page task)

Every `components/<type>.md` page has these sections, in order. No frontmatter on component pages.

1. **Header metadata table** — kind, signals, per-signal stability, distributions, `type` name, Go module, upstream README link, rename note if applicable.
2. **Description** — 1–3 paragraphs: what it does + mechanism.
3. **Main use-cases** — "Use when" / "Avoid when" bullets.
4. **Typical config** — minimal working YAML inside `service.pipelines`, **plus** full config-reference table (key, type, default, validation).
5. **Verification** — `telemetrygen` + `debug`/`file` exporter recipe proving documented behavior; cross-reference the `otel-telemetrygen` skill.
6. **Advanced use-cases** — named instances, multi-pipeline, combinations, edge configs.
7. **Known quirks** — gotchas, stability caveats, memory model, validation-error→fix table, anti-patterns, symptom→fix troubleshooting.
8. **Related components** — cross-links.

### Authoritative sources (do not invent config keys)

| Component | Primary source (read this) |
|-----------|----------------------------|
| `log_dedup` | existing page + `~/Projects/src/github.com/telemetrydrops/ai-context/collector/components/logdedupprocessor/` |
| `interval` | existing page + `.../ai-context/collector/components/intervalprocessor/` |
| `cardinality_guardian` | `.../ai-context/collector/components/cardinalityguardianprocessor/README.md` + `METADATA.md` |
| `tail_sampling` | `.../ai-context/sampling/tail-sampling.md` + `.../ai-context/collector/components/tailsamplingprocessor/README.md` |
| `drain` | `.../ai-context/collector/components/drainprocessor/README.md` + `METADATA.md` |
| `redaction` | `.../ai-context/collector/components/redactionprocessor/README.md` + `METADATA.md` |

`telemetrygen` flag reference for Verification recipes: `skills/otel-telemetrygen/SKILL.md` and `references/flags.md`.

### Per-page "definition of done" (the test for every page task)

A page is done when all are true — verify by re-reading the finished file:
- [ ] All 8 sections present, in order, with non-empty content.
- [ ] Every config key in the reference table appears in the authoritative source with matching default (spot-check 100% of keys against the source file/README).
- [ ] The Verification section contains a concrete `telemetrygen` command and a `debug` or `file` exporter config, and names the observable signal that proves the component worked.
- [ ] Stability level and `type` name match the source.
- [ ] No invented keys: every YAML key used appears in the source.

---

## Phase 1 — Refactor existing pages to the template

### Task 1.1: Create the Phase 1 Linear issue and branch

- [ ] **Step 1: Create the Linear issue**

Use the Linear MCP `save_issue` tool (Engineering team, id `06f7f546-7d11-4440-ad82-49fa77de49a0`):
- Title: `otel-collector skill: refactor existing component pages to canonical template`
- Description: link the spec path and summarize Phase 1 (refactor `log_dedup` and `interval` to the 8-section template; add Verification sections).

- [ ] **Step 2: Create the branch from Linear's suggestion**

Read the `branchName` returned by Linear for the new issue (e.g. `juraci/eng-NNN-...`). Then:

```bash
git checkout main
git checkout -b <linear-branchName>
```

Expected: `Switched to a new branch '<linear-branchName>'`

- [ ] **Step 3: Commit a marker (optional, skip if nothing to commit)** — no-op; proceed to Task 1.2.

### Task 1.2: Refactor `log_dedup.md` to the template

**Files:**
- Modify: `skills/otel-collector/components/log_dedup.md`

- [ ] **Step 1: Map current content to the 8 sections**

Current page already has: header metadata table, "When to use", "Configuration" + validation, examples, memory model, telemetry, troubleshooting, validation errors, anti-patterns, related processors. Reorganize into:
- §1 Header metadata table — keep as-is (already present at top).
- §2 Description — promote the one-line intro + "What gets matched"/"What gets emitted" summary into 1–3 paragraphs.
- §3 Main use-cases — the existing "When to use" (use/avoid) bullets.
- §4 Typical config — the "Defaults" example + the existing Configuration table + validation rules + fallback behavior.
- §5 Verification — **new** (Step 2 below).
- §6 Advanced use-cases — "Exclude volatile fields", "Deduplicate only errors", "Whitelist match keys", "Per-source named instances" examples + OTTL-context note.
- §7 Known quirks — memory model, telemetry-metric stability caveat, troubleshooting, validation-errors table, anti-patterns.
- §8 Related components — existing "Related processors".

- [ ] **Step 2: Write the Verification section**

Insert a `## Verification` section after Typical config. Use this content (logs pipeline, contrib distribution required):

````markdown
## Verification

`log_dedup` ships in the `contrib` and `k8s` distributions, so a stock contrib collector can run this.

Config (`dedup-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  log_dedup:
    interval: 5s
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [log_dedup]
      exporters: [debug]
```

Generate many identical log records (see the `otel-telemetrygen` skill):

```bash
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 50 --body "health check ok" --duration 4s
```

**What proves it worked:** the `debug` exporter prints **one** aggregated log per 5s
interval carrying a `log_count` attribute near 50 (plus `first_observed_timestamp` /
`last_observed_timestamp`), not 50 separate records.
````

- [ ] **Step 3: Verify against the definition of done**

Re-read `skills/otel-collector/components/log_dedup.md`. Confirm all 8 sections present and in order; confirm every config-table key (`interval`, `log_count_attribute`, `timezone`, `conditions`, `include_fields`, `exclude_fields`) still matches the source and the existing page.

- [ ] **Step 4: Commit**

```bash
git add skills/otel-collector/components/log_dedup.md
git commit -m "docs(otel-collector): restructure log_dedup page to canonical template

Adds Verification section; reorganizes existing content into the 8-section
template. No information loss.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 1.3: Refactor `interval.md` to the template

**Files:**
- Modify: `skills/otel-collector/components/interval.md`

- [ ] **Step 1: Map current content to the 8 sections**

- §1 Header metadata table — keep as-is.
- §2 Description — the one-line intro + "What it does to each metric type" table + "What lossy means".
- §3 Main use-cases — existing "When to use" (use/avoid) bullets.
- §4 Typical config — "Default" example + Configuration table + the upstream-typo note.
- §5 Verification — **new** (Step 2 below).
- §6 Advanced use-cases — "Keep gauges spiky", "Multiple intervals for different pipelines", "Behavior example" table.
- §7 Known quirks — "State and restart behavior", troubleshooting, anti-patterns.
- §8 Related components — existing "Related".

- [ ] **Step 2: Write the Verification section**

Insert `## Verification` after Typical config:

````markdown
## Verification

`interval` ships in the `contrib` and `k8s` distributions.

Config (`interval-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  interval:
    interval: 10s
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [interval]
      exporters: [debug]
```

Generate cumulative-sum metrics frequently (see the `otel-telemetrygen` skill):

```bash
telemetrygen metrics --otlp-insecure --otlp-endpoint localhost:4317 \
  --metric-type Sum --rate 5 --duration 25s
```

**What proves it worked:** instead of ~5 points/sec reaching `debug`, the exporter prints
the metric roughly once per 10s interval (one latest value per series). Compare against the
same run with the processor removed to see the volume drop.
````

- [ ] **Step 3: Verify against the definition of done**

Re-read the file; confirm 8 sections in order and that keys `interval`, `pass_through.gauge`, `pass_through.summary` match the source.

- [ ] **Step 4: Commit**

```bash
git add skills/otel-collector/components/interval.md
git commit -m "docs(otel-collector): restructure interval page to canonical template

Adds Verification section; reorganizes existing content into the 8-section
template. No information loss.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 1.4: Update `SKILL.md` "Adding a new component" to the canonical template

**Files:**
- Modify: `skills/otel-collector/SKILL.md` (the "Adding a new component to this skill" section, lines ~78–86)

- [ ] **Step 1: Replace the section body**

Replace the numbered list under "## Adding a new component to this skill" with the canonical 8-section template (copy the section list from the top of this plan / the spec), keeping points about adding an index row and updating description trigger phrases.

- [ ] **Step 2: Verify**

Re-read the section. Confirm it names all 8 sections in order and still tells the author to (a) add an index row, (b) update frontmatter trigger phrases when a distinct keyword is introduced.

- [ ] **Step 3: Commit**

```bash
git add skills/otel-collector/SKILL.md
git commit -m "docs(otel-collector): document canonical component-page template in SKILL.md

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 1.5: Open the Phase 1 PR

- [ ] **Step 1: Push and open PR**

```bash
git push -u origin <linear-branchName>
gh pr create --base main --title "otel-collector: canonical component-page template + refactor existing pages" \
  --body "Implements Phase 1 of docs/superpowers/specs/2026-06-09-otel-collector-component-skills-design.md. Closes ENG-NNN."
```

Expected: PR URL printed. Stop here for review before Phase 2.

---

## Phase 2 — Four new cost/quality component pages

### Task 2.1: Create the Phase 2 Linear issue and branch

- [ ] **Step 1: Create the Linear issue** (Engineering team) titled `otel-collector skill: add cardinality_guardian, tail_sampling, drain, redaction pages`, description linking the spec and listing the four components.
- [ ] **Step 2: Create branch from Linear's `branchName`:**

```bash
git checkout main
git checkout -b <linear-branchName>
```

### Task 2.2: Write `components/cardinality_guardian.md`

**Files:**
- Create: `skills/otel-collector/components/cardinality_guardian.md`

- [ ] **Step 1: Read the source**

Read `~/Projects/src/github.com/telemetrydrops/ai-context/collector/components/cardinalityguardianprocessor/README.md` and `METADATA.md` in full.

- [ ] **Step 2: Draft the page to the 8-section template**

Key facts to encode (from METADATA): `type: cardinality_guardian`; metrics only; **Development** stability (contrib v0.152.0); **not in any shipped distribution** (must be built via the OCB builder). Config keys with defaults: `max_cardinality_delta_per_epoch` (100), `epoch_duration_seconds` (300), `tag_only` (false), `never_drop_labels` ([http.status_code, region]), `metric_overrides` (nil), `top_offenders_count` (10), `max_tracker_count` (0), `estimated_cost_per_metric_month` (0.05), `drop_log_max_per_epoch` (10). Internal telemetry metrics: `otelcol_processor_cardinality_*`. **Known quirks** must include the single-writer violation in enforcement mode (`tag_only: false`) corrupting `rate()`/`increase()` on cumulative backends — prefer `tag_only: true` + routing in production.

- [ ] **Step 3: Write the Verification section**

Because it ships in no distribution, the recipe must note building a custom collector via OCB (reference the builder); then:

````markdown
## Verification

`cardinality_guardian` is not bundled in any distribution — build a custom collector that
includes `processor/cardinalityguardianprocessor` via the OpenTelemetry Collector Builder
(OCB) first.

Config (`cardguard-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  cardinality_guardian:
    max_cardinality_delta_per_epoch: 5
    epoch_duration_seconds: 10
    tag_only: true
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [cardinality_guardian]
      exporters: [debug]
```

Emit metrics whose attribute set explodes (many unique values for one label). With
`telemetrygen` use a high `--metric-type Sum` rate while varying attributes (see the
`otel-telemetrygen` skill for attribute flags):

```bash
telemetrygen metrics --otlp-insecure --otlp-endpoint localhost:4317 \
  --metric-type Sum --rate 50 --duration 30s --telemetry-attributes 'runaway="value-N"'
```

**What proves it worked (`tag_only: true`):** once a label crosses the per-epoch delta,
the `debug` exporter shows the offending data points carrying `otel.metric.overflow: true`
while other labels remain intact; the internal metric
`otelcol_processor_cardinality_labels_stripped` increments.
````

Note in the recipe that `telemetrygen`'s attribute variation is limited; if the installed
version cannot vary a label per-series, state that and suggest a `transform`/OTTL upstream
or a small custom emitter as the alternative — do not claim a flag exists if unverified.

- [ ] **Step 4: Verify against the definition of done**, re-reading the file and spot-checking every key against `METADATA.md`.

- [ ] **Step 5: Commit**

```bash
git add skills/otel-collector/components/cardinality_guardian.md
git commit -m "docs(otel-collector): add cardinality_guardian component page

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

### Task 2.3: Write `components/tail_sampling.md`

**Files:**
- Create: `skills/otel-collector/components/tail_sampling.md`

- [ ] **Step 1: Read the source** — `.../ai-context/sampling/tail-sampling.md` and `.../ai-context/collector/components/tailsamplingprocessor/README.md` in full.

- [ ] **Step 2: Draft to the template.** Encode: `type: tail_sampling`; traces only; **Beta**; in contrib/k8s distributions. §4 must cover `decision_wait`, `num_traces`, `expected_new_traces_per_sec`, `policies` (and the common policy types: `always_sample`, `latency`, `status_code`, `string_attribute`, `numeric_attribute`, `rate_limiting`, `probabilistic`, `and`, `composite`, `boolean_attribute`, `ottl_condition`). §7 (Known quirks) MUST cover: tail sampling requires all spans of a trace to reach the **same** collector instance — hence the `loadbalancing` exporter in front when scaling horizontally; memory scales with `num_traces`; `decision_wait` latency; late-arriving spans after a decision are dropped.

- [ ] **Step 3: Verify section** — generate traces and confirm only matching traces survive:

````markdown
## Verification

`tail_sampling` ships in the `contrib` and `k8s` distributions.

Config (`tailsampling-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  tail_sampling:
    decision_wait: 5s
    num_traces: 1000
    policies:
      - name: errors-only
        type: status_code
        status_code:
          status_codes: [ERROR]
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [tail_sampling]
      exporters: [debug]
```

Generate traces, some with error status (see the `otel-telemetrygen` skill — it can set
span status):

```bash
# OK traces — expect these to be dropped by the errors-only policy
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 20
# Error traces — expect these to survive
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 20 --status-code Error
```

**What proves it worked:** after `decision_wait`, the `debug` exporter shows the error
traces and not the OK traces. Confirm the `telemetrygen` version supports `--status-code`;
if not, substitute a policy you can trigger (e.g. `latency` with `--span-duration`).
````

- [ ] **Step 4: Verify** against definition of done (spot-check policy field names against the README — these change between versions).
- [ ] **Step 5: Commit** (`docs(otel-collector): add tail_sampling component page`).

### Task 2.4: Write `components/drain.md`

**Files:**
- Create: `skills/otel-collector/components/drain.md`

- [ ] **Step 1: Read the source** — `.../ai-context/collector/components/drainprocessor/README.md` and `METADATA.md` in full.

- [ ] **Step 2: Draft to the template.** Encode: `type: drain`; logs only; **Alpha**; confirm distributions from `METADATA.md`. §2 explain the Drain clustering algorithm (derives templates like `user <*> logged in from <*>`). §4 config keys with defaults: `tree_depth` (4), `merge_threshold` (0.4), `max_node_children` (100), `max_clusters` (0=unlimited), `extra_delimiters` ([]), `body_field` (""), `template_attribute` ("log.record.template"), `warmup_min_clusters` (0), `storage` (""), `save_interval` (0s). §7 quirks: memory unbounded if `max_clusters: 0` on high log diversity; warmup period before annotation; persistence requires a storage extension.

- [ ] **Step 3: Verify section:**

````markdown
## Verification

Confirm `drain` is in your distribution (check the component's `metadata.yaml`); build via
OCB if it is not yet bundled.

Config (`drain-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  drain:
    template_attribute: log.record.template
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [drain]
      exporters: [debug]
```

Send log records that share structure but differ in values (see the `otel-telemetrygen`
skill):

```bash
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 100 --body "user alice logged in from 10.0.0.1" --duration 5s
```

**What proves it worked:** the `debug` exporter shows each record annotated with a
`log.record.template` attribute holding the clustered template string (e.g.
`user <*> logged in from <*>`). Vary the body across runs to confirm differing records map
to the same template.
````

- [ ] **Step 4: Verify** against definition of done (every key confirmed against the ai-context README/METADATA).
- [ ] **Step 5: Commit** (`docs(otel-collector): add drain component page`).

### Task 2.5: Write `components/redaction.md`

**Files:**
- Create: `skills/otel-collector/components/redaction.md`

- [ ] **Step 1: Read the source** — `.../ai-context/collector/components/redactionprocessor/README.md` and `METADATA.md` in full.

- [ ] **Step 2: Draft to the template.** Encode: `type: redaction`; signals traces (**Beta**), logs + metrics (**Alpha**); in contrib/k8s distributions. §4 config keys: `allow_all_keys` (false), `allowed_keys` ([]), `ignored_keys` ([]), `ignored_key_patterns` ([]), `blocked_key_patterns` ([]), `blocked_values` ([]), `allowed_values` ([]), `hash_function` (none → asterisks; options md5/sha1/sha3/hmac-sha256/hmac-sha512), `summary` (debug/info/silent), `url_sanitizer`, `db_sanitizer`. §7 quirks: with `allow_all_keys: false` and an empty `allowed_keys`, everything is dropped (allowlist semantics); redaction adds audit attributes (`redaction.masked.keys` / counts) depending on `summary`; order of allow vs block evaluation.

- [ ] **Step 3: Verify section:**

````markdown
## Verification

`redaction` ships in the `contrib` and `k8s` distributions.

Config (`redaction-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  redaction:
    allow_all_keys: true
    blocked_values:
      - '4[0-9]{12}(?:[0-9]{3})?'   # Visa-like card numbers
    summary: info
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [redaction]
      exporters: [debug]
```

Generate spans carrying an attribute value that matches the blocked pattern (see the
`otel-telemetrygen` skill for `--span-attributes`/`--telemetry-attributes`):

```bash
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 \
  --traces 5 --telemetry-attributes 'cc_number="4111111111111111"'
```

**What proves it worked:** the `debug` exporter shows the matching value masked (replaced
with asterisks, or hashed if `hash_function` is set) while non-matching attributes pass
through; with `summary: info` the span carries redaction-count audit attributes.
````

- [ ] **Step 4: Verify** against definition of done (confirm key names/defaults against the ai-context README/METADATA).
- [ ] **Step 5: Commit** (`docs(otel-collector): add redaction component page`).

### Task 2.6: Update the `SKILL.md` component index + frontmatter triggers

**Files:**
- Modify: `skills/otel-collector/SKILL.md` (Component index table; frontmatter `description`)

- [ ] **Step 1: Add four index rows** to the component-index table for `cardinality_guardian`, `tail_sampling`, `drain`, `redaction` with their kind/signals/stability/summary.

- [ ] **Step 2: Update the stability note** — the line stating the indexed components are all Alpha is now wrong (Beta, Development present). Generalize it to "see each page's header for its stability."

- [ ] **Step 3: Add distinct trigger keywords** to the frontmatter `description` (e.g. `tail_sampling`, `cardinality_guardian`, `drain`, `redaction`).

- [ ] **Step 4: Verify** — re-read SKILL.md; confirm 6 index rows total, accurate stability column, frontmatter mentions the new keywords.

- [ ] **Step 5: Commit** (`docs(otel-collector): index four new component pages`).

### Task 2.7: Open the Phase 2 PR

- [ ] **Step 1:** `git push -u origin <linear-branchName>` then `gh pr create` referencing the spec and closing the Phase 2 issue. Stop for review.

---

## Phase 3 — Backlog for the next wave

### Task 3.1: Create the Phase 3 Linear issue and branch

- [ ] **Step 1:** Create the Engineering Linear issue titled `otel-collector skill: prioritized backlog for next-wave component pages`.
- [ ] **Step 2:** Create branch from Linear's `branchName`.

### Task 3.2: Write the backlog document

**Files:**
- Create: `docs/superpowers/specs/2026-06-09-otel-collector-component-backlog.md`

- [ ] **Step 1: Write the prioritized backlog.** For each candidate, one short entry: component `type`, kind, signals, why it's high-value for cost/quality, and primary source. Ordered:
  1. `filter` processor
  2. `transform` processor (OTTL-based; complements `otel-ottl`)
  3. `probabilistic_sampler` processor
  4. `attributes` / `resource` processors
  5. `k8s_attributes` processor
  6. `routing` connector
  7. `memory_limiter` processor
  8. `loadbalancing` exporter (prerequisite for scaling `tail_sampling`)

  State explicitly that `deltatocumulative` and `span_metrics` are out of scope.

- [ ] **Step 2: Verify** — every candidate has type/kind/signals/rationale/source; out-of-scope note present.

- [ ] **Step 3: Commit** (`docs(otel-collector): backlog for next-wave component pages`).

### Task 3.3: Create Linear sub-issues for each backlog item

- [ ] **Step 1:** For each of the 8 candidates, create a Linear sub-issue under the Phase 3 issue (Engineering team) titled `otel-collector skill: add <type> component page`, each linking the backlog doc and the canonical template in the spec.

- [ ] **Step 2: Verify** — 8 sub-issues created and parented to the Phase 3 issue.

### Task 3.4: Open the Phase 3 PR

- [ ] **Step 1:** `git push` + `gh pr create` for the backlog doc, referencing the spec and closing the Phase 3 issue.

---

## Self-review notes

- **Spec coverage:** template (Tasks 1.4, all page tasks), Phase 1 refactor (1.2, 1.3), Phase 2 four pages (2.2–2.5) + index (2.6), Phase 3 backlog + sub-issues (3.2, 3.3), Linear-per-phase + Linear-suggested branches (1.1, 2.1, 3.1), source-of-truth strategy (sources table + per-task Step 1 reads). Covered.
- **Out-of-scope honored:** `deltatocumulative` and `span_metrics` excluded (Task 3.2 Step 1).
- **No unverified flag claims:** Verification recipes flag `telemetrygen` flags that must be confirmed against the installed version (Tasks 2.2, 2.3, 2.5) rather than asserting they exist.
