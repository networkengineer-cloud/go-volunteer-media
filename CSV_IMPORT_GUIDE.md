# Animal CSV Import Template

This template can be used to bulk import animals into the HAWS Volunteer Portal.

## CSV Format

The CSV file should have the following columns (in any order):

| Column      | Required | Type    | Description                                  | Example         |
|-------------|----------|---------|----------------------------------------------|-----------------|
| group_id    | Yes      | number  | ID of the group to assign the animal to     | 1               |
| name        | Yes      | string  | Name of the animal                           | Max             |
| species     | No       | string  | Species (e.g., Dog, Cat)                    | Dog             |
| breed       | No       | string  | Breed of the animal                          | Labrador        |
| age         | No       | number  | Age of the animal in years                   | 3               |
| description | No       | string  | Description of the animal                    | Friendly dog    |
| status      | No       | string  | Status: available, adopted, or fostered      | available       |
| image_url   | No       | string  | URL to the animal's image                    | /uploads/max.jpg|

## Example CSV File

```csv
group_id,name,species,breed,age,description,status,image_url
1,Max,Dog,Labrador,3,Friendly and energetic,available,
1,Bella,Dog,German Shepherd,5,Great with kids,available,
2,Whiskers,Cat,Tabby,2,Loves to play,available,
2,Shadow,Cat,Persian,4,Very calm and quiet,fostered,
```

## Import Steps

1. Navigate to the **Animals** page in the admin menu
2. Click the **Import CSV** button
3. Select your CSV file
4. Review any warnings for rows that couldn't be imported
5. The successfully imported animals will appear in the list

## Notes

- Only `group_id` and `name` are required fields
- If `status` is not provided, it defaults to "available"
- The CSV file must have a header row with column names
- Column names are case-insensitive
- Empty fields are allowed for optional columns
- Invalid rows will be skipped with a warning message
- You can export existing animals to CSV as a template

## Export CSV

To export animals to CSV:

1. Navigate to the **Animals** page in the admin menu
2. (Optional) Filter by group to export only specific animals
3. Click the **Export CSV** button
4. The CSV file will be downloaded to your computer

The exported CSV can be used as a template for importing or as a backup of your animal data.
