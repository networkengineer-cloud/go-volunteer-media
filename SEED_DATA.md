# Seed Data for Demo

This document explains how to populate the database with demo data for testing and demonstrations.

## Quick Start

To populate the database with demo data, simply run:

```bash
make seed
```

This will create:
- 5 demo users (1 admin, 4 regular users)
- Users assigned to appropriate groups (dogs, cats, modsquad)
- 9 demo animals with various statuses (available, foster, bite_quarantine)
- Comments on animals with behavior and medical tags
- Group updates/posts
- Site-wide announcements

## Demo Credentials

After running `make seed`, you can login with these credentials:

### Admin User
- **Username**: `admin`
- **Password**: `demo123`
- **Access**: All groups, full admin privileges

### Regular Users

| Username | Password | Groups | Description |
|----------|----------|--------|-------------|
| sarah_volunteer | demo123 | dogs, modsquad | Active volunteer |
| mike_foster | demo123 | dogs | Dog foster volunteer |
| emma_cats | demo123 | cats | Cat volunteer |
| jake_modsquad | demo123 | modsquad | Behavior modification specialist |

## Command Options

### Basic Seeding
```bash
make seed
```
- Checks if users already exist
- Skips seeding if data is present
- Safe to run multiple times

### Force Seeding
```bash
make seed-force
```
- Seeds data even if users already exist
- **Warning**: May create duplicate data if run multiple times
- Use with caution

### Manual Seeding
```bash
go run cmd/seed/main.go
go run cmd/seed/main.go --force  # Force mode
```

## What Gets Created

### Users (5 total)
- 1 admin with full access
- 4 regular users assigned to different groups
- All users have pre-hashed passwords (bcrypt)
- Mix of email notification preferences

### Animals (9 total)
- **Dogs Group**: 5 animals
  - Buddy (Golden Retriever) - Available, 30 days
  - Luna (German Shepherd Mix) - In foster, 15 days total, 5 in foster
  - Charlie (Beagle) - Available, 10 days
  - Max (Labrador) - Available, 5 days
  - Rocky (Pit Bull) - Bite quarantine, 15 days total, 2 in quarantine
  
- **Cats Group**: 4 animals
  - Mittens (Domestic Shorthair) - Available, 30 days
  - Oliver (Maine Coon Mix) - In foster, 10 days total, 2 in foster
  - Whiskers (Siamese) - Available, 15 days
  - Luna (Domestic Longhair) - Available, 5 days

### Comments (7 total)
- Realistic comments on various animals
- Mix of general updates and tagged comments
- Behavior tags: Rocky, Mittens
- Medical tags: Whiskers
- Comments from different users

### Updates (4 total)
- 2 dog group updates (adoption success, training reminder)
- 2 cat group updates (kitten season, cleanup day)
- Posted by appropriate volunteers

### Announcements (2 total)
- Welcome message (no email)
- Holiday schedule (with email flag)
- Posted by admin user

## Database Requirements

Before running seed:
1. PostgreSQL must be running
2. Database must be created
3. `.env` file configured or environment variables set

### Using Docker Compose
```bash
# Start PostgreSQL
docker compose up -d postgres_dev

# Seed the database
make seed
```

### Manual Database Setup
```bash
# Create database
createdb volunteer_media_dev

# Set environment variables in .env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=volunteer_media_dev
DB_SSLMODE=disable

# Run seed
make seed
```

## Use Cases

### For Development
```bash
# Fresh start with demo data
docker compose down -v
docker compose up -d postgres_dev
make seed
make dev-backend
```

### For Demonstrations
```bash
# Ensure seed data is present
make seed

# Start the application
make dev-backend  # In one terminal
make dev-frontend # In another terminal
```

### For Testing
```bash
# Reset and seed
docker compose down -v
docker compose up -d postgres_dev
make seed
go test ./...
```

## Important Notes

1. **Password Security**: All demo users use the same password (`demo123`) for convenience. **Never use this in production.**

2. **Email Notifications**: Some users have email notifications enabled, but emails won't be sent unless SMTP is configured in `.env`.

3. **Idempotent by Default**: Running `make seed` multiple times is safe - it will skip if users already exist.

4. **Force Mode**: The `--force` flag bypasses the existing user check. Use carefully as it may create duplicates.

5. **Timestamps**: Animal arrival dates and comment timestamps are realistic (varied from 1-30 days ago).

6. **Image URLs**: Image URLs are empty by default. You can upload images through the UI after seeding.

## Cleaning Up

To remove all data and start fresh:

```bash
# Stop and remove volumes
docker compose down -v

# Restart database
docker compose up -d postgres_dev

# Re-seed
make seed
```

## Troubleshooting

### "Database already contains users"
- This means seed data already exists
- Use `make seed-force` to override
- Or clean the database first

### "Failed to connect to database"
- Check PostgreSQL is running: `docker compose ps`
- Verify `.env` configuration
- Check database exists: `psql -l`

### "Failed to create user/animal"
- Check database migrations ran successfully
- Ensure tables exist
- Check for constraint violations

### Build Errors
- Run `go mod tidy` to sync dependencies
- Ensure Go 1.21+ is installed
- Check all imports are correct

## Integration with CI/CD

For automated testing in CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Start Database
  run: docker compose up -d postgres_dev

- name: Seed Database
  run: make seed

- name: Run Tests
  run: go test ./...
```

## Future Enhancements

Potential improvements to seed data:
- [ ] Add more animals per group
- [ ] Include animal images (sample images)
- [ ] Add more comments with varied timestamps
- [ ] Create foster history for animals
- [ ] Add more diverse user roles
- [ ] Include archived animals
- [ ] Add more announcements with varied dates

## Related Documentation

- [README.md](../README.md) - Main project documentation
- [SETUP.md](../SETUP.md) - Setup instructions
- [API.md](../API.md) - API documentation
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
