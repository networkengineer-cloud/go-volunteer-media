# Bulk Edit Page Enhancements - Implementation Guide

## Overview

This document describes the enhancements made to the Bulk Edit Animals page to support individual status/group editing and comment export functionality.

## New Features

### 1. Individual Status Change Dropdowns

Each animal row in the bulk edit table now has an inline status dropdown that allows admins to quickly change an animal's status without bulk selection.

**How it works:**
- Status dropdown appears in the "Status" column for each animal
- Options: Available, Adopted, Fostered
- Change is saved immediately when dropdown value changes
- Visual feedback (row opacity) indicates update in progress
- No page reload required - table updates automatically

**API Endpoint:**
```
PUT /api/admin/animals/:animalId
Body: { "status": "adopted" }
```

### 2. Individual Group Change Dropdowns

Each animal row also has an inline group dropdown that allows admins to quickly move an animal to a different group.

**How it works:**
- Group dropdown appears in the "Group" column for each animal
- Shows all available groups
- Change is saved immediately when dropdown value changes
- Visual feedback (row opacity) indicates update in progress
- No page reload required - table updates automatically

**API Endpoint:**
```
PUT /api/admin/animals/:animalId
Body: { "group_id": 2 }
```

### 3. Export Comments CSV

A new "Export Comments" button allows admins to download a comprehensive CSV file containing all animal comments with related animal data.

**CSV Columns:**
- `comment_id` - Unique comment identifier
- `animal_id` - Animal identifier
- `animal_name` - Name of the animal
- `animal_species` - Species (Dog, Cat, etc.)
- `animal_breed` - Breed information
- `animal_status` - Current status (available, adopted, fostered)
- `group_id` - Group identifier
- `group_name` - Name of the group
- `comment_content` - Full comment text
- `comment_author` - Username of comment author
- `comment_tags` - Tags applied to comment (semicolon separated)
- `created_at` - Comment creation timestamp (ISO 8601)
- `updated_at` - Comment last update timestamp (ISO 8601)

**API Endpoint:**
```
GET /api/admin/animals/export-comments-csv
Query Parameters:
  - group_id: Optional filter by group
Response: CSV file download (animal-comments.csv)
```

## User Interface Changes

### Page Header
- Renamed "Export CSV" button to "Export Animals" for clarity
- Added new "Export Comments" button
- Button order: Export Comments | Export Animals | Import CSV

### Table Columns
- **Status column**: Changed from static badge to interactive dropdown
- **Group column**: Changed from static text to interactive dropdown
- **Visual feedback**: Rows show reduced opacity during updates

### Styling
- Added `.inline-select` class for dropdown styling
- Hover effect: Border changes to brand color
- Focus effect: Border and subtle shadow in brand color
- Disabled state: Reduced opacity and no pointer cursor
- Updating state: Row opacity reduced to 0.6

## Technical Implementation

### Backend (Go)

**New Handler: `ExportAnimalCommentsCSV`**
```go
// Location: internal/handlers/animal.go
func ExportAnimalCommentsCSV(db *gorm.DB) gin.HandlerFunc
```
- Fetches comments with preloaded User and Tags
- Joins with animals to get animal details
- Fetches group information for group names
- Supports group_id filter via query parameter
- Returns CSV with comprehensive comment and animal data

**New Handler: `UpdateAnimalAdmin`**
```go
// Location: internal/handlers/animal.go
func UpdateAnimalAdmin(db *gorm.DB) gin.HandlerFunc
```
- Updates individual animal by ID
- Admin-only access (no group membership check)
- Supports partial updates (only provided fields)
- Returns updated animal data

**Updated Struct: `AnimalRequest`**
- Added `GroupID` field for group updates
- Made fields optional for partial updates

**New Routes:**
```go
// Location: cmd/api/main.go
admin.GET("/animals/export-comments-csv", handlers.ExportAnimalCommentsCSV(db))
admin.PUT("/animals/:animalId", handlers.UpdateAnimalAdmin(db))
```

