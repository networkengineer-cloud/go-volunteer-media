# Bulk Edit Animals - Visual Guide

## UI Overview

The Bulk Edit Animals page provides a comprehensive interface for managing multiple animals efficiently. This guide shows the key UI components and workflows.

## Page Layout

### 1. Header Section
```
┌─────────────────────────────────────────────────────────────┐
│  Bulk Edit Animals          [Export CSV] [Import CSV]       │
└─────────────────────────────────────────────────────────────┘
```

**Components:**
- Page title: "Bulk Edit Animals"
- Export CSV button: Download animal data
- Import CSV button: Upload animal data from CSV file

### 2. Filter Section
```
┌─────────────────────────────────────────────────────────────┐
│  Filter by Status: [All ▼]                                  │
│  Filter by Group:  [All Groups ▼]                           │
│  Search by Name:   [Search... ]                             │
└─────────────────────────────────────────────────────────────┘
```

**Filters:**
- **Status Filter**: All, Available, Adopted, Fostered
- **Group Filter**: All Groups, Dogs, Cats, Mod Squad, etc.
- **Name Search**: Free-text search by animal name

### 3. Bulk Actions Bar
```
┌─────────────────────────────────────────────────────────────┐
│  3 animal(s) selected                                       │
│  [Select Action ▼] [Select Group ▼] [Apply]                │
└─────────────────────────────────────────────────────────────┘
```

**Actions:**
- **Move to Group**: Select target group, then click Apply
- **Change Status**: Select new status (available/adopted/fostered), then click Apply

### 4. Animals Table
```
┌──┬────┬─────────┬─────────┬────────────────┬─────┬───────────┬────────┐
│☑ │ ID │ Name    │ Species │ Breed          │ Age │ Status    │ Group  │
├──┼────┼─────────┼─────────┼────────────────┼─────┼───────────┼────────┤
│☑ │ 1  │ Max     │ Dog     │ Labrador       │ 3   │ available │ Dogs   │
│☑ │ 2  │ Bella   │ Dog     │ German Shep... │ 5   │ available │ Dogs   │
│☑ │ 3  │ Charlie │ Dog     │ Beagle         │ 2   │ available │ Dogs   │
│☐ │ 4  │ Luna    │ Cat     │ Maine Coon     │ 1   │ available │ Cats   │
│☐ │ 5  │ Shadow  │ Cat     │ Persian        │ 4   │ fostered  │ Cats   │
└──┴────┴─────────┴─────────┴────────────────┴─────┴───────────┴────────┘
```

**Features:**
- Checkboxes for individual selection
- Select all checkbox in header
- Sortable columns (ID, Name, etc.)
- Status badges with color coding
- Responsive design for mobile

## Workflows

### Workflow 1: Move Multiple Animals to a Different Group

**Scenario**: Move 3 dogs to the "Mod Squad" group

```
1. Filter (optional)
   ┌─────────────────────────────────────┐
   │ Filter by Group: [Dogs ▼]          │
   └─────────────────────────────────────┘

2. Select Animals
   ┌──┬────┬─────────┬─────────┐
   │☑ │ 1  │ Max     │ Dog     │  ← Selected
   │☑ │ 2  │ Bella   │ Dog     │  ← Selected
   │☑ │ 3  │ Charlie │ Dog     │  ← Selected
   └──┴────┴─────────┴─────────┘

3. Choose Action
   ┌────────────────────────────────────────┐
   │ 3 animal(s) selected                   │
   │ [Move to Group ▼] [Mod Squad ▼] [Apply]│
   └────────────────────────────────────────┘

4. Result
   ✓ Successfully updated 3 animals
   Animals now appear in Mod Squad group
```

### Workflow 2: Change Animal Status in Bulk

**Scenario**: Mark 2 animals as adopted

```
1. Select Animals
   ┌──┬────┬─────────┐
   │☑ │ 1  │ Max     │  ← Selected
   │☑ │ 2  │ Bella   │  ← Selected
   └──┴────┴─────────┘

2. Choose Action
   ┌────────────────────────────────────────┐
   │ 2 animal(s) selected                   │
   │ [Change Status ▼] [Adopted ▼] [Apply] │
   └────────────────────────────────────────┘

3. Result
   ✓ Successfully updated 2 animals
   Status badges change to "adopted"
```

