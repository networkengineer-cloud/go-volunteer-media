# 📋 Roadmap Update Process - Quick Start

## For Developers: Reporting Progress

### When You Complete/Advance a Roadmap Item

1. **Create GitHub Issue** using template:
   ```bash
   # Copy template
   cp .github/ROADMAP_UPDATE_TEMPLATE.md /tmp/update.md
   # Edit with your details
   # Create issue on GitHub with labels: roadmap-update, documentation
   ```

2. **OR Add to Your PR Description:**
   ```markdown
   ## 📋 Roadmap Impact
   
   **Roadmap Item:** GroupMe Integration for Announcements
   **Status Change:** In Progress → Completed
   
   **Acceptance Criteria Met:**
   - [x] Backend models updated
   - [x] GroupMe service created
   - [x] Handler supports multi-channel
   
   **Metrics:**
   - Test coverage: 73% → 78%
   
   **Learning:**
   Bot API integration was straightforward once we understood authentication.
   
   **Suggested Roadmap Update:**
   Move to "Shipped" section as v1.1.0
   ```

3. **Tag Product Owner** in issue or PR

4. **Product Owner Updates ROADMAP.md**

---

## One-Line Rule

**Devs report progress → Product Owner updates ROADMAP.md**

---

## Quick Links

- **Template:** `.github/ROADMAP_UPDATE_TEMPLATE.md`
- **Full Guide:** `.github/instructions/roadmap-communication.instructions.md`
- **Workflow:** `.github/instructions/development-workflow.instructions.md`

---

## Examples

✅ **Good:** "Created issue #123 with template reporting GroupMe integration complete"
❌ **Bad:** "Updated ROADMAP.md directly"

✅ **Good:** "Added 'Roadmap Impact' section to PR #86 showing test coverage increase"
❌ **Bad:** "Forgot to mention roadmap in PR"

---

**Questions?** See full guide at `.github/instructions/roadmap-communication.instructions.md`
