# `otel-telemetry-emissions` harness results

One section per harness run, newest first.

## PR #242 — written-counts response headers

Run with Claude Code 2.1.259 driving `claude-sonnet-5`. Three cases, three arms, three
repetitions each: 27 runs. Cases 1 and 2 ran on 2026-09-04. Case 3 was added on 2026-09-05,
after a maintainer asked this PR to also carry the v0.160.0 file. Every repetition used a
fresh session, an isolated `CLAUDE_CONFIG_DIR`, an empty working directory outside this
repository, and the same tool access: `Read`, `Glob` and `Grep` only. No arm had network
access.

Cases 1 and 2 ran before v0.160.0 was added, so their proposed arm held only v0.154.0 to
v0.159.0. Those six files are byte-identical in both batches and both cases pin v0.159.0, but
that arm is not literally the one case 3 ran against. Cases 1 and 2 were not re-run.

Every arm — not only the withheld one — ran inside the same `bwrap` sandbox, which mounted an
empty `tmpfs` over the checkout of this repository. See "Method corrections" below for why.
The `evals/` directory was removed from the skill copy in all three arms, so no arm could read
the expectations it was graded against. Only the target skill differed between arms.

Grading was done by reading each final answer against the prose expectations in `evals.json`.
No grader script was used. No repetition was retried, and no repetition was excluded by
outcome.

| Arm | Target-skill revision / state | Cases × reps | Pass | Fail | Unknown |
|---|---|---:|---:|---:|---:|
| Target skill withheld | `Withheld` | 3 × 3 | 2 | 7 | 0 |
| Current `origin/main` skill | `1bf950f6101e0c66840dce5c873070ff6f47b582` | 3 × 3 | 6 | 3 | 0 |
| Proposed PR skill | `995864d7372a09b3257784a554dc58672784b7a9` | 3 × 3 | 9 | 0 | 0 |

### Per-attempt grading — `prometheusremotewrite-v0159-metrics`

`A` is all 16 metric names, `B` is the `wal` condition with the v0.159.0 `exporter` attribute,
`C` is the Remote Write 2.0 gate and config, and `D` is the instrument types and units.

| Attempt | A | B | C | D | Result |
|---|---|---|---|---|---|
| `case1-withheld-1` | fail | fail | fail | fail | fail |
| `case1-withheld-2` | fail | fail | fail | fail | fail |
| `case1-withheld-3` | fail | fail | fail | fail | fail |
| `case1-current-1` | pass | pass | pass | pass | pass |
| `case1-current-2` | pass | pass | pass | pass | pass |
| `case1-current-3` | pass | pass | pass | pass | pass |
| `case1-proposed-1` | pass | pass | pass | pass | pass |
| `case1-proposed-2` | pass | pass | pass | pass | pass |
| `case1-proposed-3` | pass | pass | pass | pass | pass |

### Per-attempt grading — `prometheusremotewrite-written-headers`

`A` is the `Samples-Written` header, `B` is `Histograms-Written`, `C` is `Exemplars-Written`,
and `D` is the one-header-per-metric consequence for a backend that omits the exemplars header.

| Attempt | A | B | C | D | Result |
|---|---|---|---|---|---|
| `case2-withheld-1` | fail | fail | fail | pass | fail |
| `case2-withheld-2` | pass | pass | pass | pass | pass |
| `case2-withheld-3` | pass | pass | pass | pass | pass |
| `case2-current-1` | pass | pass | pass | pass | pass |
| `case2-current-2` | pass | pass | pass | pass | pass |
| `case2-current-3` | pass | pass | pass | pass | pass |
| `case2-proposed-1` | pass | pass | pass | pass | pass |
| `case2-proposed-2` | pass | pass | pass | pass | pass |
| `case2-proposed-3` | pass | pass | pass | pass | pass |

### Per-attempt grading — `prometheusremotewrite-version-coverage`

`A` is answering from a source pinned to v0.160.0, `B` is all 16 metric names, `C` is the
`wal` condition with the `exporter` attribute, and `D` is the Remote Write 2.0 gate together
with the response header each `written_*` metric reads.