### Workflow 3: Import Animals from CSV

**Scenario**: Import 5 new animals from a CSV file

```
1. Click Import CSV Button
   ┌────────────────────────────────┐
   │  [Import CSV] ← Click here     │
   └────────────────────────────────┘

2. Import Modal Opens
   ┌─────────────────────────────────────────────┐
   │ Import Animals from CSV              [×]    │
   ├─────────────────────────────────────────────┤
   │ Upload a CSV file with the following        │
   │ columns:                                    │
   │ group_id, name, species, breed, age,        │
   │ description, status, image_url              │
   │                                             │
   │ Note: Only group_id and name are required.  │
   │                                             │
   │ [Choose File] sample_animals.csv            │
   │                                             │
   ├─────────────────────────────────────────────┤
   │          [Cancel]    [Import]               │
   └─────────────────────────────────────────────┘

3. Result
   ┌─────────────────────────────────────────────┐
   │ ✓ Successfully imported 5 animals           │
   │                                             │
   │ Warnings:                                   │
   │ • Line 3: Invalid age                       │
   └─────────────────────────────────────────────┘

4. Table Updated
   New animals appear in the table
```

### Workflow 4: Export Animals to CSV

**Scenario**: Export all animals in the "Dogs" group

```
1. Filter by Group (optional)
   ┌─────────────────────────────────┐
   │ Filter by Group: [Dogs ▼]      │
   └─────────────────────────────────┘

2. Click Export CSV
   ┌────────────────────────────────┐
   │  [Export CSV] ← Click here     │
   └────────────────────────────────┘

3. Download Starts
   📥 animals.csv downloaded
   
4. CSV Contents
   group_id,name,species,breed,age,description,status,image_url
   1,Max,Dog,Labrador,3,Friendly dog,available,
   1,Bella,Dog,German Shepherd,5,Great with kids,available,
   1,Charlie,Dog,Beagle,2,Energetic,available,
```

## UI States

### Empty State
```
┌─────────────────────────────────────────────────────┐
│                                                     │
│              No animals found                       │
│                                                     │
│  Try adjusting your filters or import animals      │
│  from a CSV file.                                   │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### Loading State
```
┌─────────────────────────────────────────────────────┐
│                   Loading...                        │
└─────────────────────────────────────────────────────┘
```

### Success Message
```
┌─────────────────────────────────────────────────────┐
│ ✓ Successfully updated 3 animals                    │
└─────────────────────────────────────────────────────┘
```

### Error Message
```
┌─────────────────────────────────────────────────────┐
│ ✗ Failed to update animals: Invalid group ID       │
└─────────────────────────────────────────────────────┘
```

## Responsive Design

### Desktop View (1200px+)
- Full table with all columns visible
- Filters displayed horizontally
- Bulk actions bar spans full width

### Tablet View (768px - 1199px)
- Table scrolls horizontally if needed
- Filters stack vertically
- Bulk actions remain functional

### Mobile View (<768px)
- Filters stack vertically
- Table scrolls horizontally
- Bulk actions stack vertically
- Simplified column display

## Color Coding

### Status Badges
- **Available**: Green background (#d1fae5), dark green text (#065f46)
- **Adopted**: Blue background (#dbeafe), dark blue text (#1e40af)
- **Fostered**: Yellow background (#fef3c7), dark yellow text (#92400e)

### Action Buttons
- **Primary (Import CSV, Apply)**: Brand color (#0e6c55)
- **Secondary (Export CSV, Cancel)**: White with border

## Accessibility Features

- ✓ Keyboard navigation supported
- ✓ Screen reader compatible
- ✓ High contrast mode support
- ✓ Focus indicators on interactive elements
- ✓ ARIA labels for buttons and controls
- ✓ Semantic HTML structure

## Browser Support

- ✓ Chrome/Edge (latest)
- ✓ Firefox (latest)
- ✓ Safari (latest)
- ✓ Mobile browsers (iOS Safari, Chrome Android)

## Preview

To preview the UI design without backend:
1. Open `BULK_EDIT_UI_PREVIEW.html` in a web browser
2. This shows a static mockup of the interface
3. All interactive features are non-functional in the preview

## Notes

- All operations require admin authentication
- Changes are applied immediately
- Bulk operations are atomic (all succeed or all fail)
- CSV import validates data before inserting
- Export includes all current filters
