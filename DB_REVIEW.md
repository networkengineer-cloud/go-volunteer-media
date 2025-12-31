# Database Review - go-volunteer-media

**Review Date:** December 31, 2025  
**Reviewer:** Database Expert Agent  
**Version:** 1.0

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

| Table | Column(s) | Query Pattern | Priority |
|-------|-----------|---------------|----------|
| `animal_comments` | `user_id` | Counting user comments in statistics | High |
| `animal_comments` | `animal_id, created_at` | Fetching comments with ordering | High |
| `animal_images` | `animal_id, is_profile_picture` | Finding profile pictures | Medium |
| `animal_images` | `user_id` | Finding user's images | Medium |
| `user_groups` | `user_id, is_group_admin` | Checking group admin status | High |
| `user_groups` | `group_id` | Fetching group members | Medium |
| `updates` | `group_id, created_at` | Fetching updates with ordering | Medium |
| `protocols` | `group_id, order_index` | Fetching ordered protocols | Low |

### 3.2 Existing Indexes (✅ Good)

The following indexes are already defined:

- `users`: `username` (unique), `email` (unique), `deleted_at`, `default_group_id`
- `animals`: `group_id, status` (composite), `deleted_at`
- `animal_comments`: `animal_id, created_at` (composite)
- `announcements`: `created_at`
- `updates`: `group_id, created_at` (composite)
- `animal_tags`: `group_id, name` (composite unique)
- `comment_tags`: `group_id, name` (composite unique)
- `animal_name_history`: `animal_id, created_at` (composite)
- `animal_images`: `animal_id`

### 3.3 Recommended Index Additions

```sql
-- High Priority
CREATE INDEX idx_animal_comments_user_id ON animal_comments(user_id);
CREATE INDEX idx_user_groups_user_id_admin ON user_groups(user_id, is_group_admin);
CREATE INDEX idx_user_groups_group_id ON user_groups(group_id);

-- Medium Priority
CREATE INDEX idx_animal_images_user_id ON animal_images(user_id);
CREATE INDEX idx_animal_images_profile ON animal_images(animal_id, is_profile_picture) WHERE is_profile_picture = true;

-- Low Priority
CREATE INDEX idx_protocols_group_order ON protocols(group_id, order_index);
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
| `AnimalImage` | Large binary data with nullable `AnimalID` | Consider separate table for orphaned images |
| `UserGroup` | No `created_at` timestamp index | Add index for auditing queries |

### 6.2 Soft Delete Considerations

**Concern:** All major tables use soft deletes (`DeletedAt gorm.DeletedAt`), which:
- Increases table size over time
- Requires `Unscoped()` for admin views
- May impact query performance without proper indexes

**Recommendation:** 
1. Add indexes on `deleted_at` columns (already done for most tables ✅)
2. Consider periodic archival of old soft-deleted records
3. Use partial indexes where appropriate: `WHERE deleted_at IS NULL`

### 6.3 Data Integrity Gaps

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

### 9.1 High Priority (Fix Soon)

1. **Add missing indexes** on `animal_comments.user_id` and `user_groups` columns
2. **Refactor N+1 queries** in `statistics.go` to use aggregated queries
3. **Add connection timeout** to database DSN
4. **Add pagination** to `GetAllUsers` and statistics endpoints

### 9.2 Medium Priority (Plan for Next Sprint)

1. **Add functional index** for case-insensitive name searches
2. **Optimize admin dashboard** with CTEs or caching
3. **Make connection pool settings configurable**
4. **Add query monitoring/logging** for slow queries in production
5. **Review and optimize activity feed** memory usage

### 9.3 Low Priority (Technical Debt)

1. Consider dedicated migration tool
2. Archive old soft-deleted records
3. Add partial indexes for common query patterns
4. Document expected query performance baselines

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
BEGIN;

-- High priority indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_animal_comments_user_id 
    ON animal_comments(user_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_groups_user_admin 
    ON user_groups(user_id, is_group_admin);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_groups_group_id 
    ON user_groups(group_id);

-- Medium priority indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_animal_images_user_id 
    ON animal_images(user_id);

-- Functional index for case-insensitive search
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_animals_name_lower 
    ON animals(LOWER(name));

COMMIT;
```

---

*End of Database Review*
