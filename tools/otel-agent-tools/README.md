# otel-agent-tools

## Usage

Run from the repository root. Set `GITHUB_TOKEN` (for example `GITHUB_TOKEN=$(gh auth token)`) to avoid the unauthenticated GitHub API rate limit.

Regenerate the version index from `sdk-sources.json` (one SDK per language) and `component-sources.json` (GitHub-released components):

```bash
go run ./tools/otel-agent-tools/cmd/otel-agent-tools reconcile versions \
  --out skills/otel-sdk-versions/references/generated/otel-version-index.md
```

The index's "Versions last changed" date moves only when a row changes, so rerunning with unchanged upstream versions leaves the file untouched.

Check the declarative-config schema pins in `declarative-config-pins.json` against the dated record. Each pin is read from a file at the repository's latest release and compared with the value the record states; any difference exits non-zero:

```bash
go run ./tools/otel-agent-tools/cmd/otel-agent-tools check pins \
  --doc skills/otel-declarative-config/references/runtime-compatibility.md
```
