# Session Notes Integration Plan: UX Design Research

> **Document Status:** Research & Planning (No Code Changes)  
> **Created:** December 2024  
> **Purpose:** Define how to integrate structured session notes into animal pages while keeping the volunteer experience clean and efficient.

---

## 📋 Executive Summary

This document provides a UX design plan for integrating structured session notes into the go-volunteer-media animal pages. The goal is to provide volunteers with a streamlined way to document their interactions with animals using a guided template, while leveraging the existing tag system for behavior and medical concerns.

### Current Template Structure
```
Target Goal of Session:  
What is the result: 
Behavior Concerns: 
Medical Concerns:  
Success of Session:  
Other Comments:
```

### Key UX Principles

1. **Not all fields should be required.** The form should guide volunteers but not burden them with unnecessary data entry. Quick notes should be just as easy as detailed session reports.

2. **Session notes and photos are separate entities.** There is no direct connection between comments/session notes and photos. Photos exist independently in the animal's photo gallery, while session notes document volunteer interactions and observations. This keeps the data model clean and allows each type of content to be managed independently.

---

## 🔍 Current System Analysis

### Existing Components

| Component | Purpose | Current State |
|-----------|---------|---------------|
| `AnimalComment` | Free-text notes with optional image | ✅ Working well |
| `CommentTag` | Categorize comments (behavior, medical, etc.) | ✅ In place |
| `AnimalTag` | Persistent animal traits (behavior, walker status) | ✅ Separate from session notes |
| Comment Form | Simple textarea + tag selection | ✅ Functional but unguided |

### User Pain Points Identified

1. **Lack of Structure:** Volunteers don't know what information to include
2. **Template is Manual:** Currently requires copy/paste of template
3. **Tag Selection Friction:** Multi-select can be confusing
4. **Behavior/Medical Overlap:** Unclear when to use tags vs. when to write content

---

## 🎯 Proposed Solution: Smart Session Notes Form

### Design Philosophy

1. **Progressive Disclosure:** Show basic form first, expand for detailed sessions
2. **Optional Fields:** All fields are optional except the comment body
3. **Tag-Assisted Entry:** Auto-apply tags based on field usage
4. **Mode Toggle:** Quick note vs. Structured session report

### Standardized Field Names

Throughout this document, the following consistent terminology is used:

| Field Label | Internal Name | Description |
|-------------|---------------|-------------|
| Session Goal | `session_goal` | What the volunteer planned to do |
| Session Outcome | `session_outcome` | What actually happened |
| Behavior Concerns | `behavior_notes` | Behavioral observations |
| Medical Concerns | `medical_notes` | Health/medical observations |
| Session Rating | `session_rating` | 1-4 scale (Poor/Okay/Good/Great) |
| Additional Notes | `other_notes` | Catch-all for other comments |

### Form Modes

#### Mode 1: Quick Note (Default)
For everyday updates, photos, and brief observations.

```
┌─────────────────────────────────────────────────────────┐
│ 💬 Add a comment                                        │
├─────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Write your note...                                  │ │
│ │                                                     │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ 📷 Add Photo    🏷️ Tags ▾    [📋 Session Report]       │
│                                                         │
│                                      [ Post Comment ]   │
└─────────────────────────────────────────────────────────┘
```

#### Mode 2: Session Report (Expanded)
For structured session documentation with guided fields.

```
┌─────────────────────────────────────────────────────────┐
│ 📋 Session Report                    [← Quick Note]     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ 🎯 Session Goal (optional)                              │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ e.g., leash training, socialization, enrichment     │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ 📝 Session Outcome (optional)                           │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ How did it go? What happened?                       │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ ⚠️ Concerns (Behavior / Medical)                        │
│ ┌──────────────────────────┐  ┌──────────────────────┐ │
│ │ 🐕 Behavior              │  │ 🏥 Medical           │ │
│ │ ┌──────────────────────┐ │  │ ┌──────────────────┐ │ │
│ │ │ Any behavior notes   │ │  │ │ Any medical obs  │ │ │
│ │ └──────────────────────┘ │  │ └──────────────────┘ │ │
│ │ Auto-tag: behavior       │  │ Auto-tag: medical    │ │
│ └──────────────────────────┘  └──────────────────────┘ │
│                                                         │
│ ⭐ Session Success                                      │
│ ┌─────────────────────────────────────────────────────┐ │
│ │  😟 Poor    😐 Okay    🙂 Good    😄 Great          │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ 💭 Other Comments (optional)                            │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Anything else worth noting...                       │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ 📷 Add Photo    🏷️ Additional Tags ▾                   │
│                                                         │
│                                [ Post Session Report ]  │
└─────────────────────────────────────────────────────────┘
```

---

## 🏷️ Tag Integration Strategy

### Automatic Tag Application

When volunteers fill in specific fields, tags are automatically applied:

| Field Used | Auto-Applied Tag |
|------------|------------------|
| Behavior Concerns | `behavior` tag |
| Medical Concerns | `medical` tag |

### Tag Identification Strategy

Tags are identified using a combination of name matching and system flags:

```typescript
// Tag lookup logic (frontend)
const findTagByType = (
  tags: CommentTag[], 
  type: 'behavior' | 'medical'
): CommentTag | null => {
  // 1. First, try exact name match (case-insensitive)
  const byName = tags.find(
    t => t.name.toLowerCase() === type.toLowerCase()
  );
  if (byName) return byName;
  
  // 2. Fallback: look for system tags with matching names
  const systemTag = tags.find(
    t => t.is_system && t.name.toLowerCase().includes(type)
  );
  if (systemTag) return systemTag;
  
  // 3. No matching tag found
  return null;
};
```

