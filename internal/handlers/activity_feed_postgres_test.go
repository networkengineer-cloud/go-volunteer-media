package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// sqlCapturingLogger records every SQL statement GORM executes through it,
// so a test can inspect the *actual* query a handler issued instead of
// re-declaring an equivalent query by hand (which would only prove that
// hand-written query is bounded, not that the handler's real one is).
type sqlCapturingLogger struct {
	statements []string
}

func (l *sqlCapturingLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }
func (l *sqlCapturingLogger) Info(context.Context, string, ...interface{})    {}
func (l *sqlCapturingLogger) Warn(context.Context, string, ...interface{})    {}
func (l *sqlCapturingLogger) Error(context.Context, string, ...interface{})   {}
func (l *sqlCapturingLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	l.statements = append(l.statements, sql)
}

// These tests reproduce the shape of the production incident this fix
// addresses: GET /groups/:id/activity-feed loading every comment across
// every animal in a group (no LIMIT), dominating request latency. They
// reuse openSearchTestPostgres/envOrDefault/tcpReachable from
// search_postgres_test.go — same reachable-Postgres-or-skip contract, same
// reason (SQLite, used by the rest of this package's tests, can't exercise
// the metadata jsonb predicates this handler now pushes into SQL).

type activityFeedTestFixture struct {
	tx    *gorm.DB
	group models.Group
	user  models.User
}

func newActivityFeedTestFixture(t *testing.T, db *gorm.DB) *activityFeedTestFixture {
	t.Helper()
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	unique := time.Now().UnixNano()
	group := models.Group{Name: fmt.Sprintf("ActivityFeedTest-%d", unique)}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	user := models.User{
		Username: fmt.Sprintf("afeedtest-%d", unique),
		Email:    fmt.Sprintf("afeedtest-%d@example.com", unique),
		Password: "x",
	}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := tx.Create(&models.UserGroup{UserID: user.ID, GroupID: group.ID}).Error; err != nil {
		t.Fatalf("add user to group: %v", err)
	}
	return &activityFeedTestFixture{tx: tx, group: group, user: user}
}

// seedAnimalsWithComments creates numAnimals animals, each with
// commentsPerAnimal comments, spaced one minute apart so ordering is
// unambiguous. metaFor lets a test vary each comment's SessionMetadata
// (or return nil for none) by its overall sequence number.
func (f *activityFeedTestFixture) seedAnimalsWithComments(t *testing.T, numAnimals, commentsPerAnimal int, metaFor func(seq int) *models.SessionMetadata) []models.Animal {
	t.Helper()
	base := time.Now().Add(-24 * time.Hour)
	animals := make([]models.Animal, numAnimals)
	seq := 0
	for a := 0; a < numAnimals; a++ {
		animal := models.Animal{GroupID: f.group.ID, Name: fmt.Sprintf("Animal%d", a), Species: "Dog", Status: "available"}
		if err := f.tx.Create(&animal).Error; err != nil {
			t.Fatalf("create animal: %v", err)
		}
		animals[a] = animal
		for c := 0; c < commentsPerAnimal; c++ {
			seq++
			comment := models.AnimalComment{
				AnimalID: animal.ID,
				UserID:   f.user.ID,
				Content:  fmt.Sprintf("comment %d on animal %d", c, a),
				Metadata: metaFor(seq),
			}
			if err := f.tx.Create(&comment).Error; err != nil {
				t.Fatalf("create comment: %v", err)
			}
			// CreatedAt is stamped by GORM on Create; overwrite with a
			// synthetic, strictly-increasing value so ordering assertions
			// are deterministic regardless of how fast the inserts run.
			createdAt := base.Add(time.Duration(seq) * time.Minute)
			if err := f.tx.Model(&comment).UpdateColumn("created_at", createdAt).Error; err != nil {
				t.Fatalf("set created_at: %v", err)
			}
		}
	}
	return animals
}

func (f *activityFeedTestFixture) feedRequest(t *testing.T, query string) map[string]interface{} {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", f.user.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(f.group.ID)}}
	url := "/test"
	if query != "" {
		url += "?" + query
	}
	c.Request = httptest.NewRequest(http.MethodGet, url, nil)

	GetGroupActivityFeed(f.tx)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, w.Body.String())
	}
	return body
}

