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
| Behavior Concerns | `behavior` (system tag) |
| Medical Concerns | `medical` (system tag or create if missing) |

### Why This Works

1. **Reduces Cognitive Load:** Volunteers don't need to remember to tag
2. **Improves Searchability:** All behavior/medical notes are properly categorized
3. **Backwards Compatible:** Existing tag filters continue to work
4. **Admin Visibility:** Behavior/medical concerns surface in filtered views

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

## 🔒 Security Considerations

1. **Input Sanitization:** All text fields sanitized before storage
2. **XSS Prevention:** Markdown rendering with safe parser
3. **CSRF Protection:** Existing token implementation covers new form

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