### Graceful Degradation

When the expected tag doesn't exist for a group:

| Scenario | Behavior |
|----------|----------|
| Tag exists | Auto-select and show indicator: "Auto-tag: 🏷️ behavior" |
| Tag missing | Hide auto-tag indicator; volunteer can manually select tags |
| Tag created later | New session reports will auto-tag; existing comments unchanged |

**Implementation notes:**
- The frontend should cache available tags on form mount
- If a tag is missing, the field still works - just without auto-tagging
- Admin can create missing tags via Tag Management page
- Consider adding a one-time migration to ensure `behavior` and `medical` tags exist for all groups

### Why This Works

1. **Reduces Cognitive Load:** Volunteers don't need to remember to tag
2. **Improves Searchability:** All behavior/medical notes are properly categorized
3. **Backwards Compatible:** Existing tag filters continue to work
4. **Admin Visibility:** Behavior/medical concerns surface in filtered views
5. **Graceful Degradation:** Form works even if tags don't exist yet

### Tag Display Enhancement

Consider showing tags prominently when they indicate concerns:

```
┌─────────────────────────────────────────────────────────┐
│ 🧑 volunteer_jane • 2 hours ago                         │
│                                                         │
│ ⚠️ behavior                  🏥 medical                 │
│                                                         │
│ Session Goal: Leash training practice                   │
│                                                         │
│ Pulled hard when seeing another dog. Had to stop early. │
│                                                         │
│ Behavior: Dog-reactive, lunged at Lab. Needs work.      │
│ Medical: Limping slightly on left back leg.             │
│                                                         │
│ Session: 😐 Okay                                        │
└─────────────────────────────────────────────────────────┘
```

---

## 📱 Responsive Design Considerations

### Mobile-First Approach

On mobile devices (< 640px):

1. **Stacked Layout:** Behavior and Medical fields stack vertically
2. **Collapsible Sections:** Advanced fields behind "Show More"
3. **Large Touch Targets:** 44px minimum for all buttons
4. **Swipe to Submit:** Optional gesture for quick posts

### Mobile Form Complexity Mitigation

The Session Report mode has 6 fields which can be overwhelming on mobile. Three approaches to reduce complexity:

#### Option A: Accordion Sections (Recommended)
Group fields into collapsible sections. Only one section expanded at a time.

```
┌──────────────────────────────┐
│ 📋 Session Report            │
├──────────────────────────────┤
│                              │
│ ▼ 🎯 Goals & Outcome         │
│ ┌──────────────────────────┐ │
│ │ Session Goal             │ │
│ │ ┌──────────────────────┐ │ │
│ │ │ e.g., leash training │ │ │
│ │ └──────────────────────┘ │ │
│ │ Session Outcome          │ │
│ │ ┌──────────────────────┐ │ │
│ │ │ How did it go?       │ │ │
│ │ └──────────────────────┘ │ │
│ └──────────────────────────┘ │
│                              │
│ ▶ ⚠️ Concerns (0)            │
│ ▶ ⭐ Rating & Notes          │
│                              │
│ ┌──────────────────────────┐ │
│ │   Post Session Report    │ │
│ └──────────────────────────┘ │
└──────────────────────────────┘
```

#### Option B: Stepped Wizard
Multi-step form with progress indicator. Best for onboarding new volunteers.

```
┌──────────────────────────────┐
│ 📋 Session Report            │
│ Step 1 of 3: Goals           │
├──────────────────────────────┤
│ ○───○───○                    │
│ 1   2   3                    │
│                              │
│ 🎯 What was your goal?       │
│ ┌──────────────────────────┐ │
│ │ e.g., leash training     │ │
│ └──────────────────────────┘ │
│                              │
│ 📝 What happened?            │
│ ┌──────────────────────────┐ │
│ │ How did the session go?  │ │
│ │                          │ │
│ └──────────────────────────┘ │
│                              │
│ ┌────────┐ ┌───────────────┐ │
│ │ Cancel │ │ Next: Concerns│ │
│ └────────┘ └───────────────┘ │
└──────────────────────────────┘
```

#### Option C: "Add More Details" Progressive Disclosure
Start with just Goal and Outcome, offer to add concerns.

```
┌──────────────────────────────┐
│ 📋 Session Report            │
├──────────────────────────────┤
│                              │
│ 🎯 Session Goal              │
│ ┌──────────────────────────┐ │
│ │ e.g., leash training     │ │
│ └──────────────────────────┘ │
│                              │
│ 📝 What happened?            │
│ ┌──────────────────────────┐ │
│ │ How did it go?           │ │
│ │                          │ │
│ └──────────────────────────┘ │
│                              │
│ ┌──────────────────────────┐ │
│ │ + Add behavior/medical   │ │
│ │   concerns               │ │
│ └──────────────────────────┘ │
│                              │
│ ┌──────────────────────────┐ │
│ │   Post Session Report    │ │
│ └──────────────────────────┘ │
└──────────────────────────────┘
```

### Save Draft Feature

For complex session reports, support saving drafts:

```typescript
interface SessionDraft {
  animal_id: number;
  session_goal?: string;
  session_outcome?: string;
  behavior_notes?: string;
  medical_notes?: string;
  session_rating?: number;
  other_notes?: string;
  saved_at: string;
}

// Save to localStorage
const saveDraft = (draft: SessionDraft) => {
  const key = `session_draft_${draft.animal_id}`;
  localStorage.setItem(key, JSON.stringify(draft));
};

// Restore draft on form mount
const restoreDraft = (animalId: number): SessionDraft | null => {
  const key = `session_draft_${animalId}`;
  const saved = localStorage.getItem(key);
  return saved ? JSON.parse(saved) : null;
};

// Clear draft on successful submit
const clearDraft = (animalId: number) => {
  localStorage.removeItem(`session_draft_${animalId}`);
};
```

