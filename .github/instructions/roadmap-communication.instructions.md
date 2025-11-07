# Roadmap Communication Guidelines

## 🎯 Purpose

The **ROADMAP.md** is the single source of truth for product direction and is **maintained exclusively by the Product Owner**. This document explains how developers, QA, and other team members should communicate updates and feedback.

---

## ⚠️ Key Rule: Product Owner Owns ROADMAP.md

**DO NOT directly edit ROADMAP.md** unless you are the Product Owner.

Instead, use the communication channels below to report:
- ✅ Feature completions
- ✅ Progress updates
- ✅ Blockers or technical challenges
- ✅ Metrics and learnings
- ✅ New feature suggestions

---

## 📢 How to Communicate Roadmap Updates

### Option 1: GitHub Issue (Recommended for Feature Completion)

**When to use:** Feature is complete or significantly progressed

**Steps:**
1. Create a new GitHub issue
2. Title: `[Roadmap] Update: [Feature Name]`
3. Labels: `roadmap-update`, `documentation`
4. Use the template from `.github/ROADMAP_UPDATE_TEMPLATE.md`
5. Tag the Product Owner: `@[product-owner-username]`

**Example:**
```
Title: [Roadmap] Update: GroupMe Integration Backend Complete
Labels: roadmap-update, documentation
Assignees: @product-owner

[Use template to fill in details]
```

---

### Option 2: Pull Request Body (For Code PRs)

**When to use:** PR completes or advances a roadmap item

**Steps:**
1. In your PR description, add a section:
   ```markdown
   ## 📋 Roadmap Impact
   
   **Roadmap Item:** [Feature name from ROADMAP.md]
   **Status Change:** [e.g., "In Progress" → "Completed"]
   **Acceptance Criteria Met:**
   - [x] Criterion 1
   - [x] Criterion 2
   - [ ] Criterion 3 (remaining)
   
   **Suggested Roadmap Update:**
   Move "GroupMe Integration" to "Shipped" section with following details:
   - Shipped: [Date]
   - PR: #[number]
   - Impact: [metrics/feedback]
   - Learning: [key takeaway]
   ```

2. Tag Product Owner as reviewer
3. Product Owner will update ROADMAP.md in follow-up PR

**Example:** See PR #86 body - it provided all details needed for roadmap update

---

### Option 3: Team Meeting Notes

**When to use:** Regular status updates, discussions, or quick syncs

**Steps:**
1. During sprint planning, standup, or retrospective
2. Report roadmap progress verbally or in meeting notes
3. Product Owner takes notes and updates roadmap afterward

**Format:**
```
Roadmap Updates:
- Feature X: 80% complete, 2 acceptance criteria remaining
- Feature Y: Blocked on [issue], needs discussion
- Feature Z: Completed, metrics show [impact]
```

---

### Option 4: Slack/Chat (For Quick Updates)

**When to use:** Urgent blockers or quick status checks

**Steps:**
1. Message Product Owner or post in `#product` channel
2. Format: `Roadmap Update: [Feature Name] - [Status/Blocker]`
3. Product Owner will follow up with formal issue if needed

**Example:**
```
Roadmap Update: GroupMe Integration - Backend complete! 
Ready for frontend work. PR #85 merged.
```

---

## 📝 What Information to Provide

When reporting roadmap updates, always include:

### 1. **Context**
   - Which roadmap item? (exact name from ROADMAP.md)
   - Which section? (Now, Next, Later)
   - Current status? (In Progress, Completed, Blocked)

### 2. **Progress**
   - What was accomplished?
   - Which acceptance criteria were met?
   - What percentage complete?

### 3. **Metrics** (if applicable)
   - Performance improvements (e.g., "Load time: 3s → 1.5s")
   - Test coverage changes (e.g., "Coverage: 58% → 73%")
   - User feedback or adoption rates

### 4. **Learnings**
   - What went well?
   - What challenges did you face?
   - What would you do differently?
   - What technical insights emerged?

### 5. **Next Steps**
   - What's needed to complete?
   - Any blockers or dependencies?
   - Estimated completion date?

---

## 🚦 Update Workflow

```
Developer completes work
        ↓
Creates GitHub issue or adds to PR body
        ↓
Tags Product Owner
        ↓
Product Owner reviews
        ↓
Product Owner updates ROADMAP.md
        ↓
Team notified of roadmap change
```

