package ratelimit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/middleware"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.RateLimitCounter{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// fakeClock is a settable clock for the limiter.
type fakeClock struct{ t time.Time }

func (f *fakeClock) now() time.Time { return f.t }

func newTestLimiter(t *testing.T, db *gorm.DB) (*Limiter, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 9, 28, 12, 0, 10, 0, time.UTC)}
	l := New(db)
	l.now = clock.now
	return l, clock
}

func mustHit(t *testing.T, l *Limiter, rule Rule, subject string) (bool, time.Duration) {
	t.Helper()
	ok, wait, err := l.Hit(context.Background(), rule, subject)
	if err != nil {
		t.Fatalf("Hit: %v", err)
	}
	return ok, wait
}

func TestHit_LimitWithinWindowThenReset(t *testing.T) {
	l, clock := newTestLimiter(t, openSQLite(t))
	rule := Rule{Name: "test", Limit: 3, Window: time.Minute}

	for i := 1; i <= 3; i++ {
		if ok, _ := mustHit(t, l, rule, "alice"); !ok {
			t.Fatalf("hit %d should be allowed", i)
		}
	}
	ok, wait := mustHit(t, l, rule, "alice")
	if ok {
		t.Fatal("4th hit in the window should be limited")
	}
	// Clock is 10s into the minute, so the window resets in 50s.
	if wait != 50*time.Second {
		t.Errorf("retryAfter = %s, want 50s", wait)
	}

	clock.t = clock.t.Add(time.Minute)
	if ok, _ := mustHit(t, l, rule, "alice"); !ok {
		t.Fatal("first hit in the next window should be allowed")
	}
}

func TestHit_SubjectsAndRulesAreIndependent(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))
	a := Rule{Name: "a", Limit: 1, Window: time.Minute}
	b := Rule{Name: "b", Limit: 1, Window: time.Minute}

	mustHit(t, l, a, "alice")
	if ok, _ := mustHit(t, l, a, "bob"); !ok {
		t.Error("another subject under the same rule has its own budget")
	}
	if ok, _ := mustHit(t, l, b, "alice"); !ok {
		t.Error("the same subject under another rule has its own budget")
	}
	if ok, _ := mustHit(t, l, a, "alice"); ok {
		t.Error("alice's second hit under rule a should be limited")
	}
}

func TestHit_StaleWindowFromSkewedClockDoesNotReset(t *testing.T) {
	l, clock := newTestLimiter(t, openSQLite(t))
	rule := Rule{Name: "skew", Limit: 2, Window: time.Minute}

	mustHit(t, l, rule, "alice")
	mustHit(t, l, rule, "alice")
	// A replica whose clock is a minute behind reports the previous window;
	// that must not reset the counter.
	clock.t = clock.t.Add(-time.Minute)
	if ok, _ := mustHit(t, l, rule, "alice"); ok {
		t.Fatal("a hit stamped with an older window must count against the current one")
	}
}

func TestHit_KeysAreHashed(t *testing.T) {
	db := openSQLite(t)
	l, _ := newTestLimiter(t, db)
	mustHit(t, l, Rule{Name: "login:user_ip", Limit: 5, Window: time.Minute}, "alice|203.0.113.7")

	var rows []models.RateLimitCounter
	db.Find(&rows)
	if len(rows) != 1 || len(rows[0].Key) != 64 || bytes.Contains([]byte(rows[0].Key), []byte("alice")) {
		t.Fatalf("expected one row keyed by a 64-char hash, got %+v", rows)
	}
}

func TestSweep(t *testing.T) {
	db := openSQLite(t)
	l, clock := newTestLimiter(t, db)
	rule := Rule{Name: "sweep", Limit: 5, Window: time.Minute}
	mustHit(t, l, rule, "old")
	clock.t = clock.t.Add(2 * time.Hour)
	mustHit(t, l, rule, "new")

	n, err := l.Sweep(context.Background(), time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("Sweep deleted %d, %v; want 1", n, err)
	}
	var count int64
	db.Model(&models.RateLimitCounter{}).Count(&count)
	if count != 1 {
		t.Fatalf("%d rows remain, want 1", count)
	}
}

func TestStartSweepStops(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))
	stop := l.StartSweep(time.Millisecond, time.Hour)
	time.Sleep(5 * time.Millisecond)
	stop()
	stop() // idempotent
}

// loginRouter mirrors the /login wiring in cmd/api/main.go at small limits.
func loginRouter(l *Limiter) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", Middleware(l,
		Check{Rule: Rule{Name: "login:user_ip", Limit: 2, Window: time.Minute}, Field: "username"},
		Check{Rule: Rule{Name: "login:ip", Limit: 5, Window: time.Minute}},
	), func(c *gin.Context) {
		c.Header("X-Handler-Reached", "1")
		body, _ := io.ReadAll(c.Request.Body)
		c.String(http.StatusOK, string(body))
	})
	return r
}