**Draft UX:**
- Auto-save every 30 seconds while typing
- Show "Draft saved" indicator
- Prompt to restore on form re-open: "You have an unsaved draft. Restore?"
- Clear draft after successful submission

### Desktop Enhancement

On desktop (> 1024px):

1. **Side-by-Side Fields:** Behavior/Medical in two columns
2. **Keyboard Shortcuts:** Tab through fields, Cmd/Ctrl+Enter to submit
3. **Preview Panel:** Live preview of how note will appear

---

## ♿ Accessibility Requirements

### WCAG 2.1 AA Compliance

| Requirement | Implementation |
|-------------|----------------|
| Keyboard Navigation | All fields accessible via Tab |
| Screen Reader Labels | `aria-label` on all inputs |
| Error Messages | `role="alert"` for validation |
| Color Contrast | 4.5:1 minimum for all text |
| Focus Indicators | Visible focus rings |
| Form Labels | Associated with `for` attribute |

### Field Descriptions

Each field should have:
- Clear label
- Placeholder with example
- Optional helper text
- `aria-describedby` linking label to helper text

---

## 🔄 Data Model Considerations

### Option A: Structured Fields in Comment Content (Recommended)

Store structured data as formatted content in the existing `content` field:

```json
{
  "content": "## Session Goal\nLeash training practice\n\n## Outcome\nPulled hard when seeing another dog...\n\n## Behavior Concerns\nDog-reactive, lunged at Lab.\n\n## Medical Concerns\nLimping slightly on left back leg.\n\n## Session Rating\n3/4 (Good)\n\n## Other Notes\n...",
  "tag_ids": [1, 2]
}
```

**Pros:**
- No database changes required
- Backwards compatible with existing comments
- Searchable via full-text search
- Renders well as Markdown

**Cons:**
- Harder to extract individual fields for analytics
- No strict schema enforcement

### Option B: Structured Metadata Field (Future Enhancement)

Add a JSONB `metadata` field to `AnimalComment`:

```go
type AnimalComment struct {
  // ... existing fields ...
  Metadata *SessionMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`
}

type SessionMetadata struct {
  SessionGoal     string `json:"session_goal,omitempty"`
  SessionOutcome  string `json:"session_outcome,omitempty"`
  BehaviorNotes   string `json:"behavior_notes,omitempty"`
  MedicalNotes    string `json:"medical_notes,omitempty"`
  SessionRating   int    `json:"session_rating,omitempty"` // 1-4
}
```

**Pros:**
- Structured data for analytics
- Easy to query specific fields
- Can enforce schema

**Cons:**
- Database migration required
- More complex API
- Need to handle migration of existing data

### Recommendation

**Start with Option A (formatted content)** for MVP. This allows immediate implementation without database changes. Plan for Option B in a future release if structured analytics become important.

---

## 🎨 Visual Design Specifications

### Session Report Form Fields

| Field | Type | Required | Placeholder |
|-------|------|----------|-------------|
| Session Goal | Input | No | "e.g., leash training, enrichment" |
| Session Outcome | Textarea | No | "What happened during the session?" |
| Behavior Concerns | Textarea | No | "Any behavior observations to note?" |
| Medical Concerns | Textarea | No | "Any health or medical observations?" |
| Session Rating | Rating Selector | No | 4-point scale with emoji |
| Other Comments | Textarea | No | "Anything else worth noting..." |
| Photo | File Upload | No | Existing implementation |

### Color Palette for Concerns

```css
/* Behavior concerns indicator */
.concern-behavior {
  border-left: 4px solid #f59e0b; /* Amber */
  background: rgba(245, 158, 11, 0.05);
}

/* Medical concerns indicator */
.concern-medical {
  border-left: 4px solid #ef4444; /* Red */
  background: rgba(239, 68, 68, 0.05);
}
```

### Session Rating Visual

```
Session Success:
😟 Poor      😐 Okay      🙂 Good      😄 Great
  (1)         (2)          (3)          (4)
```

Use radio button group with emoji labels for visual feedback.

---

## 🛠️ Implementation Phases

### Phase 1: Quick Wins (No Backend Changes)

1. **Update Comment Form UI**
   - Add "Session Report" toggle button
   - Create expandable form section
   - Format output as structured Markdown

2. **Automatic Tag Application**
   - Detect when behavior/medical fields have content
   - Auto-select corresponding tags

3. **Mobile Optimization**
   - Responsive layout for form sections
   - Touch-friendly controls

### Phase 2: Enhanced Display

1. **Parsed Session Display**
   - Detect structured comments
   - Render with highlighted sections
   - Show rating visually

2. **Filter Enhancements**
   - "Sessions with concerns" filter
   - Rating-based filtering

### Phase 3: Analytics & Activity Feed (Optional Future)

1. **Metadata Field Migration**
   - Add JSONB column
   - Migrate existing structured comments

2. **Reporting Dashboard**
   - Session rating trends
   - Concern frequency by animal

3. **Enhanced Activity Feed**
   - Unified view of all activity across animals
   - Filterable by concern type, rating, date
   - Aggregated analytics per animal

---

## 📊 Success Metrics

### Quantitative

- **Form Completion Time:** Target < 60 seconds for quick note
- **Field Usage Rate:** Track which optional fields are used
- **Tag Accuracy:** % of behavior/medical comments with correct tags

### Qualitative

- **User Satisfaction:** Survey volunteers on form usability
- **Data Quality:** Review sample of session reports for completeness

---

## 🧪 Testing Strategy

### Unit Tests (Vitest)

Test the core logic components in isolation:

```typescript
// frontend/src/components/SessionReportForm.test.tsx