### Frontend (React + TypeScript)

**New State:**
```typescript
const [updatingAnimal, setUpdatingAnimal] = useState<number | null>(null);
```

**New Functions:**
```typescript
handleIndividualStatusChange(animalId, newStatus)
handleIndividualGroupChange(animalId, newGroupId)
handleExportCommentsCSV()
```

**Updated API Client:**
```typescript
// Location: frontend/src/api/client.ts
exportCommentsCSV: (groupId?: number) => Promise<Blob>
updateAnimal: (animalId: number, data: Partial<Animal>) => Promise<Animal>
```

**Updated Component:**
- Status column now renders `<select>` instead of `<span>`
- Group column now renders `<select>` instead of text
- Added "Export Comments" button to header
- Row gets `updating` class when update in progress

## Usage Examples

### Change Animal Status
1. Navigate to Bulk Edit Animals page (`/admin/animals`)
2. Find the animal you want to update
3. Click the status dropdown in that animal's row
4. Select new status (Available, Adopted, or Fostered)
5. Change is saved automatically
6. Row briefly shows reduced opacity during update

### Change Animal Group
1. Navigate to Bulk Edit Animals page (`/admin/animals`)
2. Find the animal you want to move
3. Click the group dropdown in that animal's row
4. Select the target group
5. Change is saved automatically
6. Row briefly shows reduced opacity during update

### Export Comments
1. Navigate to Bulk Edit Animals page (`/admin/animals`)
2. (Optional) Filter by group using the "Filter by Group" dropdown
3. Click "Export Comments" button
4. CSV file `animal-comments.csv` downloads automatically
5. Open in Excel or any CSV viewer to analyze comment data

### Filter and Export Comments
1. Select a specific group from "Filter by Group" dropdown
2. Click "Export Comments" button
3. Downloaded CSV contains only comments for animals in selected group
4. Useful for group-specific reporting and analysis

## Benefits

### For Administrators
- **Faster editing**: No need to select animals and use bulk actions for single changes
- **Immediate feedback**: See changes reflected instantly in the table
- **Comprehensive export**: Get all comment data with animal context in one file
- **Flexible reporting**: Filter exports by group for targeted analysis

### For Data Analysis
- **Complete context**: Comments include all relevant animal information
- **Timestamp tracking**: Both creation and update timestamps included
- **Tag support**: Comment tags included for categorization
- **Author tracking**: Know who wrote each comment

## Testing Checklist

- [ ] Individual status change updates correctly
- [ ] Individual group change updates correctly
- [ ] Visual feedback works during updates
- [ ] Multiple quick changes are handled correctly
- [ ] Export Comments downloads valid CSV
- [ ] CSV contains expected columns and data
- [ ] Group filter works with comment export
- [ ] Comment export works when no comments exist
- [ ] Bulk actions still work as expected
- [ ] Page layout remains responsive on mobile

## Future Enhancements

Potential improvements for future iterations:
- Batch edit multiple fields at once (e.g., status + group)
- Export with date range filter
- Export in additional formats (Excel, JSON)
- Inline editing for other animal fields (name, breed, etc.)
- Confirmation dialog for status changes
- Undo/redo functionality
- Comment reply and thread export
- Export with custom column selection

## Security Considerations

- All endpoints require admin authentication
- Input validation on all update requests
- Proper error handling prevents data corruption
- Group filter applied consistently for data access
- No sensitive data exposed in exports (passwords excluded)

## Performance Notes

- Individual updates use targeted PUT requests (efficient)
- Export uses single database query with joins (optimized)
- Comment export includes preloading for performance
- No pagination needed for exports (generates complete file)
- Browser handles CSV creation without server-side file storage

## Support

For issues or questions:
1. Check error messages in browser console
2. Verify admin permissions are active
3. Test with different groups and animals
4. Review API response codes for specific errors
5. Contact system administrator if issues persist
