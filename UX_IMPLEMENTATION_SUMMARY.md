# UX Implementation Summary: Returning Animals & Name Collisions

## Problem Statement

The shelter needed to handle two common scenarios with better UX:

1. **Returning Animals**: Animals that return to the shelter after being adopted/fostered (sometimes days, sometimes years later) - possibly with different names
2. **Name Collisions**: Multiple animals with the same name staying in the shelter simultaneously

## Solution Overview

### Returning Animals Tracking

**Key Insight**: Animals may return with different names, so we must track by ID, not by name.

**Implementation**:
- Added `return_count` field to Animal model (database level)
- Auto-increments when animal status changes: `archived` → `available`
- Displays "↩ Returned X×" badge on all animal views
- Works regardless of name changes because tracking is by animal ID

**Example Scenario**:
```
Timeline:
1. Create animal "Max" (ID: 42, return_count: 0)
2. Adopt out → status: archived
3. Animal returns with new name "Buddy"
4. Admin changes: name="Buddy", status=available
5. System increments return_count to 1
6. Badge shows: "↩ Returned 1×"
```

### Name Collision Detection

**Implementation**:
- API endpoint: `GET /groups/:id/animals/check-duplicates?name=X`
- Returns list of animals with matching name (case-insensitive)
- Two display modes:
  1. **Animal Cards**: Client-side calculation, shows "1 of 2" badges
  2. **Animal Form**: API check with detailed warning panel

**Example Scenario**:
```
Shelter has 3 dogs named "Max":
- Max #1 (ID: 10, Labrador, 3 years, available) → Badge: "1 of 3"
- Max #2 (ID: 15, Beagle, 2 years, foster) → Badge: "2 of 3"
- Max #3 (ID: 22, Mixed, 5 years, available) → Badge: "3 of 3"
```

## Visual Design

### Animal Card (GroupPage)

```
┌─────────────────────────────────┐
│  [Animal Photo]                 │
├─────────────────────────────────┤
│ Max                             │
│ [1 of 3] [↩ Returned 2×]       │  ← Badges
│                                 │
│ Labrador                        │
│ 3 years old                     │
│ [Available]                     │
└─────────────────────────────────┘

Badge Colors:
- Duplicate: Yellow/Amber warning
- Returning: Cyan/Teal info
```

### Animal Detail Page

```
Max
Labrador • 3 years old • ID: 42

[Available] [↩ Returned 2×]

⚠️ Bite Quarantine (if applicable)
...
```

### Animal Form - Duplicate Warning

```
┌─────────────────────────────────────────────────┐
│ ⚠️  Name Collision Detected                     │
├─────────────────────────────────────────────────┤
│ There are already 2 other animals named "Max"  │
│ in this group.                                  │
│                                                 │
│ Existing animals:                               │
│ ┌─────────────────────────────────────────────┐ │
│ │ Max (ID: 10)                                │ │
│ │ Labrador • 3 years • available              │ │
│ ├─────────────────────────────────────────────┤ │
│ │ Max (ID: 15)                                │ │
│ │ Beagle • 2 years • foster                   │ │
│ └─────────────────────────────────────────────┘ │
│                                                 │
│ 💡 Tip: Consider adding breed, color, or other │
│    identifying details to help volunteers       │
│    distinguish between animals.                 │
└─────────────────────────────────────────────────┘
```

## Implementation Details

### Backend Changes

**File**: `internal/models/models.go`
```go
type Animal struct {
    // ... existing fields ...
    ReturnCount int `gorm:"default:0" json:"return_count"`
}
```

**File**: `internal/handlers/animal_crud.go`
```go
// In UpdateAnimal handler:
if newStatus == "available" && oldStatus == "archived" {
    animal.ReturnCount++
}
```

**File**: `internal/handlers/animal_helpers.go`
```go
func CheckDuplicateNames(db *gorm.DB) gin.HandlerFunc {
    // Returns DuplicateNameInfo with animals matching name
}
```

**File**: `cmd/api/main.go`
```go
group.GET("/animals/check-duplicates", handlers.CheckDuplicateNames(db))
```

### Frontend Changes

**File**: `frontend/src/api/client.ts`
```typescript
export interface Animal {
    // ... existing fields ...
    return_count: number;
}

export interface DuplicateNameInfo {
    name: string;
    count: number;
    animals: Animal[];
    has_duplicates: boolean;
}

export const animalsApi = {
    checkDuplicates: (groupId: number, name: string) =>
        api.get<DuplicateNameInfo>(...),
    // ...
};
```

**File**: `frontend/src/pages/GroupPage.tsx`
- Calculate duplicates client-side from animals array
- Sort by ID for consistent numbering
- Show badges with proper ARIA labels

**File**: `frontend/src/pages/AnimalDetailPage.tsx`
- Display return badge if `return_count > 0`
- Show animal ID in meta section
- Styled status badge section

**File**: `frontend/src/pages/AnimalForm.tsx`
- Check for duplicates on name field change
- Show warning panel with existing animals
- Debounced API calls
- Exclude current animal when editing

### Styling