describe('SessionReportForm', () => {
  describe('Tag Auto-Application', () => {
    it('should auto-select behavior tag when behavior field has content', () => {
      // Arrange
      const mockTags = [{ id: 1, name: 'behavior' }, { id: 2, name: 'medical' }];
      
      // Act
      render(<SessionReportForm tags={mockTags} />);
      fireEvent.change(screen.getByLabelText('Behavior Concerns'), {
        target: { value: 'Dog reactive' }
      });
      
      // Assert
      expect(getSelectedTagIds()).toContain(1);
    });

    it('should not auto-select tag when behavior tag is missing', () => {
      const mockTags = [{ id: 2, name: 'medical' }]; // No behavior tag
      render(<SessionReportForm tags={mockTags} />);
      
      fireEvent.change(screen.getByLabelText('Behavior Concerns'), {
        target: { value: 'Dog reactive' }
      });
      
      // Should not throw, form still works
      expect(screen.getByLabelText('Behavior Concerns')).toHaveValue('Dog reactive');
    });

    it('should remove auto-tag when field is cleared', () => {
      const mockTags = [{ id: 1, name: 'behavior' }];
      render(<SessionReportForm tags={mockTags} />);
      
      const field = screen.getByLabelText('Behavior Concerns');
      fireEvent.change(field, { target: { value: 'Dog reactive' } });
      fireEvent.change(field, { target: { value: '' } });
      
      expect(getSelectedTagIds()).not.toContain(1);
    });
  });

  describe('Form Mode Toggle', () => {
    it('should switch from Quick Note to Session Report mode', () => {
      render(<SessionReportForm />);
      
      fireEvent.click(screen.getByText('📋 Session Report'));
      
      expect(screen.getByLabelText('Session Goal')).toBeInTheDocument();
      expect(screen.getByLabelText('Session Outcome')).toBeInTheDocument();
    });
  });

  describe('Markdown Output', () => {
    it('should format structured data as Markdown', () => {
      const output = formatSessionAsMarkdown({
        session_goal: 'Leash training',
        session_outcome: 'Went well',
        session_rating: 3
      });
      
      expect(output).toContain('## Session Goal');
      expect(output).toContain('Leash training');
    });
  });

  describe('Draft Save/Restore', () => {
    it('should save draft to localStorage', () => {
      render(<SessionReportForm animalId={123} />);
      
      fireEvent.change(screen.getByLabelText('Session Goal'), {
        target: { value: 'Training' }
      });
      
      // Wait for auto-save
      jest.advanceTimersByTime(30000);
      
      expect(localStorage.getItem('session_draft_123')).toBeTruthy();
    });
  });
});
```

### E2E Tests (Playwright)

Test the complete user journey:

```typescript
// frontend/tests/session-report.spec.ts

import { test, expect } from '@playwright/test';

test.describe('Session Report Form', () => {
  test.beforeEach(async ({ page }) => {
    // Login and navigate to animal detail page
    await page.goto('/login');
    await page.fill('[name="username"]', 'volunteer1');
    await page.fill('[name="password"]', 'password123');
    await page.click('button[type="submit"]');
    await page.goto('/groups/1/animals/1');
  });

  test('should submit a quick note successfully', async ({ page }) => {
    await page.fill('[data-testid="comment-input"]', 'Great walk today!');
    await page.click('button:has-text("Post Comment")');
    
    await expect(page.locator('.comment-card')).toContainText('Great walk today!');
  });

  test('should switch to Session Report mode', async ({ page }) => {
    await page.click('button:has-text("📋 Session Report")');
    
    await expect(page.locator('[data-testid="session-goal"]')).toBeVisible();
    await expect(page.locator('[data-testid="session-outcome"]')).toBeVisible();
  });

  test('should auto-apply behavior tag when filling behavior field', async ({ page }) => {
    await page.click('button:has-text("📋 Session Report")');
    await page.fill('[data-testid="behavior-concerns"]', 'Jumpy with strangers');
    
    // Check that behavior tag is selected
    await expect(page.locator('.tag-selected:has-text("behavior")')).toBeVisible();
  });

  test('should submit session report with all fields', async ({ page }) => {
    await page.click('button:has-text("📋 Session Report")');
    
    await page.fill('[data-testid="session-goal"]', 'Leash training');
    await page.fill('[data-testid="session-outcome"]', 'Made good progress');
    await page.fill('[data-testid="behavior-concerns"]', 'Still reactive to dogs');
    await page.click('[data-testid="rating-3"]'); // Good
    
    await page.click('button:has-text("Post Session Report")');
    
    // Verify comment appears with structured display
    await expect(page.locator('.comment-card')).toContainText('Session Goal');
    await expect(page.locator('.comment-card')).toContainText('Leash training');
    await expect(page.locator('.tag-badge:has-text("behavior")')).toBeVisible();
  });

  test('should save and restore draft on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.click('button:has-text("📋 Session Report")');
    await page.fill('[data-testid="session-goal"]', 'Training session');
    
    // Navigate away and back
    await page.goto('/groups/1');
    await page.goto('/groups/1/animals/1');
    
    // Should prompt to restore draft
    await expect(page.locator('text=Restore draft?')).toBeVisible();
    await page.click('button:has-text("Restore")');
    
    await expect(page.locator('[data-testid="session-goal"]')).toHaveValue('Training session');
  });

  test('should work with accordion sections on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.click('button:has-text("📋 Session Report")');
    
    // Only first section should be expanded
    await expect(page.locator('[data-testid="section-goals"]')).toBeVisible();
    
    // Click concerns section header
    await page.click('[data-testid="accordion-concerns"]');
    
    // Concerns section should expand
    await expect(page.locator('[data-testid="behavior-concerns"]')).toBeVisible();
  });
});

