# Database Review - go-volunteer-media

**Review Date:** December 31, 2025  
**Reviewer:** Database Expert Agent  
**Version:** 1.1  
**Last Updated:** December 31, 2025 (Corrected index findings)

---

## Executive Summary

This document provides a comprehensive review of the database setup, configuration, and usage patterns in the go-volunteer-media project. The review covers security concerns, performance issues, missing indexes, query optimization opportunities, and schema design recommendations.

**Overall Assessment:** The database implementation is reasonably well-designed with proper use of GORM, connection pooling, and security measures. However, there are several areas for improvement, particularly around query performance, missing indexes, and potential N+1 query issues.

---

## Table of Contents

1. [Database Setup & Configuration](#1-database-setup--configuration)
2. [Security Concerns](#2-security-concerns)
3. [Missing Indexes](#3-missing-indexes)
4. [N+1 Query Issues](#4-n1-query-issues)
5. [Query Performance Issues](#5-query-performance-issues)
6. [Schema Design Recommendations](#6-schema-design-recommendations)
7. [Migration Concerns](#7-migration-concerns)
8. [Connection Pool Configuration](#8-connection-pool-configuration)
9. [Recommendations Summary](#9-recommendations-summary)

---

## 1. Database Setup & Configuration

### 1.1 Connection Configuration (✅ Good)

**File:** `internal/database/database.go`

**Strengths:**
- ✅ Environment variable-based configuration
- ✅ SSL mode validation with whitelist approach
- ✅ Connection pool properly configured
- ✅ Statement timeout set to 30 seconds

**Issues:**

| Issue | Severity | Description |
|-------|----------|-------------|
| Default credentials in development | Low | Default values (`postgres`/`postgres`) are set when env vars are missing. While acceptable for development, these should never be used in production. |
| No connection timeout | Medium | The database connection does not specify a connection timeout, which could cause the application to hang if the database is unreachable. |

**Recommendation:**
```go
// Add connection timeout to DSN
dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=10",
    dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
```

### 1.2 SSL Configuration

| Issue | Severity | Description |
|-------|----------|-------------|
| Default SSL mode is `disable` | Medium | In production, SSL should be `require` or `verify-full`. The default `disable` is only appropriate for development. |

**Recommendation:** Add a warning log when SSL is disabled and the environment appears to be production.

---

## 2. Security Concerns

### 2.1 SQL Injection Prevention (✅ Mostly Good)

**Strengths:**
- ✅ GORM parameterized queries used throughout
- ✅ `escapeSQLWildcards` function for LIKE queries in `animal_crud.go`
- ✅ SSL mode validation prevents DSN injection

**Issues:**

| File | Line | Issue | Severity |
|------|------|-------|----------|
| `database.go` | 221-228 | Table name interpolation in `fixAndEnforceTableConstraint` | Low |
| `database.go` | 233, 245, 258-261, 271 | Table name interpolation in SQL queries | Low |

**Example of concern:**
```go
// Line 221-228 - Table name is interpolated into SQL
query := fmt.Sprintf(`
    SELECT EXISTS (
        SELECT FROM information_schema.columns
        WHERE table_name = '%s' AND column_name = 'group_id'
    )
`, tableName)
```

**Mitigation:** While these table names are hardcoded internal values (not user input), it's a best practice to use the `quoteIdentifier` function consistently or use GORM's safer table reference methods.

### 2.2 Password Handling (✅ Good)

- ✅ bcrypt used for password hashing
- ✅ Passwords excluded from JSON with `json:"-"` tag
- ✅ JWT secret validation for entropy and length
- ✅ Account lockout after 5 failed attempts

### 2.3 Sensitive Data Exposure

| Issue | Severity | Description |
|-------|----------|-------------|
| Binary image data in memory | Low | Large image data (`ImageData []byte`) is stored in database. While properly excluded from JSON, large blobs can impact database performance and backup size. |
| GroupMe Bot ID stored in database | Low | Bot IDs are stored in plain text. Consider if this is sensitive for your use case. |

---

## 3. Missing Indexes

### 3.1 Critical Missing Indexes

**Note:** The following assessment is based on GORM model tag definitions. Use `SELECT * FROM pg_indexes WHERE tablename = 'table_name';` to verify actual index creation in PostgreSQL.

| Table | Column(s) | Query Pattern | Priority | Notes |
|-------|-----------|---------------|----------|-------|
| `user_groups` | `user_id, is_group_admin` | Checking group admin status | **Critical** | Only has composite PK (user_id, group_id), no additional indexes |
| `user_groups` | `group_id` | Fetching group members via JOIN | **Critical** | Required for efficient `JOIN user_groups ON user_groups.group_id = groups.id` |
| `protocols` | `group_id, order_index` | Fetching ordered protocols | Low | OrderIndex used for ordering but only group_id is indexed |

### 3.2 Existing Indexes (✅ Verified in Models)

The following indexes are **defined in GORM tags** and should be created by AutoMigrate:

- `users`: `username` (unique), `email` (unique), `deleted_at`, `default_group_id`
- `animals`: `group_id, status` (composite), `deleted_at`
- `animal_comments`: `user_id` (index), `animal_id, created_at` (composite), `deleted_at`
- `animal_images`: `animal_id`, `user_id` (index), `deleted_at`
- `announcements`: `created_at`
- `updates`: `group_id, created_at` (composite)
- `animal_tags`: `group_id, name` (composite unique)
- `comment_tags`: `group_id, name` (composite unique)
- `animal_name_history`: `animal_id, created_at` (composite)

### 3.3 Indexes to Add

```sql
-- Critical Priority (user_groups has no indexes beyond PK)
CREATE INDEX idx_user_groups_user_admin ON user_groups(user_id, is_group_admin);
CREATE INDEX idx_user_groups_group_id ON user_groups(group_id);

-- Medium Priority
CREATE INDEX idx_animal_images_profile ON animal_images(animal_id, is_profile_picture) WHERE is_profile_picture = true;

-- Low Priority
CREATE INDEX idx_protocols_group_order ON protocols(group_id, order_index);
```

### 3.4 Index Verification Command

To confirm GORM actually created indexes, run:
```sql
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename IN ('animal_comments', 'animal_images', 'user_groups', 'protocols')
ORDER BY tablename, indexname;
```

---

## 4. N+1 Query Issues

### 4.1 Critical N+1 Patterns

| File | Function | Issue | Severity |
|------|----------|-------|----------|
| `statistics.go` | `GetGroupStatistics` | Loop over groups with 3 queries per group | High |
| `statistics.go` | `GetUserStatistics` | Loop over users with 3 queries per user | High |
| `statistics.go` | `GetCommentTagStatistics` | Loop over tags with 4 queries per tag | High |
| `activity_feed.go` | `GetGroupActivityFeed` | Fetches animals, then comments separately | Medium |

**Example - GetGroupStatistics (Lines 38-88):**
```go
for i, group := range groups {
    // Query 1: Count users
    db.WithContext(ctx).Model(&models.User{}).
        Joins("JOIN user_groups...").
        Count(&userCount)

    // Query 2: Count animals
    db.WithContext(ctx).Model(&models.Animal{}).
        Where("group_id = ?", group.ID).
        Count(&animalCount)

    // Query 3: Get last comment
    db.WithContext(ctx).
        Joins("JOIN animals...").
        First(&comment)
}
```

**Impact:** For 10 groups, this executes 30+ queries instead of 3 optimized queries.

**Recommendation:** Use a single aggregated query:
```go
type GroupStats struct {
    GroupID      uint
    UserCount    int64
    AnimalCount  int64
    LastActivity *time.Time
}

db.Raw(`
    SELECT 
        g.id as group_id,
        (SELECT COUNT(DISTINCT ug.user_id) FROM user_groups ug WHERE ug.group_id = g.id) as user_count,
        (SELECT COUNT(*) FROM animals a WHERE a.group_id = g.id) as animal_count,
        (SELECT MAX(ac.created_at) FROM animal_comments ac 
         JOIN animals a ON a.id = ac.animal_id WHERE a.group_id = g.id) as last_activity
    FROM groups g
`).Scan(&groupStats)
```

### 4.2 Potential N+1 in Preload Chains

| File | Function | Pattern | Risk |
|------|----------|---------|------|
| `animal_comment.go` | `GetDeletedComments` | Building `animalMap` in loop | Low (in-memory) |
| `group.go` | `GetGroupMembers` | Building `members` slice in loop | Low (in-memory) |

---

## 5. Query Performance Issues

### 5.1 Expensive Queries

| File | Function | Issue | Impact |
|------|----------|-------|--------|
| `admin_dashboard.go` | `GetAdminDashboardStats` | 10+ subqueries in single handler | High |
| `statistics.go` | All functions | Loops with multiple queries | High |
| `activity_feed.go` | `GetGroupActivityFeed` | In-memory sorting of all items | Medium |

**Admin Dashboard Example (Lines 106-123):**
```go
db.WithContext(ctx).
    Model(&models.Group{}).
    Select(`
        groups.id as group_id,
        groups.name as group_name,
        (SELECT COUNT(DISTINCT user_groups.user_id)...) as user_count,
        (SELECT COUNT(*) FROM animals...) as animal_count,
        (SELECT COUNT(*) FROM animal_comments...) as comment_count,
        (SELECT MAX(animal_comments.created_at)...) as last_activity
    `, thirtyDaysAgo).
    ...
```

**Issue:** Correlated subqueries execute for each row, causing O(n²) performance.

**Recommendation:** Use CTEs or materialized views for dashboard statistics, or cache results.

### 5.2 LIKE Query Performance

| File | Function | Issue |
|------|----------|-------|
| `animal_crud.go` | `GetAnimals` | `LOWER(name) LIKE ?` prevents index usage |
| `animal_helpers.go` | `CheckDuplicateNames` | `LOWER(name) = ?` prevents index usage |

**Recommendation:** 
1. Add a functional index: `CREATE INDEX idx_animals_name_lower ON animals(LOWER(name));`
2. Or use PostgreSQL's `citext` extension for case-insensitive text columns

### 5.3 Missing Query Limits

| File | Function | Issue | Severity |
|------|----------|-------|----------|
| `statistics.go` | `GetGroupStatistics` | No limit on groups fetched | Medium |
| `statistics.go` | `GetUserStatistics` | No limit on users fetched | Medium |
| `user.go` | `GetAllUsers` | No pagination | Medium |
| `announcement.go` | `sendAnnouncementEmails` | Fetches all opted-in users | Low |

---

## 6. Schema Design Recommendations

### 6.1 Model Issues

| Model | Issue | Recommendation |
|-------|-------|----------------|
| `Animal` | Binary protocol document data stored in DB | Use external storage (Azure Blob) consistently |
| `AnimalImage` | Nullable `AnimalID` allows orphaned images | Add scheduled cleanup job for old unlinked images (e.g., images with `animal_id IS NULL AND created_at < NOW() - INTERVAL '7 days'`) |
| `UserGroup` | **No indexes beyond composite PK** | Critical: Add indexes on `(user_id, is_group_admin)` and `group_id` for JOIN queries |

### 6.2 Orphaned Image Cleanup Concern

**Issue:** `AnimalImage.AnimalID` is nullable by design to allow image uploads before animal creation. However, this can lead to unbounded growth of orphaned images.

**Recommendation:** Add a scheduled cleanup job:
```sql
-- Delete orphaned images older than 7 days
DELETE FROM animal_images 
WHERE animal_id IS NULL 
  AND created_at < NOW() - INTERVAL '7 days';
```

### 6.3 Soft Delete Considerations

**Concern:** All major tables use soft deletes (`DeletedAt gorm.DeletedAt`), which:
- Increases table size over time
- Requires `Unscoped()` for admin views
- May impact query performance without proper indexes

**Recommendation:** 
1. Add indexes on `deleted_at` columns (already done for most tables ✅)
2. Consider periodic archival of old soft-deleted records
3. Use partial indexes where appropriate: `WHERE deleted_at IS NULL`

### 6.4 Missing created_at Indexes for Ordering

Several tables use `ORDER BY created_at DESC` but lack dedicated indexes:

| Table | Has created_at Index | Notes |
|-------|---------------------|-------|
| `protocols` | ❌ No | `created_at` may be used in admin views |
| `user_groups` | ❌ No | Useful for auditing when users joined groups |

**Note:** Tables with composite indexes including `created_at` (like `animal_comments`, `updates`) already have optimized ordering.

### 6.5 Data Integrity Gaps

| Issue | Tables Affected | Recommendation |
|-------|-----------------|----------------|
| No foreign key constraint on `AnimalImage.AnimalID` | `animal_images` | Nullable FK is intentional for unlinked images |
| No cascade delete on animal comments | `animal_comments` | Add `ON DELETE CASCADE` or handle in application |
| `ProtocolDocumentUserID` is nullable | `animals` | Should track who uploaded documents |

---

## 7. Migration Concerns

### 7.1 AutoMigrate Limitations

| Issue | Severity | Description |
|-------|----------|-------------|
| No version tracking | Medium | GORM AutoMigrate doesn't track migration versions, making rollbacks difficult |
| Constraint modifications | Medium | AutoMigrate can't modify existing constraints easily |
| Large table alterations | Low | Adding NOT NULL to large tables can lock tables |

### 7.2 Current Migration Workarounds

The codebase implements custom migration logic in `fixAndEnforceGroupIDConstraints()`:
- ✅ Fixes NULL values before adding NOT NULL
- ✅ Creates defaults before constraint enforcement
- ⚠️ Uses raw SQL which bypasses GORM's type safety

### 7.3 Recommendations

1. Consider using a dedicated migration tool (golang-migrate, goose)
2. Add migration version tracking table
3. Test migrations on production-size data before deployment

---

## 8. Connection Pool Configuration

### 8.1 Current Settings (✅ Good)

```go
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(1 * time.Hour)
sqlDB.SetConnMaxIdleTime(10 * time.Minute)
db.Exec("SET statement_timeout = '30s'")
```

### 8.2 Recommendations

| Setting | Current | Recommendation | Reason |
|---------|---------|----------------|--------|
| MaxIdleConns | 10 | 10-25 | Increase if seeing connection churn |
| MaxOpenConns | 100 | Depends on DB | Match with PostgreSQL's `max_connections` |
| ConnMaxLifetime | 1 hour | 30 min - 1 hour | Good for connection recycling |
| ConnMaxIdleTime | 10 min | 5-10 min | Good for cleanup |
| statement_timeout | 30s | 30s-60s | Good for preventing runaway queries |

**Note:** Consider making pool settings configurable via environment variables for production tuning.

---

## 9. Recommendations Summary

### 9.1 Critical Priority (Fix Immediately)

1. ✅ **COMPLETED** - **Add missing indexes on `user_groups`** - This table has NO indexes beyond the composite primary key, causing slow JOINs:
   - ✅ Added `index:idx_user_groups_user_admin` to `UserID` and `IsGroupAdmin` fields in UserGroup model
   - ✅ Added `index:idx_user_groups_group_id` to `GroupID` field in UserGroup model
   - **Implementation:** Modified `internal/models/models.go` to include composite index tags

### 9.2 High Priority (Fix Soon)

1. ✅ **COMPLETED** - **Refactor N+1 queries** in `statistics.go` to use aggregated queries
   - ✅ Refactored `GetGroupStatistics` to use single aggregated query with subqueries
   - ✅ Refactored `GetUserStatistics` to use single aggregated query with subqueries
   - ✅ Refactored `GetCommentTagStatistics` to use CTEs with window functions
   - **Impact:** Reduced queries from 3N+1 to 1 for group stats, 3N+1 to 1 for user stats, 4N+1 to 1 for tag stats
   - **Implementation:** Modified `internal/handlers/statistics.go` with optimized SQL queries

2. ✅ **COMPLETED** - **Add connection timeout** to database DSN
   - ✅ Added `connect_timeout=10` to DSN connection string
   - **Implementation:** Modified `internal/database/database.go` line 59

3. **TODO** - **Add pagination** to `GetAllUsers` and statistics endpoints
4. **TODO** - **Implement orphaned image cleanup** job for `animal_images` where `animal_id IS NULL`

### 9.3 Medium Priority (Plan for Next Sprint)

1. ✅ **COMPLETED** - **Add functional index** for case-insensitive name searches
   - ✅ Added `idx_animals_name_lower` functional index using `LOWER(name)`
   - **Implementation:** Added `createCustomIndexes()` function in `internal/database/database.go`

2. ✅ **COMPLETED** - **Add composite index** on `protocols(group_id, order_index)`
   - ✅ Added `index:idx_protocols_group_order` to `GroupID` and `OrderIndex` fields in Protocol model
   - **Implementation:** Modified `internal/models/models.go`

3. **TODO** - **Optimize admin dashboard** with CTEs or caching
4. **TODO** - **Make connection pool settings configurable**
5. **TODO** - **Add query monitoring/logging** for slow queries in production

### 9.4 Low Priority (Technical Debt)

1. Consider dedicated migration tool
2. Archive old soft-deleted records
3. ✅ **COMPLETED** - Add partial indexes for common query patterns
   - ✅ Added composite index on `animal_images(animal_id, is_profile_picture)` for profile picture queries
4. Document expected query performance baselines
5. Verify GORM actually created declared indexes using `pg_indexes`

---

## Appendix A: Query Performance Testing Commands

```sql
-- Find slow queries (requires pg_stat_statements extension)
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 20;

-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC;

-- Find missing indexes
SELECT schemaname, relname, seq_scan, seq_tup_read, idx_scan
FROM pg_stat_user_tables
WHERE seq_scan > 1000
ORDER BY seq_tup_read DESC;
```

---

## Appendix B: Suggested Index Migration

```sql
-- Run these in a transaction during maintenance window
-- Note: CREATE INDEX CONCURRENTLY cannot run inside a transaction block
-- Run each statement separately for production

-- Critical priority (user_groups has NO indexes beyond PK)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_groups_user_admin 
    ON user_groups(user_id, is_group_admin);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_groups_group_id 
    ON user_groups(group_id);

-- Medium priority
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_animal_images_profile 
    ON animal_images(animal_id, is_profile_picture) 
    WHERE is_profile_picture = true;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_protocols_group_order 
    ON protocols(group_id, order_index);

-- Functional index for case-insensitive search
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_animals_name_lower 
    ON animals(LOWER(name));
```

## Appendix C: Orphaned Image Cleanup Query

```sql
-- Find orphaned images (for review before deletion)
SELECT id, image_url, user_id, created_at 
FROM animal_images 
WHERE animal_id IS NULL 
  AND created_at < NOW() - INTERVAL '7 days';

-- Delete orphaned images older than 7 days
DELETE FROM animal_images 
WHERE animal_id IS NULL 
  AND created_at < NOW() - INTERVAL '7 days';
```

---

*End of Database Review*