| Attempt | A | B | C | D | Result |
|---|---|---|---|---|---|
| `case3-withheld-1` | fail | fail | fail | fail | fail |
| `case3-withheld-2` | fail | fail | fail | fail | fail |
| `case3-withheld-3` | fail | fail | fail | fail | fail |
| `case3-current-1` | fail | pass | pass | fail | fail |
| `case3-current-2` | fail | pass | pass | fail | fail |
| `case3-current-3` | fail | pass | pass | fail | fail |
| `case3-proposed-1` | pass | pass | pass | pass | pass |
| `case3-proposed-2` | pass | pass | pass | pass | pass |
| `case3-proposed-3` | pass | pass | pass | pass | pass |

### What differed

**Case 3 separates the arms.** The proposed skill passed 12 of 12 expectations, `origin/main`
6 of 12, and the withheld arm 0 of 12. The whole gap is version coverage. `origin/main` has no
v0.160.0 file, so all three of its runs fell back to v0.159.0 and marked the answer unverified
— the behaviour `SKILL.md` prescribes when no file matches the target version exactly. The
behaviour was correct and the data was absent. The proposed skill matched v0.160.0 exactly and
cited its `commit_sha` in all three runs. The withheld arm answered none of the three, and
asked for network access rather than reply from recall.

**Cases 1 and 2 are a null result.** The proposed arm scored the same as the `origin/main`
arm: 12/12 expectations on case 1 and 12/12 on case 2. On those two cases, the header fix does
not improve agent output.

**Case 2 cannot separate the arms.** Two of the three *withheld* runs named all three headers
correctly with no skill installed. The model holds these header names in prior knowledge, so a
prompt that asks for them directly is answered with or without the registry. Only
`case2-withheld-1` got them wrong, inverting the word order to
`X-Prometheus-Remote-Write-Written-Samples`. The case is retained because it documents the fix
and will catch a future regression in the data file, not because it discriminates today.

**The measurable difference sits in case 1, and no expectation covers it.** Case 1 does not ask
where a metric's value comes from, so the agent reports whatever the data file says. On
`origin/main`, two of three runs reproduced `X-Prometheus-Remote-Write-Samples-Written`-family
verbatim — the wording this PR removes — while the third generalised to the correct
`X-Prometheus-Remote-Write-*-Written` pattern. On the proposed skill, two of three runs gave
the exact per-metric mapping and the third did not mention headers at all. No run on the
proposed skill reproduced the misleading wording, and no run on `origin/main` gave the correct
per-metric mapping.

**The defect reproduced itself in case 3, unprompted.** `case3-current-2` wrote that the
values are "read from `X-Prometheus-Remote-Write-Samples-Written`-family response headers" —
the wording this PR removes — in answer to a question that never mentioned headers.
`case3-current-1` omitted the headers entirely and `case3-current-3` generalised to the
correct `X-Prometheus-Remote-Write-*-Written` pattern. All three proposed runs gave the exact
per-metric mapping. Across cases 1 and 3, three of six `origin/main` runs copied the wrong
string and no proposed run did.

So the two halves of this PR do different work. The header fix stops a wrong string from
reaching answers, and adds no capability, which is why cases 1 and 2 show parity. The v0.160.0
file adds coverage, and case 3 measures it.

**No regression.** Every expectation that passed on `origin/main` also passed on the proposed
skill, in all three repetitions of all three cases.

### Failing runs kept, and why

All ten failures are preserved. None was retried.

- `case1-withheld-1`, `case1-withheld-2` and `case1-withheld-3` had no registry to read.
  Rep 1 answered about `target_info` and `otel_scope_info`, which the question did not ask
  about; reps 2 and 3 named two of the 16 metrics from recall and flagged their own confidence
  as moderate. All three offered to fetch the upstream source. Genuine misses.
- `case2-withheld-1` gave the header names with the word order inverted, and passed only the
  fourth expectation.
- `case3-withheld-1`, `case3-withheld-2` and `case3-withheld-3` declined to answer. Each asked
  for `WebFetch` or `gh api` access to read the upstream source at the v0.160.0 tag. Run 2
  reported that it had checked the `otel-collector` skill and found it "verified only against
  v0.159.0", documenting configuration rather than the exporter's own self-observability
  metrics. Genuine misses, and evidence that no other skill in this repository carries the
  data.
- `case3-current-1`, `case3-current-2` and `case3-current-3` failed the version expectation
  because `origin/main` has no v0.160.0 file. Each fell back to v0.159.0 and said so, which is
  what `SKILL.md` instructs. The runs are graded as failures against the case, not as faults
  in the shipping skill.

