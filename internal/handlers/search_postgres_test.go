package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/embedding"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// These tests exercise the Postgres-specific full-text search behavior
// (tsvector/websearch_to_tsquery/GIN) that SQLite — used by every other test
// in this package — can't run. They connect to a real Postgres instance
// using the same DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_SSLMODE env vars the
// app itself reads (see internal/database.Initialize), defaulting to a
// dedicated "volunteer_media_test" database so a developer's real dev DB is
// never touched by default. If nothing is listening, the whole suite skips —
// `go test ./...` stays green without a database, matching how the e2e job
// (.github/workflows/test.yml) is the only place in CI that provisions
// Postgres. To run these locally: `docker compose up -d postgres_dev`, then
// `DB_NAME=volunteer_media_dev go test ./internal/handlers/... -run Postgres -v`
// (or create/point at a scratch "volunteer_media_test" database).
func openSearchTestPostgres(t *testing.T) *gorm.DB {
	t.Helper()

	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")

	if !tcpReachable(host, port, 2*time.Second) {
		t.Skipf("skipping: no Postgres reachable at %s:%s (run `docker compose up -d postgres_dev` to enable this test)", host, port)
	}

	user := envOrDefault("DB_USER", "postgres")
	password := envOrDefault("DB_PASSWORD", "postgres")
	dbname := envOrDefault("DB_NAME", "volunteer_media_test")
	sslmode := envOrDefault("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
		host, port, user, password, dbname, sslmode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Skipf("skipping: could not connect to postgres database %q (%v) — create it or set DB_NAME to an existing scratch database", dbname, err)
	}

	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations against test postgres: %v", err)
	}

	return db
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// tcpReachable is a fast pre-check so the common "no Postgres running"
// case skips near-instantly (connection refused) instead of waiting out
// the full Postgres client connect_timeout on every `go test ./...` run.
func tcpReachable(host, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// searchTestFixture holds transaction-scoped test data. The transaction is
// always rolled back (registered via t.Cleanup), so these tests never leave
// data behind even when pointed at a real, shared database.
type searchTestFixture struct {
	tx     *gorm.DB
	groupA models.Group
	groupB models.Group
	user   models.User
}

func newSearchTestFixture(t *testing.T, db *gorm.DB) *searchTestFixture {
	t.Helper()
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })

	unique := time.Now().UnixNano()

	groupA := models.Group{Name: fmt.Sprintf("SearchTest-A-%d", unique)}
	if err := tx.Create(&groupA).Error; err != nil {
		t.Fatalf("create groupA: %v", err)
	}
	groupB := models.Group{Name: fmt.Sprintf("SearchTest-B-%d", unique)}
	if err := tx.Create(&groupB).Error; err != nil {
		t.Fatalf("create groupB: %v", err)
	}

	user := models.User{
		Username: fmt.Sprintf("searchtest-%d", unique),
		Email:    fmt.Sprintf("searchtest-%d@example.com", unique),
		Password: "x",
	}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := tx.Create(&models.UserGroup{UserID: user.ID, GroupID: groupA.ID}).Error; err != nil {
		t.Fatalf("add user to groupA: %v", err)
	}
	if err := tx.Create(&models.UserGroup{UserID: user.ID, GroupID: groupB.ID}).Error; err != nil {
		t.Fatalf("add user to groupB: %v", err)
	}

	return &searchTestFixture{tx: tx, groupA: groupA, groupB: groupB, user: user}
}

func (f *searchTestFixture) searchRequest(t *testing.T, groupID uint, query string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	return f.searchRequestWithParams(t, groupID, url.Values{"q": {query}})
}

// searchRequestWithParams builds a request with arbitrary query params (q,
// type, limit, offset, ...), for tests that need more than the default
// type=all search.
func (f *searchTestFixture) searchRequestWithParams(t *testing.T, groupID uint, params url.Values) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", f.user.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(groupID)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/test?"+params.Encode(), nil)
	return c, w
}

func decodeSearchResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, w.Body.String())
	}
	return body
}

