# Preferred workflow

How to take a change through this repository, end to end.

Follow this whenever a change adds, renames, moves, or removes a skill, or alters what a skill triggers on, retrieves, or recommends. Typo and prose fixes go straight to a PR — but only outside `SKILL.md` frontmatter. A `description:` edit changes when a skill activates, so it takes the full path however small the wording change looks. Skip a step only when it clearly does not apply, and say so rather than skipping silently.

Reviews run in a fixed order — **local self-review → Agent in CI → human**. Never ask for the next reviewer until the previous one is clean; a human should never be the one to find what a local check or the Agent in CI would have caught.

This is the maintainer- and agent-facing path. [`CONTRIBUTING.md`](../CONTRIBUTING.md) is the contributor-facing guide and the source of truth on the evidence bar, the CLA, and what belongs in this repo; where the two overlap, `CONTRIBUTING.md` wins.

## 1. Track it

Find the existing [issue](https://github.com/ollygarden/opentelemetry-agent-skills/issues); open one if none fits. For a new skill or another large change, open a proposal *before* investing in implementation — scope disagreements are cheap to resolve in an issue and expensive to resolve in a finished PR.

## 2. Branch and isolate

Always branch from freshly fetched `origin/main` — never from the working checkout's `HEAD`, which is sometimes detached or on a stale branch. Prefer a dedicated worktree per task so unrelated work stays isolated and the main checkout stays clean:

```bash
git fetch origin
git worktree add --no-track ../oats-worktrees/<slug> -b <branch> origin/main
```

`--no-track` keeps the new branch from tracking `origin/main`, which reduces the risk of an accidental push to the default branch. It is not a complete guard: `push.default` and `remote.pushDefault` can still route a bare `git push` elsewhere, so check those before relying on it. One task per worktree; one owner per task, including its later review rounds.

## 3. Do the work

Read the target `SKILL.md` and every reference it routes to before editing. Make the smallest defensible change.

Two rules specific to this repository, because they are what makes a skill here worth its context cost:

- **Keep it DRY.** Prefer linking the maintained upstream source over copying its content in. A skill that inlines a table someone else already maintains is stale the moment upstream moves.
- **Stay vendor-neutral and non-opinionated.** Opinions belong in the companion [`skills`](https://github.com/ollygarden/skills) repo. This one holds facts.

## 4. Prove it helps

Any change that can alter when a skill triggers, or what an agent retrieves, recommends, or generates, needs harness evidence. A skill that does not measurably help is context-window cost with no benefit.

The bar is set in [`CONTRIBUTING.md`](../CONTRIBUTING.md#proving-the-skill-helps-harness-results): the same representative prompt(s), cases, repetitions, model, harness, grading rules, and tool access, run in fresh sessions across **three arms** — target skill withheld, current `origin/main` skill, and the proposed PR skill — reported in the PR template's table with transcript links.

Editing an existing skill is the case that makes the third arm non-optional. "Withheld vs mine" can look like a clear win while the change has quietly broken something the shipping version already did; only `origin/main` vs proposed can see that. For a brand-new skill that arm names no revision (`Not present`) and is the same configuration as the withheld arm, so it needs no extra runs — but it still reports results.

If a change genuinely cannot alter agent behavior, say `Not applicable — no harness comparison required` in the PR and why.

## 5. Run the gates

```bash
./bin/validate-skill.sh              # spec conformance + under-500-lines
./bin/check-skill-inventory.py     # skills/, marketplace.json, README table + tree
```

Both run in CI, so a miss here surfaces as a failed check rather than a review comment. `validate-skill.sh` shells out to `skills-ref`, which needs a one-time `uv tool install "$(cat bin/skills-ref.requirement)"` — see [`CONTRIBUTING.md`](../CONTRIBUTING.md).

If your change adds or edits links, reproduce the CI link check locally with the same command it runs, from the repository root ([lychee](https://github.com/lycheeverse/lychee) 0.24.2):

```bash
lychee --no-progress --root-dir "$PWD" --config .github/lychee.toml .
```

It honours `.gitignore`, so scratch files under `local/` are not scanned. Quote any count you report with the command and version that produced it — an error total on its own is not reproducible.

If you touched `tools/otel-agent-tools/` or anything it generates, also run `go build ./cmd/otel-agent-tools` and `go test ./...` from that directory, and **regenerate** the output rather than hand-editing it. If you edit `skills/otel-declarative-config/references/runtime-compatibility.md`, run `check pins` as well (see [`tools/otel-agent-tools/README.md`](../tools/otel-agent-tools/README.md)): it reads the recorded pins from that file's evidence lines, so rewording them can break the daily check.

## 6. Sweep for stale documentation

A change is done when nothing left in the repository describes the old behavior — not when the checks pass. Walk the places that can now be lying, and fix them in this PR rather than a follow-up:

- `README.md` — the "Available Skills" tables and the Repository Structure tree.
- `AGENTS.md` — the skill constraints, the registration checklist, and the local-gate commands. `CLAUDE.md` is a symlink to it, so there is only one file to edit.
- `.claude-plugin/marketplace.json` — enforced by `bin/check-skill-inventory.py`, so let the script find this rather than eyeballing it.
- `CONTRIBUTING.md` and `.github/PULL_REQUEST_TEMPLATE.md` — whenever the change adds or alters a command, a script, or a prerequisite a contributor must install.
- **This file** — `AGENTS.md` points here as the authoritative path, so a new gate or script that is not written down here does not exist for anyone following it.
- Any `references/` doc, or any other skill, that restates a version, path, or config key this change moved.

Be exact about what is enforced and what is not. A doc that promises a check nothing runs is worse than no doc, because the next contributor trusts it; when a constraint is review-enforced rather than automated, say so where the constraint is stated.

## 7. Review it locally, adversarially

Before pushing, attack your own diff — or dispatch a read-only sub-agent to do it. Verify every factual claim against the upstream source instead of trusting the prose, hunt for contradictions with existing guidance, and judge whether the change is actionable and worth its context cost. Ask for prioritized findings with evidence, not praise. Fix the valid ones; state why you dismissed the rest.

For this repository specifically: check that every URL a skill adds actually resolves and points at the *maintained* page, not a snapshot or a redirect to a moved doc.

## 8. Commit and open the PR

Conventional Commits for both the commit message and the PR title, with an optional scope naming the skill (`docs(otel-go): …`). Use the repository template and fill in every section it asks for:

- what changed and why;
- the harness results — prompts, model, harness, and how the outputs differed;
- the gate results, including "no registration change needed";
- agent involvement, if an agent authored or implemented the change;
- limitations and anything you chose not to do.

## 9. Clear the Agent in CI

CodeRabbit reviews PRs here. Read its threads with a thread-aware query — `gh pr view --comments` shows the flat view and hides resolution state:

```bash
gh api graphql -f query='{repository(owner:"ollygarden",name:"opentelemetry-agent-skills"){pullRequest(number:<pr>){reviewThreads(first:100,after:null){pageInfo{hasNextPage endCursor}nodes{id isResolved path line comments(first:10){pageInfo{hasNextPage endCursor}nodes{author{login} body}}}}}}}'
```

Both connections are paged, and each pages **independently** — which is why the query selects `pageInfo` at both levels and the thread `id` alongside it. If `reviewThreads.pageInfo.hasNextPage` is true, repeat with `after:"<endCursor>"` until it is false; an unresolved count taken from one page is not a count of zero. `comments(first:10)` truncates the same way, so page a thread's own comments with its `id` before concluding it was answered:

```bash
gh api graphql -f query='query($id:ID!,$cursor:String){node(id:$id){... on PullRequestReviewThread{comments(first:100,after:$cursor){pageInfo{hasNextPage endCursor}nodes{author{login} body}}}}}' -f id='<thread-id>'
```

Verify each suggestion against the current head, fix the valid ones, and reply with evidence. Dismiss a finding only with a stated reason — a bare resolve is not an answer. Resolve only what is genuinely addressed, push, then wait for a review of the *new* head. If none appears within ~10 minutes, report it and stop rather than polling on or declaring victory. Repeat until checks are green and unresolved threads are zero.

If a review round changes skill content, rerun the harness comparison against the new head and update the results in the PR body. Evidence from an older head does not describe what is about to merge.

## 10. Then request human review

Only once the Agent in CI is clean and every one of its comments is addressed or explicitly dismissed. Summarize what changed since the last look, what the local review and the Agent in CI found, and what you chose not to act on.

External contributors: a maintainer reviews and merges; the CLA bot must be satisfied first.

## 11. Merge behind the gate

Approval binds to the SHA it was given on. GitHub only dismisses a stale approval when branch protection is configured to, and `--match-head-commit` only refuses a *different* head — neither guarantees the approval you have describes the head you are merging. Confirm the review was submitted against the current head SHA before merging, and get fresh approval if it was not. A merge command cannot be recalled once sent and unwinding a merge is disruptive, so do not send it while a review question or ambiguity is open. Run it from the primary checkout, not from inside the task worktree:

```bash
gh pr merge <pr> --repo ollygarden/opentelemetry-agent-skills --squash --match-head-commit <approved-full-sha>
```

## 12. Close out

Close the issue with the merge SHA. Verify the worktree is tracked-clean, remove it with a non-forced `git worktree remove <exact-path>`, and only then delete the local branch. Never prune broadly or touch worktrees you did not create.

## Why each gate exists

Every rule above came from a change that went wrong somewhere in OllyGarden's skill repositories: branching from a detached primary checkout, an approval that had gone stale against a newer head, a merge command sent while a review question was still open, worktree cleanup that risked unrelated worktrees, and a documentation claim (`local/` is gitignored) that was true on the author's machine and false in a fresh clone.

The evidence rules have the same origin. Single runs have read as fixes when they were sampling luck, and an all-pass suite has hidden a defect when the fixture rewarded behavior the skill's own contract forbids.