---

## 📋 Examples of Good vs Bad Updates

### ✅ Good Update (GitHub Issue)

```markdown
Title: [Roadmap] Update: Test Coverage Improvements Complete

**Roadmap Section:** Now (Q4 2025)
**Feature Name:** Test Coverage Improvements
**Current Status:** Completed

**What Changed:**
PR #86 merged with comprehensive test coverage improvements.

**Acceptance Criteria Met:**
- [x] Backend test coverage: 58.9% → 60%+ (GetGroups: 84%, sendAnnouncementEmails: 76.9%)
- [x] All critical handlers tested (auth, announcement, group)
- [x] Frontend unit tests infrastructure set up
- [x] E2E tests cover critical user journeys (97.8% passing)

**Metrics/Impact:**
- Backend: +1.1% overall, +24% on targeted handlers
- Frontend: 135/138 tests passing (97.8%)
- TypeScript: All `any` types replaced with proper types

**Learnings:**
Error path testing caught edge cases we missed in happy path tests.
Context validation was critical for production stability.

**Suggested Roadmap Update:**
Move "Test Coverage Improvements" to "Shipped" section as v1.0.11
```

### ❌ Bad Update (Too Vague)

```markdown
Title: Tests are done

I finished the tests. Update the roadmap.
```

**Why it's bad:**
- No context about which roadmap item
- No details about what was accomplished
- No metrics or acceptance criteria
- No learnings documented
- Doesn't help Product Owner write good roadmap entry

---

## 🎨 Special Cases

### New Feature Request (Not on Roadmap)

1. Create GitHub issue with `enhancement` label
2. Describe the problem (not the solution)
3. Explain who benefits and why
4. Provide user feedback or data if available
5. Product Owner will evaluate and may add to roadmap

### Blocker or Pivot

1. Create GitHub issue immediately with `roadmap-update`, `blocked` labels
2. Explain the blocker clearly
3. Suggest alternatives or solutions
4. Tag Product Owner for discussion
5. May result in roadmap reprioritization

### Metrics Update (No Code Change)

1. Post in Slack/chat or create lightweight issue
2. Provide metrics: "Feature X adoption: 85% of users"
3. Product Owner updates "Impact" or "Success Metrics" in roadmap

---

## 🔄 Roadmap Update Cadence

**Product Owner commits to:**
- Reviewing roadmap update requests within **2 business days**
- Updating ROADMAP.md after each release (v1.x)
- Major roadmap review at **start of each quarter**
- Ad-hoc updates for completed features or significant changes

**Team commits to:**
- Reporting completions via GitHub issue within **1 day of PR merge**
- Including roadmap impact in PR descriptions
- Raising blockers immediately
- Providing metrics and learnings

---

## 📊 Benefits of This Process

1. **Single Source of Truth:** ROADMAP.md always accurate
2. **Clear Ownership:** Product Owner controls messaging
3. **Developer Autonomy:** Devs focus on building, not documentation
4. **Traceability:** All updates tracked in issues/PRs
5. **Better Quality:** Product Owner adds strategic context
6. **Team Alignment:** Everyone knows how to communicate

---

## 🛠️ Quick Reference

| Scenario | Action | Where |
|----------|--------|-------|
| Feature complete | Create issue with template | GitHub Issues |
| PR advances roadmap item | Add "Roadmap Impact" section | PR body |
| Quick progress update | Message Product Owner | Slack/Chat |
| Blocker | Create issue, tag PO | GitHub Issues |
| New feature idea | Create enhancement issue | GitHub Issues |
| Metrics/feedback | Post in #product | Slack or Issue |

---

## 📞 Questions?

- **"Can I suggest edits to roadmap text?"** Yes! Use the template's "Proposed Text" section
- **"What if it's urgent?"** Message Product Owner directly, follow up with issue
- **"Can I update my own feature status?"** No, but you can request the update
- **"What if PO is unavailable?"** Create issue anyway, it will be reviewed when they return

---

**Remember:** The roadmap is a product management tool, not a developer task list. Your job is to build great features and communicate progress. The Product Owner's job is to maintain the strategic narrative in ROADMAP.md.

---

**Last Updated:** November 6, 2025  
**Maintained By:** Product Owner  
**Feedback:** Open a GitHub issue with `process-improvement` label