func postLogin(r *gin.Engine, ip, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":12345"
	r.ServeHTTP(w, req)
	return w
}

func TestMiddleware_SharedIPVolunteersDoNotThrottleEachOther(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))
	r := loginRouter(l)
	shelterIP := "198.51.100.10"

	// Two volunteers on the shelter's IP each get their own per-account budget.
	for _, user := range []string{"alice", "bob"} {
		for i := 0; i < 2; i++ {
			if w := postLogin(r, shelterIP, `{"username":"`+user+`","password":"x"}`); w.Code != http.StatusOK {
				t.Fatalf("%s attempt %d: got %d", user, i+1, w.Code)
			}
		}
	}
	// Alice's third attempt trips her per-account limit (case-insensitively).
	w := postLogin(r, shelterIP, `{"username":"ALICE ","password":"x"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("alice's 3rd attempt: got %d, want 429", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("429 must carry Retry-After")
	}
	// The same username from another IP is a different subject.
	if w := postLogin(r, "203.0.113.5", `{"username":"alice","password":"x"}`); w.Code != http.StatusOK {
		t.Fatalf("alice from another IP: got %d, want 200", w.Code)
	}
}

func TestMiddleware_PerIPCeilingCapsSpraying(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))
	r := loginRouter(l)
	ip := "192.0.2.44"

	users := []string{"u1", "u2", "u3", "u4", "u5", "u6"}
	var last *httptest.ResponseRecorder
	for _, u := range users {
		last = postLogin(r, ip, `{"username":"`+u+`","password":"x"}`)
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("6th distinct username from one IP: got %d, want 429 from the per-IP rule", last.Code)
	}
}

func TestMiddleware_PassesBodyThroughAndHandlesMissingField(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))
	r := loginRouter(l)

	body := `{"username":"carol","password":"secret"}`
	if w := postLogin(r, "192.0.2.1", body); w.Code != http.StatusOK || w.Body.String() != body {
		t.Fatalf("handler should receive the original body; got %d %q", w.Code, w.Body.String())
	}
	// No username (or invalid JSON): only the per-IP rule applies.
	for i := 0; i < 5; i++ {
		if w := postLogin(r, "192.0.2.2", `not json`); w.Code != http.StatusOK {
			t.Fatalf("request %d without a username: got %d", i+1, w.Code)
		}
	}
	if w := postLogin(r, "192.0.2.2", `not json`); w.Code != http.StatusTooManyRequests {
		t.Fatalf("per-IP rule should still apply without a username; got %d", w.Code)
	}
}

func TestMiddleware_DatabaseErrorFailsOpen(t *testing.T) {
	db := openSQLite(t)
	l, _ := newTestLimiter(t, db)
	r := loginRouter(l)
	sqlDB, _ := db.DB()
	sqlDB.Close()

	if w := postLogin(r, "192.0.2.3", `{"username":"dave"}`); w.Code != http.StatusOK {
		t.Fatalf("a limiter DB error should not block login; got %d", w.Code)
	}
}

// loginRouterWithBodyCap mirrors loginRouter but also applies
// middleware.MaxRequestBodySize ahead of the rate limiter, the way
// cmd/api/main.go's global 10MB cap sits ahead of every /api route. capBytes
// is kept tiny in tests so a modestly padded body is enough to trip it.
func loginRouterWithBodyCap(l *Limiter, capBytes int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", middleware.MaxRequestBodySize(capBytes), Middleware(l,
		Check{Rule: Rule{Name: "login:user_ip", Limit: 2, Window: time.Minute}, Field: "username"},
		Check{Rule: Rule{Name: "login:ip", Limit: 3, Window: time.Minute}},
	), func(c *gin.Context) {
		c.Header("X-Handler-Reached", "1")
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "too large")
			return
		}
		c.String(http.StatusOK, string(body))
	})
	return r
}

// TestMiddleware_OversizedBodyNeverReachesHandler reproduces the bypass where
// a body padded past a size cap made the middleware skip every check and
// hand the request to the handler (which ignores trailing padding). Both
// ways of padding must stop at the middleware: past an upstream cap
// (MaxRequestBodySize), and past bodyPeekCap inside the JSON object, where
// a truncated prefix would otherwise hide the username from the
// per-account check.
func TestMiddleware_OversizedBodyNeverReachesHandler(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))
	cases := map[string]struct {
		router *gin.Engine
		body   string
	}{
		"past an upstream body cap": {
			router: loginRouterWithBodyCap(l, 1024),
			body:   `{"username":"victim","password":"` + strings.Repeat("A", 4000) + `"}`,
		},
		"past bodyPeekCap, padded inside the object": {
			router: loginRouter(l),
			body:   `{"username":"victim","password":"x"` + strings.Repeat(" ", bodyPeekCap+10) + `}`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 5; i++ {
				w := postLogin(tc.router, "203.0.113.9", tc.body)
				if w.Header().Get("X-Handler-Reached") != "" {
					t.Fatalf("request %d reached the handler with an oversized body (status %d)", i+1, w.Code)
				}
				if w.Code != http.StatusRequestEntityTooLarge && w.Code != http.StatusTooManyRequests {
					t.Fatalf("request %d: got %d, want 413 or 429", i+1, w.Code)
				}
			}
		})
	}
}

// TestMiddleware_FieldExtractionMatchesJSONKeyCaseInsensitively reproduces a
// bypass of the per-account check: the limiter extracted the field with
// map[string]json.RawMessage (an exact key match), but the login handler
// binds into a struct, and encoding/json matches struct fields
// case-insensitively. So {"USERNAME":"victim"} skipped the per-account
// check entirely even though the handler resolves it to the same account.
func TestMiddleware_FieldExtractionMatchesJSONKeyCaseInsensitively(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))
	r := loginRouter(l)
	ip := "198.51.100.20"

	for i := 0; i < 2; i++ {
		postLogin(r, ip, `{"username":"victim","password":"x"}`)
	}
	w := postLogin(r, ip, `{"USERNAME":"victim","password":"x"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("a case-varied key must still resolve to victim's exhausted budget; got %d", w.Code)
	}
}

