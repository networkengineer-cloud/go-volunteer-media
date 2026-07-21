package handlers

import (
	"context"
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
	"github.com/pgvector/pgvector-go"
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

// setAnimalEmbedding directly UPDATEs an animal's embedding column with a
// hand-crafted vector, bypassing the real embedding pipeline so tests get
// deterministic, controllable nearest-neighbor behavior instead of depending
// on a real Voyage call or model output.
func (f *searchTestFixture) setAnimalEmbedding(t *testing.T, animalID uint, vec []float32) {
	t.Helper()
	if err := f.tx.Exec(
		"UPDATE animals SET embedding = ?, embedding_updated_at = now() WHERE id = ?",
		pgvector.NewVector(vec), animalID,
	).Error; err != nil {
		t.Fatalf("failed to set animal embedding: %v", err)
	}
}

func (f *searchTestFixture) setCommentEmbedding(t *testing.T, commentID uint, vec []float32) {
	t.Helper()
	if err := f.tx.Exec(
		"UPDATE animal_comments SET embedding = ?, embedding_updated_at = now() WHERE id = ?",
		pgvector.NewVector(vec), commentID,
	).Error; err != nil {
		t.Fatalf("failed to set comment embedding: %v", err)
	}
}

// vectorWithOneAt returns a Dimension-length vector that is 1.0 at the given
// index and 0 elsewhere — an orthogonal basis vector, so cosine distance
// between vectorWithOneAt(i) and vectorWithOneAt(j) is deterministic and
// maximal for i != j, and zero for i == i.
func vectorWithOneAt(index int) []float32 {
	v := make([]float32, embedding.Dimension)
	v[index] = 1.0
	return v
}

// fixedVectorEmbedder is an Embedder stub whose EmbedQuery always returns a
// fixed vector, for tests that need to control exactly what the "query
// embedding" is (rather than deriving it from the query text like
// StubEmbedder does), to make a specific semantic match deterministic.
type fixedVectorEmbedder struct {
	vector []float32
}

func (f *fixedVectorEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return f.vector, nil
}
func (f *fixedVectorEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = f.vector
	}
	return out, nil
}
func (f *fixedVectorEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return f.vector, nil
}
func (f *fixedVectorEmbedder) IsConfigured() bool {
	return true
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

func TestSearch_Postgres_SemanticMatchSurfacesResultWithNoKeywordOverlap(t *testing.T) {
	// SEMANTIC_SEARCH_ENABLED is opt-in (defaults to disabled) — this test
	// specifically needs semantic search active to exercise RRF fusion.
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "true")

	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	// "Gets snappy near his bowl" shares zero keyword-search vocabulary with
	// "resource guarding" — a pure keyword search would miss it. Its
	// embedding is set to be identical to the query embedding.
	semanticOnly := models.Animal{
		GroupID: f.groupA.ID, Name: "Bowl", Species: "Dog", Status: "available",
		Description: "Gets snappy near his bowl during mealtime.",
	}
	if err := f.tx.Create(&semanticOnly).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}
	f.setAnimalEmbedding(t, semanticOnly.ID, vectorWithOneAt(0))

	noOverlap := models.Animal{GroupID: f.groupA.ID, Name: "Unrelated", Species: "Cat", Status: "available"}
	if err := f.tx.Create(&noOverlap).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}
	f.setAnimalEmbedding(t, noOverlap.ID, vectorWithOneAt(500)) // orthogonal, far away

	// fixedVectorEmbedder (defined above) forces the query embedding to
	// exactly match semanticOnly's stored vector, making the semantic match
	// deterministic without a real Voyage call.
	c, w := f.searchRequest(t, f.groupA.ID, "resource guarding")
	Search(f.tx, &fixedVectorEmbedder{vector: vectorWithOneAt(0)})(c)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeSearchResponse(t, w)
	animals, _ := body["animals"].([]interface{})
	found := false
	for _, a := range animals {
		if a.(map[string]interface{})["name"] == "Bowl" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected semantic match \"Bowl\" to surface despite no keyword overlap, got animals: %v", animals)
	}

	// Regression check: a keyword-only Count() would report 0 here (neither
	// animal matches "resource guarding" by keyword), even though a semantic
	// match is genuinely present in the results above. total_animals must
	// reflect the fused result set, not just the keyword count, or the
	// frontend's canLoadMore (animals.length < totalAnimals) can never fire
	// for a page made entirely of semantic-only matches.
	if body["total_animals"].(float64) < 1 {
		t.Fatalf("expected total_animals to count the semantic-only match, got %v", body["total_animals"])
	}
}

