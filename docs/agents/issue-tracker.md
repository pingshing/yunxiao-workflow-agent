# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub issues. Use the `gh` CLI for all operations.

Remote:

- `ssh://git@github.com/pingshing/yunxiao-workflow-agent.git`

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`
- **Read an issue**: `gh issue view <number> --comments`
- **List issues**: `gh issue list --state open`
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `gh issue edit <number> --remove-label "..."`
- **Close an issue**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v`. Running `gh` inside this clone should target the correct repository.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.

## Project posture

- Larger changes should usually become a PRD issue first, then be split into implementation issues.
- If the user explicitly wants to keep something local and exploratory, it can stay in conversation or local markdown temporarily, but GitHub Issues remain the default system of record.