test.describe('Session Report Accessibility', () => {
  test('should be keyboard navigable', async ({ page }) => {
    await page.goto('/groups/1/animals/1');
    
    // Tab through form elements
    await page.keyboard.press('Tab');
    await expect(page.locator('[data-testid="comment-input"]')).toBeFocused();
    
    await page.keyboard.press('Tab');
    await expect(page.locator('button:has-text("📋 Session Report")')).toBeFocused();
  });

  test('should have proper ARIA labels', async ({ page }) => {
    await page.goto('/groups/1/animals/1');
    await page.click('button:has-text("📋 Session Report")');
    
    await expect(page.locator('[aria-label="Session goal"]')).toBeVisible();
    await expect(page.locator('[aria-describedby]')).toBeVisible();
  });

  test('should announce errors to screen readers', async ({ page }) => {
    await page.goto('/groups/1/animals/1');
    await page.click('button:has-text("📋 Session Report")');
    
    // Try to submit empty form (if validation exists)
    await page.click('button:has-text("Post Session Report")');
    
    // Error should have role="alert"
    await expect(page.locator('[role="alert"]')).toBeVisible();
  });
});
```

### Test Coverage Targets

| Area | Target Coverage |
|------|-----------------|
| Tag auto-application logic | 100% |
| Form mode toggle | 100% |
| Markdown formatting | 100% |
| Draft save/restore | 90% |
| Form validation | 100% |
| E2E happy path | Covered |
| E2E edge cases | Covered |
| Mobile responsive behavior | Covered |
| Accessibility | WCAG AA audit |

---

## 🔒 Security Considerations

### Input Validation & Sanitization

1. **Input Sanitization:** All text fields sanitized before storage
2. **XSS Prevention:** Markdown rendering with safe parser (e.g., `marked` with `sanitize: true`)
3. **CSRF Protection:** Existing token implementation covers new form

### Content Length Limits

Prevent abuse and ensure reasonable data sizes:

| Field | Max Length | Rationale |
|-------|------------|-----------|
| Session Goal | 200 chars | Brief objective |
| Session Outcome | 2000 chars | Detailed narrative |
| Behavior Concerns | 1000 chars | Important observations |
| Medical Concerns | 1000 chars | Critical health info |
| Additional Notes | 1000 chars | Supplementary details |
| Total content | 5000 chars | Database efficiency |

```typescript
// Frontend validation
const FIELD_LIMITS = {
  session_goal: 200,
  session_outcome: 2000,
  behavior_notes: 1000,
  medical_notes: 1000,
  other_notes: 1000,
} as const;

// Backend validation (Go)
func validateSessionReport(metadata SessionMetadata) error {
  if len(metadata.SessionGoal) > 200 {
    return errors.New("session goal exceeds 200 character limit")
  }
  // ... additional validations
}
```

### Rate Limiting for Session Reports

Structured reports require more processing; consider separate rate limits:

| Operation | Rate Limit | Window |
|-----------|------------|--------|
| Quick Note | 30/minute | Per user |
| Session Report | 10/minute | Per user |
| With Medical Concerns | 10/minute | Per user, logged |

```go
// Rate limiter configuration example
sessionReportLimiter := middleware.NewRateLimiter(
  10,           // requests
  time.Minute,  // window
  "session_report",
)
```

### Medical Data Sensitivity

Medical concerns may contain sensitive health information. Consider:

1. **Access Control:** Only group members can view medical notes
2. **Audit Logging:** Log access to comments tagged with `medical`
3. **Data Retention:** Define retention policy for medical observations
4. **Export Restrictions:** Medical data excluded from bulk exports (or require admin approval)
5. **Disclaimer:** Add note that this is not a substitute for professional veterinary records

```typescript
// Display disclaimer when medical field is used
{showMedicalField && (
  <div className="medical-disclaimer" role="note">
    <strong>Note:</strong> Medical observations recorded here are for volunteer 
    awareness only. Please report urgent health concerns to staff immediately.
  </div>
)}
```

### Validation Flow

```
┌─────────────────┐
│ User Input      │
└────────┬────────┘
         ▼
┌─────────────────┐
│ Frontend        │
│ - Length check  │
│ - XSS sanitize  │
│ - Rate limit    │
└────────┬────────┘
         ▼
┌─────────────────┐
│ Backend API     │
│ - Auth check    │
│ - Re-sanitize   │
│ - Length verify │
│ - Rate limit    │
└────────┬────────┘
         ▼