func TestSearch_Postgres_DegradesToKeywordOnlyWhenEmbedderFails(t *testing.T) {
	// SEMANTIC_SEARCH_ENABLED is opt-in (defaults to disabled) — must be
	// explicitly enabled here, or Usable(embedder) would short-circuit on
	// the flag alone and this test would never actually reach (and
	// exercise) failingEmbedder's EmbedQuery failure below.
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "true")

	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	match := models.Animal{
		GroupID: f.groupA.ID, Name: "Rex", Species: "Dog", Status: "available",
		Description: "Shows resource guarding around food bowls.",
	}
	if err := f.tx.Create(&match).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}

	failingEmbedder := &embedding.StubEmbedder{Err: fmt.Errorf("simulated Voyage outage")}
	c, w := f.searchRequest(t, f.groupA.ID, "resource guarding")
	Search(f.tx, failingEmbedder)(c)

	assert.Equal(t, http.StatusOK, w.Code, "a failed query embedding must degrade to keyword-only results, not fail the request")
	body := decodeSearchResponse(t, w)
	animals, _ := body["animals"].([]interface{})
	if len(animals) != 1 || animals[0].(map[string]interface{})["name"] != "Rex" {
		t.Fatalf("expected keyword-only match for Rex despite embedder failure, got: %v", animals)
	}
}

// TestSearch_Postgres_DegradesToKeywordOnlyWhenSemanticQueryFails covers a
// different failure point than the test above: the query embedding itself
// succeeds (semanticAvailable is true), but the per-resource semantic SELECT
// against the embedding column fails — simulated here via a dimension
// mismatch (the query vector is a different size than the stored one),
// which pgvector rejects at execution time. Before the fix, this branch
// fused the already-fetched, pool-limited keywordRows (no real Offset
// applied) instead of falling back to a genuine keyword-only page, which
// silently broke pagination once offset+limit exceeded the candidate pool.
//
// This test deliberately does NOT use the shared f.tx fixture the rest of
// this file uses: once a statement inside an explicit Postgres transaction
// errors, the *whole transaction* is aborted and every later statement on
// it fails too ("current transaction is aborted"), which would make the
// fixed code's keyword-only re-query fail regardless of correctness — an
// artifact of the transaction-per-test isolation technique, not of
// production behavior. Production never wraps a request in an explicit
// transaction (middleware.GetDB only does WithContext, never Begin), so
// this test commits real rows and cleans them up manually instead.
func TestSearch_Postgres_DegradesToKeywordOnlyWhenSemanticQueryFails(t *testing.T) {
	// SEMANTIC_SEARCH_ENABLED is opt-in (defaults to disabled) — must be
	// explicitly enabled here, or Usable(embedder) would short-circuit on
	// the flag alone and this test would never reach the per-resource
	// semantic query it's meant to fail.
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "true")

	db := openSearchTestPostgres(t)

	unique := time.Now().UnixNano()
	group := models.Group{Name: fmt.Sprintf("SearchTest-SemFail-%d", unique)}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	user := models.User{
		Username: fmt.Sprintf("searchtest-semfail-%d", unique),
		Email:    fmt.Sprintf("searchtest-semfail-%d@example.com", unique),
		Password: "x",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.UserGroup{UserID: user.ID, GroupID: group.ID}).Error; err != nil {
		t.Fatalf("add user to group: %v", err)
	}
	match := models.Animal{
		GroupID: group.ID, Name: "Rex", Species: "Dog", Status: "available",
		Description: "Shows resource guarding around food bowls.",
	}
	if err := db.Create(&match).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}
	t.Cleanup(func() {
		db.Unscoped().Delete(&match)
		db.Unscoped().Where("user_id = ? AND group_id = ?", user.ID, group.ID).Delete(&models.UserGroup{})
		db.Unscoped().Delete(&user)
		db.Unscoped().Delete(&group)
	})

	// A real, correctly-sized embedding so the semantic query has at least
	// one row to actually evaluate `embedding <=> ?` against — otherwise a
	// dimension-mismatched query vector could return zero rows without
	// Postgres ever raising the mismatch error.
	if err := db.Exec(
		"UPDATE animals SET embedding = ?, embedding_updated_at = now() WHERE id = ?",
		pgvector.NewVector(vectorWithOneAt(0)), match.ID,
	).Error; err != nil {
		t.Fatalf("failed to set animal embedding: %v", err)
	}

	// EmbedQuery succeeds (unlike the test above), but returns a vector of
	// the wrong dimension, so the per-resource semantic query fails at the
	// database level rather than at the embedding-call level.
	wrongDimEmbedder := &fixedVectorEmbedder{vector: make([]float32, embedding.Dimension/2)}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", user.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(group.ID)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/test?"+url.Values{"q": {"resource guarding"}}.Encode(), nil)

	Search(db, wrongDimEmbedder)(c)

	assert.Equal(t, http.StatusOK, w.Code, "a failed semantic query must degrade to keyword-only results, not fail the request: %s", w.Body.String())
	body := decodeSearchResponse(t, w)
	animals, _ := body["animals"].([]interface{})
	if len(animals) != 1 || animals[0].(map[string]interface{})["name"] != "Rex" {
		t.Fatalf("expected keyword-only match for Rex despite semantic query failure, got: %v", animals)
	}
	if total, _ := body["total_animals"].(float64); total != 1 {
		t.Fatalf("expected total_animals=1 (real keyword count), got %v", body["total_animals"])
	}
}