// TestMiddleware_FieldExtractionMatchesJSONDuplicateKeyLastWins reproduces
// the companion bypass: a request with a decoy key followed by the real key
// (e.g. {"username":"decoy","Username":"victim"}) charged the decoy's
// budget under exact-match extraction, instead of victim's - the account
// encoding/json's last-key-wins struct decoding actually resolves to.
func TestMiddleware_FieldExtractionMatchesJSONDuplicateKeyLastWins(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))
	r := loginRouter(l)
	ip := "198.51.100.21"

	for i := 0; i < 2; i++ {
		postLogin(r, ip, `{"username":"victim2","password":"x"}`)
	}
	w := postLogin(r, ip, `{"username":"decoy","Username":"victim2","password":"x"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("duplicate keys should extract the last value, matching encoding/json; got %d", w.Code)
	}
}

// TestHit_RetryAfterCappedAtWindow reproduces a skewed replica writing a
// window_start in the future: retryAfter was computed straight off that
// row and could exceed the rule's own window.
func TestHit_RetryAfterCappedAtWindow(t *testing.T) {
	db := openSQLite(t)
	l, clock := newTestLimiter(t, db)
	rule := Rule{Name: "skew-future", Limit: 1, Window: time.Minute}

	futureWindowStart := clock.t.Add(time.Hour).Unix()
	futureWindowStart -= futureWindowStart % 60
	if err := db.Exec(`INSERT INTO rate_limit_counters (key, window_start, count) VALUES (?, ?, ?)`,
		counterKey(rule.Name, "alice"), futureWindowStart, rule.Limit+1).Error; err != nil {
		t.Fatalf("seed skewed row: %v", err)
	}

	_, wait, err := l.Hit(context.Background(), rule, "alice")
	if err != nil {
		t.Fatalf("Hit: %v", err)
	}
	if wait > rule.Window {
		t.Fatalf("retryAfter = %s, must be capped at the rule window %s", wait, rule.Window)
	}
}

// TestMiddleware_ContextCanceledNotLoggedAsError reproduces Hit failing open
// on a canceled request context being logged at error level, which pages
// on-call for something that just means the client went away.
func TestMiddleware_ContextCanceledNotLoggedAsError(t *testing.T) {
	l, _ := newTestLimiter(t, openSQLite(t))

	var buf bytes.Buffer
	testLogger := logging.New(logging.DEBUG, &buf, false)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("logger", testLogger) })
	r.POST("/login", Middleware(l, Check{Rule: Rule{Name: "ctxcancel", Limit: 5, Window: time.Minute}}),
		func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString("{}"))
	req = req.WithContext(ctx)
	req.RemoteAddr = "203.0.113.99:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if strings.Contains(buf.String(), "ERROR:") {
		t.Fatalf("a canceled request context should not be logged as an error: %s", buf.String())
	}
}