**GroupPage.css** - Animal card badges:
```css
.animal-header-badges {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
}

.badge-duplicate {
    background: #fbbf24;
    color: #78350f;
    border: 1px solid #f59e0b;
}

.badge-returning {
    background: #06b6d4;
    color: #083344;
    border: 1px solid #0891b2;
}
```

**Form.css** - Duplicate warning panel:
```css
.duplicate-warning {
    background: #fef3c7;
    border: 2px solid #f59e0b;
    padding: 1.25rem;
}
```

## Accessibility Features

✅ **ARIA Labels**:
- Duplicate badge: `aria-label="1 of 2 animals named Max"`
- Return badge: `aria-label="Returning animal - 2 returns"`

✅ **Semantic HTML**:
- Warning panel uses `role="alert"`
- Proper heading hierarchy
- List markup for animal lists

✅ **Keyboard Navigation**:
- All badges tabbable
- Focus indicators visible
- Logical tab order maintained

✅ **Screen Reader Support**:
- Tooltips provide additional context
- Badge content announced in proper order
- No empty or misleading labels

✅ **Color Contrast**:
- WCAG AA compliant for both light/dark modes
- Does not rely on color alone (uses icons and text)

## Dark Mode Support

All new UI elements have dark mode variants:
- Duplicate badge: Darker amber with lighter text
- Return badge: Darker teal with lighter text
- Warning panel: Dark brown with amber accents

## Testing Strategy

### Playwright E2E Tests

**File**: `frontend/tests/returning-animals-and-duplicates.spec.ts`

Test Coverage:
- ✅ Return badge display on cards and detail pages
- ✅ Duplicate badge calculation and display
- ✅ API integration for duplicate checking
- ✅ Form warning panel rendering
- ✅ Combined scenarios (both badges)
- ✅ Name change on return handling
- ✅ Accessibility compliance
- ✅ Responsive design
- ✅ Performance (no layout shift)

### Manual Testing Scenarios

1. **Create animal with duplicate name**:
   - Enter name "Max"
   - See warning with list of existing "Max" animals
   - Can still save (warning, not error)

2. **Animal returns with same name**:
   - Archive animal "Max"
   - Change status back to available
   - See "↩ Returned 1×" badge

3. **Animal returns with different name**:
   - Archive animal "Max" (ID: 42)
   - Edit: change name to "Buddy", status to available
   - See "↩ Returned 1×" badge (tracked by ID!)

4. **Multiple duplicates**:
   - Create 3 animals named "Max"
   - See "1 of 3", "2 of 3", "3 of 3" badges
   - Each has different breed/age for disambiguation

5. **Both scenarios combined**:
   - Returning animal that also has duplicate name
   - See both badges side by side

## Benefits to Shelter Operations

### For Volunteers
- ✅ Instantly see when animal has returned (behavioral insights)
- ✅ Easily distinguish between animals with same name
- ✅ Clear visual indicators without reading full profiles
- ✅ Proactive warnings prevent confusion

### For Admins
- ✅ Automatic tracking of return history
- ✅ No extra work required (automatic counting)
- ✅ Helpful warnings when creating animals
- ✅ Animal ID provides permanent reference

### For Animals
- ✅ Better tracking even if name changes
- ✅ Return history preserved for behavioral patterns
- ✅ Proper identification reduces mistakes
- ✅ Continuity of care

## Edge Cases Handled

✅ **Case-insensitive matching**: "max", "Max", "MAX" all considered duplicates
✅ **Name changes on return**: Tracked by ID, not name
✅ **Multiple returns**: Count increments each time
✅ **Archived animals**: Included in duplicate check
✅ **Current animal excluded**: When editing, don't count self as duplicate
✅ **Empty names**: No API call until name has 2+ characters
✅ **API errors**: Gracefully handled, doesn't break form
✅ **Long names**: Proper text wrapping and truncation
✅ **Special characters**: Handled correctly in names

## Performance Considerations

✅ **Client-side duplicate calc**: Fast, no API delay on GroupPage
✅ **Debounced API calls**: AnimalForm only calls API after typing pauses
✅ **Efficient queries**: Backend uses indexed fields
✅ **No layout shift**: Badges have fixed space allocation
✅ **Minimal re-renders**: React optimization best practices

## Future Enhancements (Not Implemented)

Potential future improvements:
- [ ] Track full return history with dates
- [ ] Show reason for return (if captured)
- [ ] Analytics on return rates
- [ ] Suggest alternative names when duplicates exist
- [ ] Photo matching for visual identification
- [ ] Integration with adoption records

## Conclusion

This implementation provides a **surgical, minimal-change solution** that significantly improves the UX for two common shelter scenarios:

1. **Returning animals** are clearly identified with automatic tracking
2. **Name collisions** are proactively detected and clearly communicated

The solution respects the constraint that animals may return with different names by tracking returns at the database level using animal IDs rather than names. This ensures accurate return counting regardless of name changes while still providing helpful duplicate name warnings to prevent confusion among current animals.

All changes maintain the existing code style, follow accessibility best practices, and include comprehensive test coverage.
