package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/embedding"
)

func TestSearch_ForbiddenWhenNoGroupAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := SetupTestDB(t)

	group := CreateTestGroup(t, db, "Dogs", "Dog group")
	CreateTestUser(t, db, "outsider", "outsider@example.com", "password123", false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", uint(999))
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(group.ID)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/test?q=guarding", nil)

	Search(db, &embedding.StubEmbedder{})(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestSearch_BadRequestWhenTypeInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := SetupTestDB(t)

	group := CreateTestGroup(t, db, "Dogs", "Dog group")
	user := CreateTestUser(t, db, "member", "member@example.com", "password123", false)
	AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", user.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(group.ID)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/test?q=guarding&type=bogus", nil)

	Search(db, &embedding.StubEmbedder{})(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSearch_TypeUpdatesIsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := SetupTestDB(t)

	group := CreateTestGroup(t, db, "Dogs", "Dog group")
	user := CreateTestUser(t, db, "member", "member@example.com", "password123", false)
	AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", user.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(group.ID)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/test?q=guarding&type=updates", nil)

	Search(db, &embedding.StubEmbedder{})(c)

	// This only exercises the request-validation layer (the validTypes map
	// in Search): SetupTestDB is SQLite, which can't execute the
	// Postgres-only search_vector/websearch_to_tsquery SQL the handler runs
	// past validation, so a DB-layer 500 here is expected and not itself a
	// failure — checking w.Code != 400 alone can't distinguish "updates
	// passed validation" from "updates was rejected as invalid," so assert
	// specifically against the type-validation error message. Full
	// end-to-end coverage of type=updates (real matches, real ranking) is
	// in search_postgres_test.go's TestSearch_Postgres_MatchesUpdatesByKeyword,
	// which runs against a real Postgres instance.
	if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "type must be one of") {
		t.Fatalf("expected type=updates to pass validation, got 400: %s", w.Body.String())
	}
}

func TestSearch_BadRequestWhenQueryMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := SetupTestDB(t)

	group := CreateTestGroup(t, db, "Dogs", "Dog group")
	user := CreateTestUser(t, db, "member", "member@example.com", "password123", false)
	AddUserToGroupWithAdmin(t, db, user.ID, group.ID, false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", user.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(group.ID)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	Search(db, &embedding.StubEmbedder{})(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