// TestActivityFeed_Postgres_CommentQueryIsActuallyBounded is the one test in
// this file that can't be fooled by the final Go-level pagination slice
// (activityItems[start:end]) alone - both the fixed and the original buggy
// handler return exactly `limit` items in the JSON response, since that
// slicing already existed before this fix. What changed is whether the SQL
// query feeding it was itself bounded. A sqlCapturingLogger records every
// statement GORM actually sends to Postgres during a real handler call, so
// this inspects the handler's real generated SQL - not a hand-written query
// that merely looks equivalent (which would only prove that hand-written
// query has a LIMIT, never touching the handler code at all).
func TestActivityFeed_Postgres_CommentQueryIsActuallyBounded(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newActivityFeedTestFixture(t, db)

	f.seedAnimalsWithComments(t, 26, 21, func(int) *models.SessionMetadata { return nil })

	capture := &sqlCapturingLogger{}
	loggedDB := f.tx.Session(&gorm.Session{Logger: capture})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", f.user.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(f.group.ID)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/test?limit=20", nil)

	GetGroupActivityFeed(loggedDB)(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var commentSelects []string
	for _, stmt := range capture.statements {
		// The comments page query specifically - not the animals lookup,
		// not the COUNT(*) query, not the lean metadata-only stats scan
		// (which is deliberately unbounded - see activity_feed.go).
		lower := strings.ToLower(stmt)
		if strings.Contains(lower, "select * from \"animal_comments\"") || strings.HasPrefix(lower, `select "animal_comments".*`) {
			commentSelects = append(commentSelects, stmt)
		}
	}

	if len(commentSelects) == 0 {
		t.Fatalf("no comments SELECT captured among %d statements - test setup may not be matching the real query shape:\n%s",
			len(capture.statements), strings.Join(capture.statements, "\n"))
	}
	for _, stmt := range commentSelects {
		if !strings.Contains(strings.ToUpper(stmt), "LIMIT") {
			t.Fatalf("expected the comments page query to contain LIMIT, got: %s", stmt)
		}
	}
}

func TestActivityFeed_Postgres_BoundsCommentFetchForLargeGroup(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newActivityFeedTestFixture(t, db)

	// Mirrors the traced incident: 26 animals, hundreds of comments total,
	// no rating filter (the default load every user hits on opening the tab).
	const numAnimals = 26
	const commentsPerAnimal = 21
	f.seedAnimalsWithComments(t, numAnimals, commentsPerAnimal, func(int) *models.SessionMetadata { return nil })
	totalComments := numAnimals * commentsPerAnimal

	body := f.feedRequest(t, "limit=20")

	items, _ := body["items"].([]interface{})
	if len(items) != 20 {
		t.Fatalf("expected exactly 20 items (the requested page size), got %d", len(items))
	}
	if got := int(body["total"].(float64)); got != totalComments {
		t.Fatalf("expected total=%d (the true count across the whole group, not just the fetched page), got %d", totalComments, got)
	}
	if body["hasMore"] != true {
		t.Fatalf("expected hasMore=true with %d comments and a page size of 20, got %v", totalComments, body["hasMore"])
	}
}

func TestActivityFeed_Postgres_PreservesChronologicalOrderAcrossDeepPagination(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newActivityFeedTestFixture(t, db)

	f.seedAnimalsWithComments(t, 10, 8, func(int) *models.SessionMetadata { return nil })

	page1 := f.feedRequest(t, "limit=20&offset=0")
	page2 := f.feedRequest(t, "limit=20&offset=20")
	page3 := f.feedRequest(t, "limit=20&offset=40")

	itemKey := func(raw interface{}) string {
		item := raw.(map[string]interface{})
		return fmt.Sprintf("%v-%v", item["type"], item["id"])
	}

	seen := map[string]bool{}
	var allTimestamps []time.Time
	for _, page := range []map[string]interface{}{page1, page2, page3} {
		items, _ := page["items"].([]interface{})
		for _, raw := range items {
			key := itemKey(raw)
			if seen[key] {
				t.Fatalf("item %s appeared on more than one page — pagination is not correctly bounding/ordering results", key)
			}
			seen[key] = true

			ts, err := time.Parse(time.RFC3339, raw.(map[string]interface{})["created_at"].(string))
			if err != nil {
				t.Fatalf("parse created_at: %v", err)
			}
			allTimestamps = append(allTimestamps, ts)
		}
	}

	for i := 1; i < len(allTimestamps); i++ {
		if allTimestamps[i].After(allTimestamps[i-1]) {
			t.Fatalf("items are not in strict newest-first order across pages: item %d (%v) is after item %d (%v)",
				i, allTimestamps[i], i-1, allTimestamps[i-1])
		}
	}
}