func TestSearch_Postgres_KeywordOnlyModePreservesRealTsRankAndUnboundedPagination(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	// Distinct term-frequency gives distinctly different (not tied) ts_rank
	// values, so a position-only RRF score (which would only ever be
	// 1/(60+1), 1/(60+2), ...) is distinguishable from the real ts_rank.
	low := models.Animal{
		GroupID: f.groupA.ID, Name: "LowRank", Species: "Dog", Status: "available",
		Description: "Occasional resource guarding noted around mealtime.",
	}
	high := models.Animal{
		GroupID: f.groupA.ID, Name: "HighRank", Species: "Dog", Status: "available",
		Description:  "Severe resource guarding. Resource guarding around food, toys, and beds.",
		TrainerNotes: "Resource guarding resource guarding — handle with extreme care.",
	}
	for _, a := range []*models.Animal{&low, &high} {
		if err := f.tx.Create(a).Error; err != nil {
			t.Fatalf("create animal %s: %v", a.Name, err)
		}
	}

	// SEMANTIC_SEARCH_ENABLED=false forces the keyword-only path regardless
	// of the embedder passed in, exercising the "flag off" degrade case
	// specifically (as opposed to "embedder call failed").
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "false")

	c, w := f.searchRequest(t, f.groupA.ID, "resource guarding")
	Search(f.tx, &embedding.StubEmbedder{})(c)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeSearchResponse(t, w)
	animals, _ := body["animals"].([]interface{})
	if len(animals) != 2 {
		t.Fatalf("expected 2 matching animals, got %d: %v", len(animals), animals)
	}

	ranksByName := map[string]float64{}
	for _, a := range animals {
		row := a.(map[string]interface{})
		ranksByName[row["name"].(string)] = row["rank"].(float64)
	}

	// A position-only RRF score for a 2-row keyword-only list would be
	// 1/(60+1) ≈ 0.01639 and 1/(60+2) ≈ 0.01613 — close together regardless
	// of actual relevance. The real ts_rank values for these two animals
	// differ far more sharply (HighRank repeats the term many more times),
	// so asserting the gap is larger than the RRF position-score gap proves
	// the real ts_rank survived, not a position-derived substitute.
	rrfPositionGap := 1.0/61.0 - 1.0/62.0
	actualGap := ranksByName["HighRank"] - ranksByName["LowRank"]
	if actualGap <= rrfPositionGap {
		t.Fatalf("expected real ts_rank values (gap should exceed the RRF position-score gap of %v), got HighRank=%v LowRank=%v (gap=%v) — looks like a position-only score, not real ts_rank",
			rrfPositionGap, ranksByName["HighRank"], ranksByName["LowRank"], actualGap)
	}
}

func TestEmbedAnimalNow_Postgres_SkipsStaleWriteAfterConcurrentEdit(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	animal := models.Animal{GroupID: f.groupA.ID, Name: "Rex", Species: "Dog", Status: "available"}
	if err := f.tx.Create(&animal).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}

	// Capture the row's state as of creation — this simulates what a
	// write-path embed goroutine captures before its (slow) embed API call.
	staleAnimal := animal

	// Simulate a second edit landing before that goroutine's embed call
	// completes: bump updated_at by saving again.
	if err := f.tx.Model(&animal).Update("name", "Rex Updated").Error; err != nil {
		t.Fatalf("update animal: %v", err)
	}

	// Embed using the STALE captured copy (older updated_at) — this must be
	// silently discarded rather than overwrite the row, since a newer edit
	// has landed since this text was captured.
	if err := embedAnimalNow(f.tx, &embedding.StubEmbedder{}, staleAnimal); err != nil {
		t.Fatalf("embedAnimalNow returned an unexpected error: %v", err)
	}

	var embeddingIsNull bool
	if err := f.tx.Raw("SELECT embedding IS NULL FROM animals WHERE id = ?", animal.ID).Scan(&embeddingIsNull).Error; err != nil {
		t.Fatalf("query embedding: %v", err)
	}
	if !embeddingIsNull {
		t.Fatal("expected the stale embed write to be discarded (embedding should still be NULL) since a newer edit landed after the embedded text was captured, but it was persisted anyway")
	}
}

