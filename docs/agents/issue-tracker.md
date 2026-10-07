# Issue tracker: Local Markdown

This project uses versioned files in `.scratch/`. No remote tracker is configured.
The `triage` skill is not installed; there is no separate triage-label configuration.

## Feature operations

- One feature per `.scratch/<feature-slug>/` directory.
- Spec: `spec.md`, with `Status: draft` or `Status: ready-for-agent` near the top.
- Implementation tickets: `issues/<NN>-<slug>.md`, starting at `01`, one file per ticket.
- Use plain `Status:`, `Type:`, `Blocked by:`, and `Spec:` metadata lines. Accept
  upstream's bold metadata when reading; normalize it on the next substantive update.
- Implementation statuses: `draft`, `ready-for-agent`, `in-progress`, `blocked`,
  `done`, `wontfix`. Only agreed, sufficiently specified work is `ready-for-agent`.
- "Publish" means create/update the local file. "Fetch" means read the referenced file.
  Resolve numeric references within the named feature; require a path if ambiguous.
- Include observable acceptance criteria, public test boundaries, and blockers in each
  implementation ticket. Blockers are links to tickets in the same feature by default.
  Cross-feature blockers use explicit relative paths.
- A ticket is runnable when it is `ready-for-agent` and all its blockers are `done`.
- Record the implementation baseline SHA before edits. Record commands, results,
  failures, and acceptance evidence under `## Validation`; append discussion under
  `## Comments`. Mark `done` only when scoped acceptance is satisfied.
- Preserve spec → acceptance case → ticket → test/evidence links. Keep unresolved
  choices explicitly open. Do not turn an unconfirmed assumption into a requirement.

## Wayfinding operations

Decision tickets and implementation tickets have different completion states.

- Map: `.scratch/<effort>/map.md` with Destination, Notes, Decisions so far,
  Not yet specified, and Out of scope sections.
- Child: `issues/<NN>-<slug>.md`. `Type:` is `grilling`, `research`, `prototype`, or
  `task`. Store the question and its mode (`HITL` or `AFK`) in the ticket.
- Decision statuses: `open`, `claimed`, `resolved`, `closed`.
- `Blocked by: None`, ticket filenames, or explicit relative paths. A decision
  blocker is satisfied when `resolved`; `closed` is cancellation, not a resolution.
- Frontier: open, unclaimed tickets with all blockers resolved, in numeric order.
- Claim: set `Status: claimed` and `Owner: <session identifier>` before work.
  For an interrupted claim, check that the owning session is inactive before reclaiming.
- Resolve: write `## Answer` with the actual decision/evidence, set `Status: resolved`,
  clear the owner, and add a linked one-line context pointer to Decisions so far.
- Close out-of-scope tickets as `closed`, linking the reason in the map's Out of scope.
- A human decision resolves through the user's answers. Research evidence can be gathered
  by the agent. A prototype is authorized only when the user permits that code work.
- Local tracker artifacts replace upstream references to remote issues, research branch
  pushes, and gists. Research evidence stays in the ticket or a linked local file.