┌─────────────────┐
│ Database        │
│ - Stored safely │
└─────────────────┘
```

---

## 📎 Appendix: Wireframe Mockups

### A1: Quick Note Mode (Mobile)

```
┌──────────────────────────────┐
│ ← Back to [Animal Name]      │
├──────────────────────────────┤
│                              │
│ 💬 Add a comment             │
│                              │
│ ┌──────────────────────────┐ │
│ │ Write your note...       │ │
│ │                          │ │
│ │                          │ │
│ │                          │ │
│ └──────────────────────────┘ │
│                              │
│ ┌────────┐ ┌────────────┐   │
│ │📷 Photo│ │🏷️ Tags    ▾│   │
│ └────────┘ └────────────┘   │
│                              │
│ ┌──────────────────────────┐ │
│ │ 📋 Switch to Session     │ │
│ │    Report Mode           │ │
│ └──────────────────────────┘ │
│                              │
│ ┌──────────────────────────┐ │
│ │     Post Comment         │ │
│ └──────────────────────────┘ │
│                              │
└──────────────────────────────┘
```

### A2: Session Report Mode (Mobile)

```
┌──────────────────────────────┐
│ ← Back to Quick Note         │
├──────────────────────────────┤
│                              │
│ 📋 Session Report            │
│                              │
│ 🎯 Session Goal              │
│ ┌──────────────────────────┐ │
│ │ e.g., leash training     │ │
│ └──────────────────────────┘ │
│                              │
│ 📝 What Happened?            │
│ ┌──────────────────────────┐ │
│ │ How did it go?           │ │
│ │                          │ │
│ └──────────────────────────┘ │
│                              │
│ ⚠️ Behavior Concerns         │
│ ┌──────────────────────────┐ │
│ │ Any behavior notes?      │ │
│ └──────────────────────────┘ │
│ Auto-tag: 🏷️ behavior        │
│                              │
│ 🏥 Medical Concerns          │
│ ┌──────────────────────────┐ │
│ │ Any medical observations?│ │
│ └──────────────────────────┘ │
│ Auto-tag: 🏷️ medical         │
│                              │
│ ⭐ Session Rating            │
│ ┌──────────────────────────┐ │
│ │ 😟  😐  🙂  😄            │ │
│ └──────────────────────────┘ │
│                              │
│ 💭 Other Comments            │
│ ┌──────────────────────────┐ │
│ │ Anything else...         │ │
│ │                          │ │
│ └──────────────────────────┘ │
│                              │
│ ┌────────┐ ┌────────────┐   │
│ │📷 Photo│ │🏷️ More Tags▾│   │
│ └────────┘ └────────────┘   │
│                              │
│ ┌──────────────────────────┐ │
│ │   Post Session Report    │ │
│ └──────────────────────────┘ │
│                              │
└──────────────────────────────┘
```

### A3: Desktop View (Session Report Mode)

```
┌────────────────────────────────────────────────────────────────────────────────┐
│ 📋 Session Report                                      [← Switch to Quick Note]│
├────────────────────────────────────────────────────────────────────────────────┤
│                                                                                │
│ ┌────────────────────────────────────────────────────────────────────────────┐ │
│ │ 🎯 Session Goal (optional)                                                  │ │
│ │ ┌────────────────────────────────────────────────────────────────────────┐ │ │
│ │ │ e.g., leash training, socialization, enrichment                        │ │ │
│ │ └────────────────────────────────────────────────────────────────────────┘ │ │
│ └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                │
│ ┌────────────────────────────────────────────────────────────────────────────┐ │
│ │ 📝 Session Outcome (optional)                                               │ │
│ │ ┌────────────────────────────────────────────────────────────────────────┐ │ │
│ │ │ How did the session go? What happened?                                 │ │ │
│ │ │                                                                        │ │ │
│ │ └────────────────────────────────────────────────────────────────────────┘ │ │
│ └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                │
│ ┌────────────────────────────────┐  ┌────────────────────────────────────┐   │
│ │ ⚠️ Behavior Concerns            │  │ 🏥 Medical Concerns                 │   │
│ │ ┌────────────────────────────┐ │  │ ┌────────────────────────────────┐ │   │
│ │ │ Any behavior observations? │ │  │ │ Any health observations?       │ │   │
│ │ │                            │ │  │ │                                │ │   │
│ │ └────────────────────────────┘ │  │ └────────────────────────────────┘ │   │
│ │ 🏷️ Auto-tag: behavior          │  │ 🏷️ Auto-tag: medical               │   │
│ └────────────────────────────────┘  └────────────────────────────────────┘   │
│                                                                                │
│ ┌────────────────────────────────────────────────────────────────────────────┐ │
│ │ ⭐ Session Success                                                          │ │
│ │    ○ 😟 Poor     ○ 😐 Okay     ○ 🙂 Good     ○ 😄 Great                    │ │
│ └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                │
│ ┌────────────────────────────────────────────────────────────────────────────┐ │
│ │ 💭 Other Comments (optional)                                                │ │
│ │ ┌────────────────────────────────────────────────────────────────────────┐ │ │
│ │ │ Anything else worth noting...                                          │ │ │
│ │ │                                                                        │ │ │
│ │ └────────────────────────────────────────────────────────────────────────┘ │ │
│ └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                │
│ ┌─────────────────┐  ┌───────────────────────┐                                │
│ │ 📷 Add Photo    │  │ 🏷️ Additional Tags ▾  │                                │
│ └─────────────────┘  └───────────────────────┘                                │
│                                                                                │
│                                                       [ Post Session Report ]  │
└────────────────────────────────────────────────────────────────────────────────┘
```

---

## ✅ Next Steps

1. **Review this plan** with stakeholders
2. **Prioritize phases** based on timeline and resources
3. **Create GitHub issues** for Phase 1 tasks
4. **Design review** for wireframes
5. **Implement Phase 1** frontend changes

---

## 📚 References

- [Nielsen Norman Group: Form Design Best Practices](https://www.nngroup.com/articles/web-form-design/)
- [WCAG 2.1 Quick Reference](https://www.w3.org/WAI/WCAG21/quickref/)
- Existing Comment Form Implementation: `frontend/src/pages/AnimalDetailPage.tsx`
- Tag Management System: `frontend/src/pages/AdminAnimalTagsPage.tsx`

---

## 📰 Appendix: Activity Feed Design (Phase 3)

> This section describes the enhanced activity feed assuming Phase 3 (Analytics & Activity Feed) is complete.

### Design Principles for Activity Feed

1. **Session notes and photos remain separate** - Photos uploaded to the gallery are not linked to session notes
2. **Unified timeline** - All volunteer activity (session notes, quick comments) shown chronologically
3. **Structured data parsing** - Session reports display with parsed sections and ratings
4. **Filtering by concern type** - Quickly surface behavior or medical concerns across all animals

### A4: Group Activity Feed (Desktop - Phase 3 Complete)

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│ 📰 Dogs Group Activity Feed                                              [⚙️ Filters]│
├──────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                      │
│ ┌─── Filter Bar ──────────────────────────────────────────────────────────────────┐ │
│ │ [All Activity ▾]  [All Animals ▾]  [All Tags ▾]  [All Ratings ▾]  [Date Range]  │ │
│ │                                                                                  │ │
│ │ Quick Filters: [⚠️ Behavior] [🏥 Medical] [😟 Poor Sessions] [📋 Session Reports]│ │
│ └──────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
│ ┌─── Session Note ─────────────────────────────────────────────────────────────────┐ │
│ │ 🐕 Max • by @volunteer_jane • 2 hours ago                                        │ │
│ │ ────────────────────────────────────────────────────────────────────             │ │
│ │ 🏷️ behavior  🏷️ medical                                            Session: 😐  │ │
│ │                                                                                  │ │
│ │ ┌─ Goal ──────────────────────────────────────────────────────────────────────┐ │ │
│ │ │ Leash training practice                                                     │ │ │
│ │ └─────────────────────────────────────────────────────────────────────────────┘ │ │
│ │                                                                                  │ │
│ │ ┌─ Outcome ───────────────────────────────────────────────────────────────────┐ │ │
│ │ │ Pulled hard when seeing another dog. Had to stop early due to reactivity.  │ │ │
│ │ └─────────────────────────────────────────────────────────────────────────────┘ │ │
│ │                                                                                  │ │
│ │ ┌─ ⚠️ Behavior ─────────────────┐  ┌─ 🏥 Medical ────────────────────────────┐ │ │
│ │ │ Dog-reactive, lunged at a    │  │ Limping slightly on left back leg.      │ │ │
│ │ │ Lab across the street.       │  │ Noticed when walking back to kennel.    │ │ │
│ │ └──────────────────────────────┘  └──────────────────────────────────────────┘ │ │
│ │                                                                                  │ │
│ │                                                    [View Max's Profile →]        │ │
│ └──────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
│ ┌─── Quick Note ───────────────────────────────────────────────────────────────────┐ │
│ │ 🐕 Bella • by @volunteer_mike • 3 hours ago                                      │ │
│ │ ────────────────────────────────────────────────────────────────────             │ │
│ │                                                                                  │ │
│ │ Great walk today! Bella was calm and enjoyed sniffing around the park. No       │ │
│ │ issues with other dogs or people. Very treat motivated.                          │ │
│ │                                                                                  │ │
│ │                                                   [View Bella's Profile →]       │ │
│ └──────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
│ ┌─── Session Note ─────────────────────────────────────────────────────────────────┐ │
│ │ 🐕 Rocky • by @volunteer_sam • 5 hours ago                                       │ │
│ │ ────────────────────────────────────────────────────────────────────             │ │
│ │ 🏷️ behavior                                                        Session: 😄  │ │
│ │                                                                                  │ │
│ │ ┌─ Goal ──────────────────────────────────────────────────────────────────────┐ │ │
│ │ │ Enrichment and play session                                                 │ │ │
│ │ └─────────────────────────────────────────────────────────────────────────────┘ │ │
│ │                                                                                  │ │
│ │ ┌─ Outcome ───────────────────────────────────────────────────────────────────┐ │ │
│ │ │ Loved the puzzle toy! Figured it out in under 5 minutes. Very engaged.     │ │ │
│ │ └─────────────────────────────────────────────────────────────────────────────┘ │ │
│ │                                                                                  │ │
│ │ ┌─ ⚠️ Behavior ─────────────────────────────────────────────────────────────┐   │ │
│ │ │ Resource guards toys slightly. Growled once when I reached for the ball.  │   │ │
│ │ └───────────────────────────────────────────────────────────────────────────┘   │ │
│ │                                                                                  │ │
│ │                                                   [View Rocky's Profile →]       │ │
│ └──────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
│ ┌──────────────────────────────────────────────────────────────────────────────────┐ │
│ │                            [ Load More Activity ]                                │ │
│ └──────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### A5: Activity Feed with Medical/Behavior Filter Active (Desktop)

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│ 📰 Dogs Group Activity Feed                                              [⚙️ Filters]│
├──────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                      │
│ ┌─── Filter Bar ──────────────────────────────────────────────────────────────────┐ │
│ │ Showing: [🏥 Medical Concerns] across all animals                    [✕ Clear]  │ │
│ │                                                                                  │ │
│ │ 📊 Summary: 3 animals with medical notes in the last 7 days                     │ │
│ └──────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
│ ┌─── Session Note ─────────────────────────────────────────────────────────────────┐ │
│ │ 🐕 Max • by @volunteer_jane • 2 hours ago                                        │ │
│ │ ────────────────────────────────────────────────────────────────────             │ │
│ │ 🏥 MEDICAL CONCERN                                                  Session: 😐  │ │
│ │                                                                                  │ │
│ │ ┌─ 🏥 Medical ───────────────────────────────────────────────────────────────┐  │ │
│ │ │ Limping slightly on left back leg. Noticed when walking back to kennel.   │  │ │
│ │ └────────────────────────────────────────────────────────────────────────────┘  │ │
│ │                                                                                  │ │
│ │                                                    [View Full Note] [View Max →] │ │
│ └──────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
│ ┌─── Session Note ─────────────────────────────────────────────────────────────────┐ │
│ │ 🐕 Luna • by @volunteer_alex • Yesterday                                         │ │
│ │ ────────────────────────────────────────────────────────────────────             │ │
│ │ 🏥 MEDICAL CONCERN                                                  Session: 🙂  │ │
│ │                                                                                  │ │
│ │ ┌─ 🏥 Medical ───────────────────────────────────────────────────────────────┐  │ │
│ │ │ Hot spot developing on right shoulder. Looks irritated and she's licking  │  │ │
│ │ │ it frequently. Should be checked by vet.                                   │  │ │
│ │ └────────────────────────────────────────────────────────────────────────────┘  │ │
│ │                                                                                  │ │
│ │                                                   [View Full Note] [View Luna →] │ │
│ └──────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### A6: Activity Feed (Mobile - Phase 3 Complete)

```
┌──────────────────────────────┐
│ 📰 Activity Feed             │
│ Dogs Group                   │
├──────────────────────────────┤
│ ┌──────────────────────────┐ │
│ │ 🔍 Filter ▾              │ │
│ │ [Behavior][Medical][All] │ │
│ └──────────────────────────┘ │
│                              │
│ ┌──────────────────────────┐ │
│ │ 🐕 Max                   │ │
│ │ @volunteer_jane • 2h ago │ │
│ │ ────────────────────────│ │
│ │ 🏷️ behavior 🏷️ medical   │ │
│ │ Session Rating: 😐 Okay  │ │
│ │                          │ │
│ │ Goal: Leash training     │ │
│ │                          │ │
│ │ ⚠️ Behavior: Dog-reactive│ │
│ │ lunged at a Lab...       │ │
│ │                          │ │
│ │ 🏥 Medical: Limping on   │ │
│ │ left back leg...         │ │
│ │                          │ │
│ │ [View More →]            │ │
│ └──────────────────────────┘ │
│                              │
│ ┌──────────────────────────┐ │
│ │ 🐕 Bella                 │ │
│ │ @volunteer_mike • 3h ago │ │
│ │ ────────────────────────│ │
│ │                          │ │
│ │ Great walk today! Bella  │ │
│ │ was calm and enjoyed     │ │
│ │ sniffing around...       │ │
│ │                          │ │
│ │ [View More →]            │ │
│ └──────────────────────────┘ │
│                              │
│ ┌──────────────────────────┐ │
│ │ 🐕 Rocky                 │ │
│ │ @volunteer_sam • 5h ago  │ │
│ │ ────────────────────────│ │
│ │ 🏷️ behavior              │ │
│ │ Session Rating: 😄 Great │ │
│ │                          │ │
│ │ Goal: Enrichment         │ │
│ │                          │ │
│ │ ⚠️ Behavior: Resource    │ │
│ │ guards toys slightly...  │ │
│ │                          │ │
│ │ [View More →]            │ │
│ └──────────────────────────┘ │
│                              │
│ ┌──────────────────────────┐ │
│ │    [ Load More ]         │ │
│ └──────────────────────────┘ │
└──────────────────────────────┘
```

### Activity Feed Data Model (Phase 3)

With the JSONB metadata field in place, the activity feed can query structured session data efficiently:

```sql
-- Example query: Find all sessions with medical concerns, sorted by recency
SELECT ac.*, a.name as animal_name, u.username
FROM animal_comments ac
JOIN animals a ON ac.animal_id = a.id
JOIN users u ON ac.user_id = u.id
WHERE ac.metadata->>'medical_notes' IS NOT NULL
  AND ac.metadata->>'medical_notes' != ''