// TestEmbedAnimalNow_Postgres_PersistsWhenNoConcurrentEdit complements
// TestEmbedAnimalNow_Postgres_SkipsStaleWriteAfterConcurrentEdit's reject
// path: it proves the "AND updated_at = ?" guard isn't so strict it also
// discards the normal, non-conflicting case where nothing else has touched
// the row since the embed text was captured.
func TestEmbedAnimalNow_Postgres_PersistsWhenNoConcurrentEdit(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	animal := models.Animal{GroupID: f.groupA.ID, Name: "Rex", Species: "Dog", Status: "available"}
	if err := f.tx.Create(&animal).Error; err != nil {
		t.Fatalf("create animal: %v", err)
	}

	if err := embedAnimalNow(f.tx, &embedding.StubEmbedder{}, animal); err != nil {
		t.Fatalf("embedAnimalNow returned an unexpected error: %v", err)
	}

	var embeddingIsNull bool
	if err := f.tx.Raw("SELECT embedding IS NULL FROM animals WHERE id = ?", animal.ID).Scan(&embeddingIsNull).Error; err != nil {
		t.Fatalf("query embedding: %v", err)
	}
	if embeddingIsNull {
		t.Fatal("expected the embed write to persist since no concurrent edit landed, but embedding is still NULL")
	}
}

func TestSearch_Postgres_MatchesUpdatesByKeyword(t *testing.T) {
	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	update := models.Update{
		GroupID: f.groupA.ID, UserID: f.user.ID,
		Title: "Playgroup Saturday", Content: "Pairings: Rex+Fido, Bella+Max. 10am at the field.",
	}
	if err := f.tx.Create(&update).Error; err != nil {
		t.Fatalf("create update: %v", err)
	}
	other := models.Update{
		GroupID: f.groupA.ID, UserID: f.user.ID,
		Title: "Unrelated", Content: "Reminder to restock supplies.",
	}
	if err := f.tx.Create(&other).Error; err != nil {
		t.Fatalf("create other update: %v", err)
	}

	c, w := f.searchRequestWithParams(t, f.groupA.ID, url.Values{"q": {"pairings"}, "type": {"updates"}})
	Search(f.tx, &embedding.StubEmbedder{})(c)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeSearchResponse(t, w)
	assert.Equal(t, float64(1), body["total_updates"])
	updates, _ := body["updates"].([]interface{})
	if len(updates) != 1 || updates[0].(map[string]interface{})["title"] != "Playgroup Saturday" {
		t.Fatalf("expected exactly the matching update, got: %v", updates)
	}
	if _, present := body["animals"]; present {
		t.Fatalf("expected no 'animals' key when type=updates, got %v", body["animals"])
	}
}

func TestSearch_Postgres_DeepPaginationCoversResultsBeyondDefaultPool(t *testing.T) {
	// SEMANTIC_SEARCH_ENABLED is opt-in (defaults to disabled) — this test
	// specifically exercises candidatePoolSize's growth, which only matters
	// on the semantic/RRF-fusion path (applyPageOrPool ignores pool
	// entirely once semanticAvailable is false), so it must be enabled here.
	t.Setenv("SEMANTIC_SEARCH_ENABLED", "true")

	db := openSearchTestPostgres(t)
	f := newSearchTestFixture(t, db)

	// 60 matching animals — more than the 50-row default candidate pool
	// floor — to prove candidatePoolSize actually grows for a deep offset.
	for i := 0; i < 60; i++ {
		a := models.Animal{
			GroupID: f.groupA.ID, Name: fmt.Sprintf("Animal%02d", i), Species: "Dog", Status: "available",
			Description: "Loves the playgroup.",
		}
		if err := f.tx.Create(&a).Error; err != nil {
			t.Fatalf("create animal %d: %v", i, err)
		}
	}

	c, w := f.searchRequestWithParams(t, f.groupA.ID, url.Values{"q": {"playgroup"}, "limit": {"10"}, "offset": {"55"}})
	Search(f.tx, &embedding.StubEmbedder{})(c)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeSearchResponse(t, w)
	assert.Equal(t, float64(60), body["total_animals"])
	animals, _ := body["animals"].([]interface{})
	if len(animals) != 5 {
		t.Fatalf("expected 5 animals on the last partial page (60 total, offset=55, limit=10), got %d: %v", len(animals), animals)
	}
}