func TestActivityFeed_Postgres_SummaryReflectsFullDatasetNotJustFetchedPage(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newActivityFeedTestFixture(t, db)

	// More comments than any single page (limit=20 by default) so the
	// summary can only be right if it's computed over the full group, not
	// just whatever comments happened to be fetched for display.
	const numAnimals = 5
	const commentsPerAnimal = 12 // 60 total, well beyond the default page of 20
	f.seedAnimalsWithComments(t, numAnimals, commentsPerAnimal, func(seq int) *models.SessionMetadata {
		switch seq % 4 {
		case 0:
			return &models.SessionMetadata{SessionRating: 1, BehaviorNotes: "resource guarding at mealtime"}
		case 1:
			return &models.SessionMetadata{SessionRating: 5, MedicalNotes: "cleared for light exercise"}
		default:
			return nil
		}
	})
	total := numAnimals * commentsPerAnimal
	expectedBehaviorAndPoor := 0
	expectedMedical := 0
	for seq := 1; seq <= total; seq++ {
		switch seq % 4 {
		case 0:
			expectedBehaviorAndPoor++
		case 1:
			expectedMedical++
		}
	}

	body := f.feedRequest(t, "limit=20")

	items, _ := body["items"].([]interface{})
	if len(items) != 20 {
		t.Fatalf("expected the page itself to still be bounded to 20, got %d", len(items))
	}

	summary := body["summary"].(map[string]interface{})
	if got := int(summary["behavior_concerns_count"].(float64)); got != expectedBehaviorAndPoor {
		t.Fatalf("behavior_concerns_count: got %d, want %d (full-group count, not just the fetched page)", got, expectedBehaviorAndPoor)
	}
	if got := int(summary["medical_concerns_count"].(float64)); got != expectedMedical {
		t.Fatalf("medical_concerns_count: got %d, want %d", got, expectedMedical)
	}
	if got := int(summary["poor_sessions_count"].(float64)); got != expectedBehaviorAndPoor {
		t.Fatalf("poor_sessions_count: got %d, want %d", got, expectedBehaviorAndPoor)
	}
}

func TestActivityFeed_Postgres_RatingFilterMatchesOriginalSemantics(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newActivityFeedTestFixture(t, db)

	// seq%5 buckets: 0 -> poor (rating 1), 1 -> rating 5, 2 -> rating 3,
	// 3,4 -> no rating at all.
	const numAnimals = 4
	const commentsPerAnimal = 15 // 60 total, 12 per bucket
	f.seedAnimalsWithComments(t, numAnimals, commentsPerAnimal, func(seq int) *models.SessionMetadata {
		switch seq % 5 {
		case 0:
			return &models.SessionMetadata{SessionRating: 1}
		case 1:
			return &models.SessionMetadata{SessionRating: 5}
		case 2:
			return &models.SessionMetadata{SessionRating: 3}
		default:
			return nil
		}
	})
	total := numAnimals * commentsPerAnimal
	countBucket := func(pred func(seq int) bool) int {
		n := 0
		for seq := 1; seq <= total; seq++ {
			if pred(seq) {
				n++
			}
		}
		return n
	}
	ratedCount := countBucket(func(seq int) bool { return seq%5 == 0 || seq%5 == 1 || seq%5 == 2 })
	poorCount := countBucket(func(seq int) bool { return seq%5 == 0 })
	fiveCount := countBucket(func(seq int) bool { return seq%5 == 1 })

	t.Run("poor matches only rating 1-2", func(t *testing.T) {
		body := f.feedRequest(t, "type=comments&limit=200&rating=poor")
		if got := int(body["total"].(float64)); got != poorCount {
			t.Fatalf("rating=poor total: got %d, want %d", got, poorCount)
		}
	})

	t.Run("exact numeric rating matches only that value", func(t *testing.T) {
		body := f.feedRequest(t, "type=comments&limit=200&rating=5")
		if got := int(body["total"].(float64)); got != fiveCount {
			t.Fatalf("rating=5 total: got %d, want %d", got, fiveCount)
		}
	})

	t.Run("non-numeric non-poor value is a no-op beyond has-a-rating", func(t *testing.T) {
		// Matches the original handler's pre-fix behavior: an unparseable
		// rating value doesn't narrow further, but comments with no rating
		// at all are still excluded (see activity_feed.go's rating-filter
		// comment for the full reasoning).
		body := f.feedRequest(t, "type=comments&limit=200&rating=not-a-number")
		if got := int(body["total"].(float64)); got != ratedCount {
			t.Fatalf("rating=<garbage> total: got %d, want %d (every rated comment, un-narrowed)", got, ratedCount)
		}
	})
}

