---
name: roadmap-update
description: Communicate a feature completion or roadmap-impacting change to the Product Owner. Creates a GitHub issue with the roadmap-update label or adds a Roadmap Impact section to the current PR. ROADMAP.md is managed by the Product Owner only — never edit it directly. Use explicitly when a feature is complete and needs roadmap reflection.
disable-model-invocation: true
argument-hint: [feature name or PR number]
---

# Communicating Roadmap Updates

**ROADMAP.md is read-only for developers and agents.** Only the Product Owner updates it. Your job is to communicate completions clearly so the PO can act on them.

There are two ways to communicate a roadmap-impacting change:

---

## Option A — Open a GitHub Issue (for completed features)

Create an issue with the `roadmap-update` label using this template:

**Title:** `[Roadmap Update] <Feature Name>`

**Body:**
```markdown
## Feature Completed

**Feature name:** <exact name as it appears or should appear in ROADMAP.md>
**Completed in PR:** #<PR number>
**Branch merged:** <branch name>

## What was implemented

<2–4 sentence description of what was built and how it fits the roadmap>

## Acceptance criteria met

- [ ] <criterion 1>
- [ ] <criterion 2>

## Notes for Product Owner

<Any caveats, follow-up items, or scope changes the PO should know about>
```

**Labels to apply:** `roadmap-update`

To open the issue:
1. Go to the repository's Issues tab on GitHub
2. Click "New issue"
3. Use the template above
4. Apply the `roadmap-update` label before submitting

---

## Option B — Add a "Roadmap Impact" section to the PR description

When the PR itself is the natural place to record the change, add this section to the PR body:

```markdown
## Roadmap Impact

**Feature:** <Feature name>
**Status:** Complete / Partial / In Progress
**Notes:** <What changed, any scope adjustments, follow-on work needed>
```

---

## What NOT to Do

- ❌ Do not edit `ROADMAP.md` directly — your change will be reverted
- ❌ Do not open a PR that modifies `ROADMAP.md`
- ❌ Do not assume the PO saw a PR merge — explicitly create an issue if the feature is significant

---

## Example Issue

**Title:** `[Roadmap Update] Animal Profile Group Sessions`

**Body:**
```
## Feature Completed

**Feature name:** Animal Profile - Group Sessions
**Completed in PR:** #87
**Branch merged:** feature/group-session-comments

## What was implemented

Added SessionMetadata to animal comments, allowing volunteers to record session
goal, outcome, behavior notes, medical observations, and a 1–5 rating alongside
each comment. Data is stored as JSONB in the animal_comments table.

## Acceptance criteria met

- [x] Volunteers can add session metadata when commenting
- [x] Session notes are visible on the animal profile
- [x] Admins can view and export session history

## Notes for Product Owner

Session notes are not yet filterable by date range — that can be a follow-on task.
```
