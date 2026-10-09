# Texas Poker agent guide

## Project and current phase

Go backend and HTML/JS/CSS frontend for a single-room poker game.
Local gameplay tickets 03–08 are implemented; ticket 12 awaits real cloud access and acceptance.
Before game implementation, read [.scratch/online-poker/spec.md](.scratch/online-poker/spec.md)
and the selected ticket in [.scratch/online-poker/ticket-plan.md](.scratch/online-poker/ticket-plan.md).
The current user decision is memory-only chips and reconnect reset to 100; its
supersession details are in [.scratch/online-poker/memory-reset-decision.md](.scratch/online-poker/memory-reset-decision.md).
Do not infer further variants or recovery guarantees.

## Workflow

- For a large effort with unresolved choices, use `$wayfinder` to map decisions.
- For a scoped design discussion, explicitly invoked `$grill-with-docs` loads both
  `grilling` and `domain-modeling`; record resolved terms as they arise.
- Preserve agreed behavior with `$to-spec`, then `$to-tickets` for multi-session work.
  Small agreed changes can skip unnecessary planning artifacts.
- Before implementing game behavior, establish inputs, observable outcomes,
  acceptance cases, and agreed public test boundaries. Implement one ticket at a time.
- Use `tdd` and `codebase-design` at the relevant boundaries. A failing test must
  fail for the intended behavior before implementing it. Do not test private structure.
- Review Standards and Spec separately with `code-review`; identify the exact
  baseline and spec. Also inspect correctness, state transitions, and concurrency.
- Capture reproducible checks and limitations in the ticket before marking it done.
  `$retro` proposes environment improvements from actual session friction.
- Preserve existing user changes. `$implement` includes a local commit: include only
  its scoped changes. Record the baseline commit before starting implementation;
  ensure the reviewed diff includes that work. Follow any user override.
- Every commit subject must use `<type>: <summary>`, including prototype and docs
  commits on every branch (for example `feat:`, `fix:`, `docs:`, `test:`, `prototype:`).
  Before committing, check the prefix; afterward verify `git log -1 --format=%s`.
  Enable the repository's [commit-message hook](docs/agents/harness.md#commit-message-check)
  in each clone so a missing prefix rejects the commit.

## Agent skills

### Issue tracker

Local Markdown under `.scratch/<feature>/`, checked into this repository.
See [tracker operations](docs/agents/issue-tracker.md).

### Domain docs

Single context: root `GLOSSARY.md` and `docs/adr/`, created when actual terms or
qualifying decisions resolve. See [domain rules](docs/agents/domain.md).

### Installation and compatibility

Repo skills live in `.agents/skills/`. Read the selected `SKILL.md` and its actual
dependencies; merely naming a skill does not execute it. If no Skill tool exists,
read these files directly and follow them. Codex uses `$skill-name` / `/skills`.
See [pinned upstream and local adaptations](docs/agents/skills-source.md).

## Verification

Run from the repository root in Windows PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1
```

`bootstrap-go.ps1` installs the checksum-verified, pinned SDK in `.tools/`.
`verify.ps1` checks harness structure, Go formatting, vet, tests, and build. Use
`-Stage Harness` for docs/skills-only changes; `-Stage Go` for the Go checks alone.
Use `-Race` for concurrent behavior when a C compiler is available. Missing tools
and failed checks are failures, never silently skipped. See
[setup and recorded baseline](docs/agents/harness.md).

## Standards and navigation

Read [CODING_STANDARDS.md](CODING_STANDARDS.md) when implementing or reviewing.
Read the glossary if it exists and ADRs relevant to the affected area, not all docs
before every edit. Specs hold behavior; the glossary holds vocabulary.
Put temporary outputs in ignored `artifacts/`; keep decisions and tasks in `.scratch/`.
Discuss product choices with the user; resolve routine implementation details using
the agreed scope. A planning request does not initiate business implementation.