func TestSearch_Postgres_MatchReturnsExpectedResults(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	match := models.Animal{
		GroupID:     f.groupA.ID,
		Name:        "Rex",
		Species:     "Dog",
		Description: "Shows resource guarding around food bowls.",
		Status:      "available",
	}
	if err := f.tx.Create(&match).Error; err != nil {
		t.Fatalf("create matching animal: %v", err)
	}
	nonMatch := models.Animal{GroupID: f.groupA.ID, Name: "Fido", Species: "Dog", Status: "available"}
	if err := f.tx.Create(&nonMatch).Error; err != nil {
		t.Fatalf("create non-matching animal: %v", err)
	}

	c, w := f.searchRequest(t, f.groupA.ID, "resource guarding")
	Search(f.tx, &embedding.StubEmbedder{})(c)

	assert.Equal(t, http.StatusOK, w.Code)

	body := decodeSearchResponse(t, w)
	assert.Equal(t, float64(1), body["total_animals"], "total_animals must reflect the match count")
	animals, _ := body["animals"].([]interface{})
	if len(animals) != 1 {
		t.Fatalf("expected exactly 1 matching animal, got %d: %v", len(animals), animals)
	}
	got := animals[0].(map[string]interface{})
	if got["name"] != "Rex" {
		t.Fatalf("expected match to be Rex, got %v", got["name"])
	}
}

func TestSearch_Postgres_ExcludesCommentsOnDeletedAnimal(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	keptAnimal := models.Animal{GroupID: f.groupA.ID, Name: "Buddy", Species: "Dog", Status: "available"}
	if err := f.tx.Create(&keptAnimal).Error; err != nil {
		t.Fatalf("create kept animal: %v", err)
	}
	deletedAnimal := models.Animal{GroupID: f.groupA.ID, Name: "Ghost", Species: "Dog", Status: "available"}
	if err := f.tx.Create(&deletedAnimal).Error; err != nil {
		t.Fatalf("create animal to be deleted: %v", err)
	}

	if err := f.tx.Create(&models.AnimalComment{
		AnimalID: keptAnimal.ID,
		UserID:   f.user.ID,
		Content:  "Great playgroup session today, no issues.",
	}).Error; err != nil {
		t.Fatalf("create comment on kept animal: %v", err)
	}
	if err := f.tx.Create(&models.AnimalComment{
		AnimalID: deletedAnimal.ID,
		UserID:   f.user.ID,
		Content:  "Great playgroup session today, no issues either.",
	}).Error; err != nil {
		t.Fatalf("create comment on animal to be deleted: %v", err)
	}

	// Soft-delete, exactly as DeleteAnimal does.
	if err := f.tx.Delete(&deletedAnimal).Error; err != nil {
		t.Fatalf("soft-delete animal: %v", err)
	}

	c, w := f.searchRequest(t, f.groupA.ID, "playgroup")
	Search(f.tx, &embedding.StubEmbedder{})(c)

	assert.Equal(t, http.StatusOK, w.Code)

	body := decodeSearchResponse(t, w)
	assert.Equal(t, float64(1), body["total_comments"], "total_comments must exclude the deleted animal's comment, not just the returned page")
	comments, _ := body["comments"].([]interface{})
	if len(comments) != 1 {
		t.Fatalf("expected exactly 1 comment (deleted animal's comment excluded), got %d: %v", len(comments), comments)
	}
	got := comments[0].(map[string]interface{})
	if got["animal_name"] != "Buddy" {
		t.Fatalf("expected surviving comment to belong to Buddy, got %v", got["animal_name"])
	}
}

