# Skills source and Codex adaptations

Upstream: [mattpocock/skills](https://github.com/mattpocock/skills).
Pinned commit: `f3fc5632f401156837ee3872f14fe33ccf1024ea`.
Installed with OpenAI's `skill-installer` helper using `--ref <SHA>` and
`--dest D:/texas-poker/.agents/skills`. No Node installer is needed to use the skills.
MIT license is preserved in `.agents/skills/LICENSE`.

The exact inventory, dependencies, and upstream/installed file hashes are recorded
in `.agents/skills.lock.json`. Run `scripts/verify.ps1 -Stage Harness` after changes.

## Installed workflow

- Setup: `setup-matt-pocock-skills`.
- Project shaping: `wayfinder`.
- Main flow: `grill-with-docs`, `to-spec`, `to-tickets`, `implement`, `code-review`, `retro`.
- Supporting disciplines: `grilling`, `domain-modeling`, `tdd`, `codebase-design`,
  `research`, `prototype`, `writing-for-agents`.
- Architecture exploration: `improve-codebase-architecture`, explicitly invoked;
  uses the existing `codebase-design`, `grilling`, and `domain-modeling` dependencies.

The last three close dependencies found while reading wayfinder and retro. Triage,
PR generation, and whole-spec parallel implementation are not part of this bootstrap.

## Adaptations

1. Preserve upstream `disable-model-invocation` frontmatter and `agents/openai.yaml`
   with `allow_implicit_invocation: false` for user-invoked skills. Codex uses
   `$skill-name`, and can select via `/skills`. The bundled minimal skill validator
   does not accept the Claude-specific key; validate remaining metadata on temporary
   copies without changing the installed files, then check the original invocation
   policies separately and confirm that Codex loads the upstream manifests.
2. A missing Skill tool means read the named local `SKILL.md` and its references.
   Do not pretend a tool was called. This mapping lives in `AGENTS.md`.
3. The configured local tracker replaces remote publishing and research-branch pushes.
   Keep research evidence in local files. Preserve two independent review axes; use
   subagents when the skill requests them and the environment supports them. If not,
   report that the reviews were sequential, with no claim of independent execution.
4. A spec-approved test boundary is already agreed for `tdd`; ask again only when
   it changes. Setup choices were authorized by the user's request to initialize
   the proposed harness. Future setup reruns preserve existing choices.
5. Match spec and ticket detail to actual scope. The upstream long user-story template
   must not invent behavior. Unresolved product choices stay in planning tickets.

All installed skill content and reference assets retain their upstream form. Updates are
intentional: install into a temporary directory at a new pinned SHA, review the diff,
preserve invocation policies, refresh lock hashes, and run harness verification. Do not
overwrite local changes with an unreviewed update or silently move the upstream pin.
