# Issue tracker: Linear

Issues, specifications, and PRDs for this repository live in Linear. Use the official Linear MCP integration for all tracker operations.

## Scope

- Default team: `ENG`
- Default project: `mancing-id`
- Treat Linear as the source of truth. Do not silently fall back to GitHub Issues or local markdown files.
- If the Linear MCP integration is unavailable or unauthorized, stop and ask the user how to proceed.

## Conventions

- Search the `ENG` team and `mancing-id` project for duplicates before creating an issue.
- Create issues in team `ENG` and project `mancing-id` unless the user explicitly selects another destination.
- Use the Linear MCP tools to read, list, create, comment on, label, assign, update, and close issues.
- Preserve existing workspace workflow states and label conventions. Use `docs/agents/triage-labels.md` for triage-role labels.
- Refer to issues by their Linear identifier, such as `ENG-123`, and include the Linear URL when publishing results.
- Record blocking relationships with Linear's native issue relations whenever the integration supports them.

## When a skill says "publish to the issue tracker"

Create a Linear issue in team `ENG` and project `mancing-id`, then return its identifier and URL.

## When a skill says "fetch the relevant ticket"

Fetch the issue by its Linear identifier or URL. If only a title or description is available, search within team `ENG` and project `mancing-id` and ask before choosing between ambiguous matches.

## Wayfinding operations

Used by `/wayfinder`. The map is one Linear issue with linked child issues as tickets.

- **Map**: one issue labelled `wayfinder:map`, containing the Notes, Decisions-so-far, and Fog sections.
- **Child ticket**: a child or related issue labelled `wayfinder:<type>`, where type is `research`, `prototype`, `grilling`, or `task`.
- **Blocking**: use Linear's native blocking and blocked-by relations. A ticket is unblocked when all blockers are completed or cancelled.
- **Frontier**: choose the first open, unblocked, and unassigned child in map order.
- **Claim**: assign the issue to the driving developer before beginning work.
- **Resolve**: comment with the answer, move the issue to the team's completed state, and append a context pointer with the issue link to the map's Decisions-so-far.