func TestSearch_Postgres_DoesNotLeakAcrossGroups(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	inGroupA := models.Animal{GroupID: f.groupA.ID, Name: "Rex", Species: "Dog", Description: "Loves the playgroup.", Status: "available"}
	if err := f.tx.Create(&inGroupA).Error; err != nil {
		t.Fatalf("create animal in group A: %v", err)
	}
	inGroupB := models.Animal{GroupID: f.groupB.ID, Name: "Max", Species: "Dog", Description: "Loves the playgroup.", Status: "available"}
	if err := f.tx.Create(&inGroupB).Error; err != nil {
		t.Fatalf("create animal in group B: %v", err)
	}

	c, w := f.searchRequest(t, f.groupA.ID, "playgroup")
	Search(f.tx, &embedding.StubEmbedder{})(c)

	assert.Equal(t, http.StatusOK, w.Code)

	body := decodeSearchResponse(t, w)
	assert.Equal(t, float64(1), body["total_animals"], "total_animals must not count group B's matching animal")
	animals, _ := body["animals"].([]interface{})
	if len(animals) != 1 {
		t.Fatalf("expected exactly 1 animal (only group A's), got %d: %v", len(animals), animals)
	}
	got := animals[0].(map[string]interface{})
	if got["name"] != "Rex" {
		t.Fatalf("expected group A's Rex only — group B's Max must not leak into group A's search, got %v", got["name"])
	}
}

func TestSearch_Postgres_TypeFilterScopesToRequestedResource(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	matchingAnimal := models.Animal{GroupID: f.groupA.ID, Name: "Rex", Species: "Dog", Description: "Loves the playgroup.", Status: "available"}
	if err := f.tx.Create(&matchingAnimal).Error; err != nil {
		t.Fatalf("create matching animal: %v", err)
	}
	otherAnimal := models.Animal{GroupID: f.groupA.ID, Name: "Fido", Species: "Dog", Status: "available"}
	if err := f.tx.Create(&otherAnimal).Error; err != nil {
		t.Fatalf("create other animal: %v", err)
	}
	if err := f.tx.Create(&models.AnimalComment{
		AnimalID: otherAnimal.ID,
		UserID:   f.user.ID,
		Content:  "Fido had a great playgroup session today.",
	}).Error; err != nil {
		t.Fatalf("create matching comment: %v", err)
	}

	t.Run("type=animals returns only animals, omits comments entirely", func(t *testing.T) {
		c, w := f.searchRequestWithParams(t, f.groupA.ID, url.Values{"q": {"playgroup"}, "type": {"animals"}})
		Search(f.tx, &embedding.StubEmbedder{})(c)

		assert.Equal(t, http.StatusOK, w.Code)
		body := decodeSearchResponse(t, w)

		assert.Equal(t, float64(1), body["total_animals"])
		animals, _ := body["animals"].([]interface{})
		if len(animals) != 1 {
			t.Fatalf("expected exactly 1 animal, got %d: %v", len(animals), animals)
		}
		if _, present := body["comments"]; present {
			t.Fatalf("expected no 'comments' key at all when type=animals, got %v", body["comments"])
		}
		if _, present := body["total_comments"]; present {
			t.Fatalf("expected no 'total_comments' key at all when type=animals, got %v", body["total_comments"])
		}
	})

	t.Run("type=comments returns only comments, omits animals entirely", func(t *testing.T) {
		c, w := f.searchRequestWithParams(t, f.groupA.ID, url.Values{"q": {"playgroup"}, "type": {"comments"}})
		Search(f.tx, &embedding.StubEmbedder{})(c)

		assert.Equal(t, http.StatusOK, w.Code)
		body := decodeSearchResponse(t, w)

		assert.Equal(t, float64(1), body["total_comments"])
		comments, _ := body["comments"].([]interface{})
		if len(comments) != 1 {
			t.Fatalf("expected exactly 1 comment, got %d: %v", len(comments), comments)
		}
		if _, present := body["animals"]; present {
			t.Fatalf("expected no 'animals' key at all when type=comments, got %v", body["animals"])
		}
		if _, present := body["total_animals"]; present {
			t.Fatalf("expected no 'total_animals' key at all when type=comments, got %v", body["total_animals"])
		}
	})
}