### Method corrections

Two earlier batches of 18 runs, and one batch of three, were discarded. All are preserved
outside the repository. They are recorded here because they changed the method, not because
they changed a score.

1. **The first batch leaked.** The withheld arm removed the skill from its config directory,
   but the repository still sat on disk and `Read` accepts absolute paths. One run globbed `/`,
   found this checkout, and answered case 1 from the working tree — reproducing the corrected
   headers and the file's own `commit_sha` and `last_verified`. The other 17 runs of that batch
   were clean. Every arm now runs under `bwrap` with the checkout hidden, so the arms stay
   identical and no arm can reach it.
2. **Case 2 was rewritten after its first run.** The original prompt opened with a symptom —
   "`written_exemplars` stays at zero while `written_samples` increases" — and scored 0/9 on
   the proposed arm. The transcripts showed why: all three runs answered from the
   `otel-collector` skill's `prometheus_remote_write` pages and never opened the emissions
   registry. The case measured which skill a troubleshooting question routes to, not the data
   under test. It was rewritten to lead with the lookup — "which response header does each of
   the three `written_*` metrics read its value from" — and a fourth expectation was added for
   the histograms header, which the new prompt brings into scope. The rewrite fixed retrieval
   and produced the null result above. The instrument was changed once, for a stated reason;
   no case was re-run in the hope of a better number.

3. **The case-3 proposed arm was rebuilt.** Its first batch ran against an arm copied from the
   working tree, which also held 28 data files that are not part of this pull request —
   other components' inventories that are untracked locally. The `origin/main` arm came from a
   clean worktree and held none of them, so the two arms differed by more than the target
   change. The arm was rebuilt from commit `995864d7372a09b3257784a554dc58672784b7a9` with
   `git archive`, checked against that commit with `diff -r` and found identical, and the three
   proposed runs were repeated. The grade did not change: 12/12 before and after. The discarded
   outputs are preserved outside the repository. One wording difference is worth recording —
   the discarded batch held a run that wrote "All 14 metrics" in a closing note while its own
   table listed all 16 correctly; no run in the clean batch did.

A further method fix applied to the final batch: the skill's own `evals/` directory was removed
from all three arms. The proposed arm's copy carried the expected header names inside its
expectations while the `origin/main` copy did not, so leaving it in place would have graded the
proposed arm against its own answer key.

### Limitations

- Three cases, three repetitions per arm. Three repetitions is the minimum `CONTRIBUTING.md`
  asks for.
- The proposed arm that cases 1 and 2 ran against was overwritten while case 3 was being set
  up, so its exact file composition can no longer be checked. The case-3 arm is verified
  against commit `995864d7372a09b3257784a554dc58672784b7a9`; the cases 1-2 arm is not, and it
  is stated here rather than assumed to have been clean.
- One model and one harness. The result may differ on another model.
- `claude -p` prints the final answer only. The saved output holds no tool calls and no
  reasoning, so a reviewer sees what the agent concluded, not how it got there. The leak in the
  first batch was found in the session logs, not in these outputs.
- All 170 other skills were present in every arm. The set was identical across arms, so it
  cannot explain a difference between them — but it does explain case 2's original failure,
  since `otel-collector` also covers this component.
- The cases test one component at two versions, v0.159.0 and v0.160.0. They do not test the
  other five version files this PR corrects.
- The strongest observation in this report — the wording propagating from the data file into
  answers — is not covered by any expectation. Adding one to case 1 would penalise a correct
  answer for omitting something the prompt never requested.

### Transcripts

The 27 final answers hold no credentials, no local paths, no customer data and no private
repository content. They were checked for all four before this summary was written.

## PR #235 — prometheusremotewriteexporter inventory

Run on 2026-09-02 with Claude Code 2.1.258 driving `claude-sonnet-5`. Every repetition used a
fresh session, an isolated `CLAUDE_CONFIG_DIR`, an empty working directory outside this
repository, and the same tool access: `Read`, `Glob` and `Grep` only. No arm had network
access, so no arm could read the upstream source. The prompt, the model and the tool set were
identical in all three arms. Only the target skill changed.

The withheld arm removed `otel-telemetry-emissions` alone. The other 170 skills stayed in
place in every arm.

