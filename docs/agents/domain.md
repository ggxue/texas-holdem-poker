# Domain docs

Single-context repository. Domain terminology and product choices are not settled yet.

## Read when relevant

- Root `GLOSSARY.md`, when it exists, for the domain words used by the task.
- ADRs under `docs/adr/` that affect the area being discussed or changed.
- The originating spec and ticket for the behavior being implemented or reviewed.

Missing glossary or ADR files are normal during bootstrap. Create them lazily when
there is real resolved content; do not seed a guessed poker glossary or architecture.

## Write in the right place

- `GLOSSARY.md`: agreed vocabulary only, using the `domain-modeling` skill's
  `GLOSSARY-FORMAT.md`. Update terms as they resolve in conversation.
- `docs/adr/NNNN-<slug>.md`: use `ADR-FORMAT.md` only for a choice that is hard to
  reverse, surprising without context, and involves a real trade-off. All three apply.
- `.scratch/<feature>/spec.md`: agreed behavior, interfaces, failure cases, acceptance
  examples, test boundaries, and deliberate exclusions.
- Decision tickets: open questions and their eventual answers. Implementation
  tickets: slices of agreed behavior with independently verifiable acceptance.

Use canonical terms consistently. Surface conflicts with an ADR or spec instead of
silently replacing the existing decision. Routine implementation choices do not need
an ADR. Poker rules require concrete scenarios before they become acceptance cases.