func TestSearch_Postgres_PaginatesWithLimitAndOffset(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	for _, name := range []string{"Alpha", "Bravo", "Charlie"} {
		a := models.Animal{GroupID: f.groupA.ID, Name: name, Species: "Dog", Description: "Loves the playgroup.", Status: "available"}
		if err := f.tx.Create(&a).Error; err != nil {
			t.Fatalf("create animal %s: %v", name, err)
		}
	}

	c, w := f.searchRequestWithParams(t, f.groupA.ID, url.Values{"q": {"playgroup"}, "limit": {"2"}, "offset": {"0"}})
	Search(f.tx, &embedding.StubEmbedder{})(c)
	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeSearchResponse(t, w)
	assert.Equal(t, float64(3), body["total_animals"], "total_animals must reflect the full match count, not the page size")
	firstPage, _ := body["animals"].([]interface{})
	if len(firstPage) != 2 {
		t.Fatalf("expected 2 animals on first page (limit=2, offset=0), got %d: %v", len(firstPage), firstPage)
	}

	c2, w2 := f.searchRequestWithParams(t, f.groupA.ID, url.Values{"q": {"playgroup"}, "limit": {"2"}, "offset": {"2"}})
	Search(f.tx, &embedding.StubEmbedder{})(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	body2 := decodeSearchResponse(t, w2)
	assert.Equal(t, float64(3), body2["total_animals"], "total_animals on the second page must still reflect the full match count")
	secondPage, _ := body2["animals"].([]interface{})
	if len(secondPage) != 1 {
		t.Fatalf("expected 1 animal on second page (limit=2, offset=2), got %d: %v", len(secondPage), secondPage)
	}

	// Ranks may legitimately tie here (all three descriptions are identical),
	// so we don't assert which specific animals land on which page — only
	// that limit/offset correctly partition the 3 matches with no overlap.
	seenOnFirstPage := map[string]bool{}
	for _, a := range firstPage {
		seenOnFirstPage[a.(map[string]interface{})["name"].(string)] = true
	}
	for _, a := range secondPage {
		name := a.(map[string]interface{})["name"].(string)
		if seenOnFirstPage[name] {
			t.Fatalf("animal %q appeared on both pages — limit/offset are not partitioning results correctly", name)
		}
	}
}

func TestSearch_Postgres_RanksMultipleMatchesByRelevance(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	// Distinct term-frequency of "resource guarding" in each animal's
	// concatenated tsvector source gives ts_rank distinctly different (not
	// tied) scores, so the resulting order is deterministic.
	low := models.Animal{
		GroupID: f.groupA.ID, Name: "LowRank", Species: "Dog", Status: "available",
		Description: "Occasional resource guarding noted around mealtime.",
	}
	mid := models.Animal{
		GroupID: f.groupA.ID, Name: "MidRank", Species: "Dog", Status: "available",
		Description:  "Resource guarding around food.",
		TrainerNotes: "Watch for resource guarding during group play.",
	}
	high := models.Animal{
		GroupID: f.groupA.ID, Name: "HighRank", Species: "Dog", Status: "available",
		Description:  "Severe resource guarding. Resource guarding around food, toys, and beds.",
		TrainerNotes: "Resource guarding resource guarding — handle with extreme care.",
	}
	for _, a := range []*models.Animal{&low, &mid, &high} {
		if err := f.tx.Create(a).Error; err != nil {
			t.Fatalf("create animal %s: %v", a.Name, err)
		}
	}

	c, w := f.searchRequest(t, f.groupA.ID, "resource guarding")
	Search(f.tx, &embedding.StubEmbedder{})(c)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeSearchResponse(t, w)
	assert.Equal(t, float64(3), body["total_animals"])

	animals, _ := body["animals"].([]interface{})
	if len(animals) != 3 {
		t.Fatalf("expected 3 matching animals, got %d: %v", len(animals), animals)
	}
	names := make([]string, len(animals))
	ranks := make([]float64, len(animals))
	for i, a := range animals {
		row := a.(map[string]interface{})
		names[i] = row["name"].(string)
		ranks[i] = row["rank"].(float64)
	}

	if names[0] != "HighRank" || names[1] != "MidRank" || names[2] != "LowRank" {
		t.Fatalf("expected rank order [HighRank, MidRank, LowRank], got %v (ranks: %v)", names, ranks)
	}
	if !(ranks[0] > ranks[1] && ranks[1] > ranks[2]) {
		t.Fatalf("expected strictly decreasing ranks matching the returned order, got %v for %v", ranks, names)
	}
}
