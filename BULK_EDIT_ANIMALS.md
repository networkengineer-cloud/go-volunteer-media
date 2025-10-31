# Bulk Edit Animals Feature

## Overview

The Bulk Edit Animals feature allows administrators to efficiently manage large numbers of animals in the HAWS Volunteer Portal. This feature includes:

1. **Bulk Operations**: Select multiple animals and apply actions in bulk
2. **CSV Import**: Import animals from a CSV file
3. **CSV Export**: Export animals to a CSV file for backup or editing

## Features

### Bulk Animal Management Page (`/admin/animals`)

- **View All Animals**: See all animals across all groups in a single table
- **Filtering**: Filter animals by status, group, and name
- **Bulk Selection**: Select individual animals or all animals at once
- **Bulk Actions**:
  - Move selected animals to a different group
  - Change the status of selected animals (available, adopted, fostered)

### CSV Import

Import animals from a CSV file with the following columns:

| Column      | Required | Description                           |
|-------------|----------|---------------------------------------|
| group_id    | Yes      | ID of the group to assign animal to   |
| name        | Yes      | Name of the animal                    |
| species     | No       | Species (e.g., Dog, Cat)             |
| breed       | No       | Breed of the animal                   |
| age         | No       | Age in years                          |
| description | No       | Description of the animal             |
| status      | No       | Status (available/adopted/fostered)   |
| image_url   | No       | URL to animal's image                 |

**Features**:
- Validation with detailed error messages
- Warnings for invalid rows (skipped)
- Success message with import count
- Sample CSV file included: `sample_animals.csv`

### CSV Export

Export animals to CSV format:
- Export all animals or filter by group
- Downloaded as `animals.csv`
- Can be used as a template for imports
- Includes all animal data fields

## API Endpoints

### Admin Endpoints (require admin authentication)

#### Get All Animals
```
GET /api/admin/animals
Query Parameters:
  - status: Filter by status (optional)
  - group_id: Filter by group (optional)
  - name: Search by name (optional)
Response: Array of Animal objects
```

#### Bulk Update Animals
```
POST /api/admin/animals/bulk-update
Body:
  {
    "animal_ids": [1, 2, 3],
    "group_id": 2,           // optional
    "status": "adopted"      // optional
  }
Response:
  {
    "message": "Successfully updated 3 animals",
    "count": 3
  }
```

#### Import CSV
```
POST /api/admin/animals/import-csv
Content-Type: multipart/form-data
Body:
  file: CSV file
Response:
  {
    "message": "Successfully imported 5 animals",
    "count": 5,
    "warnings": ["Line 3: Invalid age", ...]  // optional
  }
```

#### Export CSV
```
GET /api/admin/animals/export-csv
Query Parameters:
  - group_id: Filter by group (optional)
Response: CSV file download
```

## Usage

### Accessing the Feature

1. Log in as an administrator
2. Click **Animals** in the navigation menu
3. You'll see the bulk edit page with all animals

### Moving Animals to a Different Group

1. Filter animals if needed (by status, group, or name)
2. Select the animals you want to move by checking their checkboxes
3. Select **Move to Group** from the action dropdown
4. Choose the target group
5. Click **Apply**
6. Animals will be moved to the new group

Example: Moving 3 dogs to the "Mod Squad" group:
1. Filter by Group = "Dogs"
2. Select the 3 dogs
3. Action = "Move to Group"
4. Target = "Mod Squad"
5. Apply

### Changing Animal Status

1. Select the animals you want to update
2. Select **Change Status** from the action dropdown
3. Choose the new status (available, adopted, fostered)
4. Click **Apply**

### Importing from CSV

1. Prepare your CSV file (see `CSV_IMPORT_GUIDE.md` and `sample_animals.csv`)
2. Click **Import CSV** button
3. Select your CSV file
4. Click **Import**
5. Review success message and any warnings
6. Animals will appear in the table

### Exporting to CSV

1. (Optional) Filter by group to export specific animals
2. Click **Export CSV** button
3. The file will download as `animals.csv`
4. Open in Excel or any CSV editor

## Technical Details

### Backend Implementation

**Files Modified**:
- `internal/handlers/animal.go`: Added bulk update, CSV import/export handlers
- `cmd/api/main.go`: Added new admin routes

**New Handlers**:
- `BulkUpdateAnimals`: Updates multiple animals at once
- `ImportAnimalsCSV`: Parses CSV and creates animals
- `ExportAnimalsCSV`: Generates CSV from animal data
- `GetAllAnimals`: Returns all animals with filtering

**Security**:
- All endpoints require admin authentication
- CSV files are validated (size, type, content)
- SQL injection protected through GORM
- Input validation and sanitization

### Frontend Implementation

**Files Created**:
- `frontend/src/pages/BulkEditAnimalsPage.tsx`: Main component
- `frontend/src/pages/BulkEditAnimalsPage.css`: Styling

**Files Modified**:
- `frontend/src/App.tsx`: Added route for `/admin/animals`
- `frontend/src/components/Navigation.tsx`: Added navigation link
- `frontend/src/api/client.ts`: Added API methods

**Features**:
- Responsive design for mobile and desktop
- Real-time filtering without page reload
- Visual feedback for selected animals
- Modal dialog for CSV import
- Error handling with user-friendly messages
- Dark mode support

## Testing

### Manual Testing

1. **Bulk Update**:
   - Select animals across different groups
   - Move them to a new group
   - Verify they appear in the correct group
   - Check status changes work correctly

2. **CSV Import**:
   - Use `sample_animals.csv`
   - Import and verify animals are created
   - Test with invalid data to verify error handling

3. **CSV Export**:
   - Export all animals
   - Export filtered animals (by group)
   - Verify CSV format matches import format

4. **Filtering**:
   - Test status filter
   - Test group filter
   - Test name search
   - Combine multiple filters

### Automated Testing

Playwright tests are included in `frontend/tests/bulk-edit-animals.spec.ts` covering:
- Page accessibility
- Filtering functionality
- Bulk selection
- Bulk actions
- CSV import/export
- Admin access control

## Future Enhancements

Potential improvements:
- Batch delete functionality
- More bulk actions (e.g., update breed, species)
- CSV import preview before committing
- Undo functionality
- Export filtered results
- Template download button
- Drag-and-drop CSV upload
- Progress bar for large imports
- Audit log for bulk changes

## Documentation Files

- `CSV_IMPORT_GUIDE.md`: Detailed CSV import instructions
- `sample_animals.csv`: Sample CSV file for testing
- `BULK_EDIT_ANIMALS.md`: This file

## Support

For issues or questions:
1. Check the CSV import guide
2. Review error messages in the UI
3. Check browser console for detailed errors
4. Contact system administrator