// TestActivityFeed_Postgres_HasIndexForCommentQuery guards against a future
// change accidentally dropping the migration this fix relies on: the
// pre-existing idx_comment_animal_created index is (created_at, animal_id) -
// created_at leading, which Postgres can't use efficiently for this
// handler's "WHERE animal_id IN (...)" filter. Without
// idx_animal_comments_animal_deleted_created (animal_id, deleted_at,
// created_at), the query this fix bounded with a real LIMIT would fall back
// to a sequential scan over the whole animal_comments table on every
// activity-feed load, reintroducing the latency this fix was written for.
func TestActivityFeed_Postgres_HasIndexForCommentQuery(t *testing.T) {
	db := openSearchTestPostgres(t)

	var count int64
	if err := db.Raw(`
		SELECT count(*) FROM pg_indexes
		WHERE tablename = 'animal_comments'
		AND indexname = 'idx_animal_comments_animal_deleted_created'
	`).Scan(&count).Error; err != nil {
		t.Fatalf("query pg_indexes: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected idx_animal_comments_animal_deleted_created to exist on animal_comments, found %d", count)
	}

	var indexDef string
	if err := db.Raw(`SELECT indexdef FROM pg_indexes WHERE indexname = 'idx_animal_comments_animal_deleted_created'`).
		Scan(&indexDef).Error; err != nil {
		t.Fatalf("query index definition: %v", err)
	}
	if !strings.Contains(indexDef, "animal_id") || !strings.Contains(indexDef, "deleted_at") || !strings.Contains(indexDef, "created_at") {
		t.Fatalf("expected index to cover animal_id, deleted_at, and created_at, got: %s", indexDef)
	}
}

// TestActivityFeed_Postgres_TotalCommentsCorrectWithTagFilterGroupBy closes
// the loop on a finding from an automated PR review: applyTagFilter (see
// animal_comment.go) adds a JOIN + GROUP BY animal_comments.id to dedupe
// multi-tag matches, and the claim was that GORM's Count() on a grouped
// query returns an arbitrary single group's row count rather than the true
// distinct total. Investigated and refuted (GORM's Count() explicitly uses
// RowsAffected - the number of rows/groups actually returned - whenever a
// GROUP BY clause is present, which is exactly "number of distinct
// groups"), including empirically against real Postgres. This test pins
// that down permanently: each seeded comment carries two tags (so the join
// produces two rows per comment before GROUP BY collapses them back down),
// which is precisely the shape that would surface a bug here if one
// existed.
func TestActivityFeed_Postgres_TotalCommentsCorrectWithTagFilterGroupBy(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newActivityFeedTestFixture(t, db)

	animal := models.Animal{GroupID: f.group.ID, Name: "Rex", Species: "Dog", Status: "available"}
	if err := f.tx.Create(&animal).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}

	unique := time.Now().UnixNano()
	tag1 := models.CommentTag{Name: fmt.Sprintf("behavior%d", unique), Color: "#FF0000"}
	tag2 := models.CommentTag{Name: fmt.Sprintf("medical%d", unique), Color: "#00FF00"}
	if err := f.tx.Create(&tag1).Error; err != nil {
		t.Fatalf("create tag1: %v", err)
	}
	if err := f.tx.Create(&tag2).Error; err != nil {
		t.Fatalf("create tag2: %v", err)
	}

	const numTaggedComments = 5
	for i := 0; i < numTaggedComments; i++ {
		comment := models.AnimalComment{
			AnimalID: animal.ID,
			UserID:   f.user.ID,
			Content:  fmt.Sprintf("tagged comment %d", i),
			Tags:     []models.CommentTag{tag1, tag2},
		}
		if err := f.tx.Create(&comment).Error; err != nil {
			t.Fatalf("create tagged comment %d: %v", i, err)
		}
	}
	// An untagged comment too, to confirm the filter is actually narrowing
	// results rather than everything happening to match.
	if err := f.tx.Create(&models.AnimalComment{AnimalID: animal.ID, UserID: f.user.ID, Content: "untagged comment"}).Error; err != nil {
		t.Fatalf("create untagged comment: %v", err)
	}

	body := f.feedRequest(t, fmt.Sprintf("type=comments&limit=2&tags=%s", tag1.Name))

	if got := int(body["total"].(float64)); got != numTaggedComments {
		t.Fatalf("total: got %d, want %d (the true distinct comment count under the tag filter's GROUP BY, not an arbitrary per-group value)", got, numTaggedComments)
	}
	items, _ := body["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("expected the page itself to still respect limit=2, got %d items", len(items))
	}
	if body["hasMore"] != true {
		t.Fatalf("expected hasMore=true (2 of %d tagged comments shown), got %v", numTaggedComments, body["hasMore"])
	}
}