ORDER BY ac.created_at DESC
LIMIT 20;

-- Example query: Average session ratings by animal
SELECT a.name, AVG((ac.metadata->>'session_rating')::int) as avg_rating
FROM animal_comments ac
JOIN animals a ON ac.animal_id = a.id
WHERE ac.metadata->>'session_rating' IS NOT NULL
GROUP BY a.id, a.name
ORDER BY avg_rating DESC;
```

### Activity Feed API Endpoint (Phase 3)

```typescript
interface ActivityFeedResponse {
  items: SessionActivity[];
  total: number;
  hasMore: boolean;
  filters: ActiveFilters;
  summary?: {
    behavior_concerns_count: number;
    medical_concerns_count: number;
    poor_sessions_count: number;
  };
}

interface SessionActivity {
  id: number;
  type: 'session_report' | 'quick_note';
  created_at: string;
  animal: {
    id: number;
    name: string;
    image_url: string;
  };
  user: {
    id: number;
    username: string;
  };
  content: string;
  tags: string[];
  metadata?: {
    session_goal?: string;
    session_outcome?: string;
    behavior_notes?: string;
    medical_notes?: string;
    session_rating?: number; // 1-4
  };
}
```

### Key Activity Feed Features (Phase 3)

| Feature | Description |
|---------|-------------|
| **Unified Timeline** | All session notes and quick comments in one scrollable feed |
| **Smart Filtering** | Filter by concern type, rating, animal, date range |
| **Concern Highlighting** | Medical and behavior concerns visually emphasized |
| **Summary Stats** | Quick overview of concerns and trends at the top |
| **Parsed Display** | Structured session reports shown with labeled sections |
| **Rating Indicators** | Emoji ratings visible at a glance |
| **Cross-Animal View** | See activity across all animals in a group |
| **Link to Animal** | Quick navigation to animal detail page |

> **Note:** Photos remain in the separate Photo Gallery and are not displayed in the activity feed. This maintains a clear separation between session documentation and photo management.
