# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- `CONTEXT.md` at the repo root
- `docs/adr/` when ADRs exist for the area being changed

If any of these files do not exist, proceed silently. Do not stop work just because ADRs are absent.

## File structure

Single-context repo:

```text
/
├── CONTEXT.md
├── docs/
│   └── adr/
└── ingress-service/
```

This repo is currently single-context. There is no `CONTEXT-MAP.md`.

## Use the glossary's vocabulary

When naming domain concepts in issues, plans, tests, refactor proposals, or debugging hypotheses, use the terms from `CONTEXT.md` instead of drifting to synonyms.

## Additional repo guidance

- `docs/current-state.md` is a project-specific handoff file. Read it early for current progress, but do not treat it as the domain glossary.
- `README.md` is the architecture and startup overview, not the source of canonical terminology.
- Do not put short-lived task state or bug lists into `CONTEXT.md`.

## ADR posture

- `docs/adr/` is reserved for architectural decisions that are hard to reverse and surprising without context.
- This repo does not yet have formal ADRs, but the path is reserved and should be used if those decisions begin to accumulate.
