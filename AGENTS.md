# Repository Guidelines

## Agent skills

### Issue tracker

Issues and PRDs are tracked as local Markdown files under `.scratch/<feature-slug>/`; external PRs are not a triage surface. See `docs/agents/issue-tracker.md`.

### Triage labels

Triage uses the default five-role vocabulary: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, and `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

This repo uses a single-context domain-doc layout: `CONTEXT.md` at the repo root and ADRs under `docs/adr/`. See `docs/agents/domain.md`.

### Repo-local Go skills

This repo installs Go-specific skills under `.agents/skills/`. Use them when working on Go code in this repository:

- `golang-code-style`: Go code clarity, control flow, variable declarations, line breaking, and comment judgment.
- `golang-error-handling`: idiomatic Go error creation, wrapping, inspection, propagation, and logging.
- `golang-testing`: table-driven tests, test structure, fuzzing, fixtures, coverage, integration tests, and flaky or slow test diagnosis.