A deterministic grader marked a repetition passing only when all four expectations in
`evals.json` were present in the final answer. No repetition was retried, and no repetition
was excluded.

| Arm | Target-skill revision / state | Cases × reps | Pass | Fail | Unknown |
|---|---|---:|---:|---:|---:|
| Target skill withheld | `Withheld` | 1 × 3 | 0 | 3 | 0 |
| Current `origin/main` skill | `55e1d7e411b5cddefb3ae95f128b220da8354044` | 1 × 3 | 0 | 3 | 0 |
| Proposed PR skill | `029f34324742a15f49eea0e181947efb83e8ae1c` | 1 × 3 | 3 | 0 | 0 |

### Per-attempt grading

`A` is all 16 metric names, `B` is the `wal` condition with the v0.159.0 `exporter` attribute,
`C` is the Remote Write 2.0 gate and config, and `D` is the instrument types and units.

| Attempt | Names found | A | B | C | D | Result |
|---|---:|---|---|---|---|---|
| `withheld-1` | 2/16 | fail | fail | fail | fail | fail |
| `withheld-2` | 0/16 | fail | fail | fail | fail | fail |
| `withheld-3` | 0/16 | fail | fail | fail | fail | fail |
| `current-1` | 0/16 | fail | fail | fail | fail | fail |
| `current-2` | 0/16 | fail | fail | fail | fail | fail |
| `current-3` | 0/16 | fail | fail | fail | fail | fail |
| `proposed-1` | 16/16 | pass | pass | pass | pass | pass |
| `proposed-2` | 16/16 | pass | pass | pass | pass | pass |
| `proposed-3` | 16/16 | pass | pass | pass | pass | pass |

### What differed

**The withheld arm could not do the task.** `withheld-1` gave two generic exporter metrics,
`otelcol_exporter_sent_metric_points` and `otelcol_exporter_send_failed_metric_points`, which
every exporter emits and which the question did not ask for. It then named only two of the 16
component metrics, with no type, unit or attribute. It inferred from the `wal.lag_record_frequency`
config key that a WAL lag metric must exist, and refused to name it. `withheld-2` and
`withheld-3` gave no answer at all. They asked for permission to fetch the upstream source.

**The `origin/main` arm failed correctly.** `current-1` and `current-2` reported that the
component is not in the registry, and refused to guess. This is the behavior that
`SKILL.md` asks for: "If the component or version is not covered at all, say so — do not infer
telemetry from semantic conventions." So these are not defects in the shipping skill. They are
the shipping skill telling the truth about a coverage gap. `current-3` asked to fetch the
source, like the withheld arm.

**The proposed arm answered completely, three times out of three.** Every repetition named all
16 metrics, separated the four unconditional metrics from the nine WAL metrics and the three
Remote Write 2.0 metrics, gave the instrument type and unit for each, and stated that v0.159.0
adds the `exporter` attribute to the WAL metrics that earlier versions emit unattributed. Two
repetitions also gave the histogram bucket boundaries.

**No regression.** The proposed arm did not lose any behavior that the `origin/main` arm had.
The `origin/main` arm gave no component data to lose. The "say so when not covered" rule stays
in the skill and is unchanged by this PR.

### Failing runs kept, and why

All six failures are preserved. None was retried.

- `withheld-2`, `withheld-3` and `current-3` stopped to ask for network permission instead of
  answering. This is a genuine miss, not a harness failure. The agent had no source and would
  not guess. The exit code was 0 in each.
- `withheld-1`, `current-1` and `current-2` gave an answer that failed all four expectations.

### Limitations

- One case, three repetitions per arm. This is the minimum that `CONTRIBUTING.md` asks for.
- One model and one harness. The result may differ on another model.
- `claude -p` prints the final answer only. The saved output holds no tool calls and no
  reasoning, so a reviewer sees what the agent concluded, not how it got there.
- All 170 other skills were present in every arm. Their descriptions add trigger surface. The
  set was identical across arms, so it cannot explain the difference between them.
- The case tests retrieval of one component at one version. It does not test the accuracy of
  the other 5 versions in this PR, and it does not test any other component.

### Transcripts

The nine final answers hold no credentials, no local paths, no customer data and no private
repository content. They were checked for all four before this summary was written.
