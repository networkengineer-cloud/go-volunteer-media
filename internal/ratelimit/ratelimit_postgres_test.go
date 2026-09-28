package ratelimit

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openPostgres connects with the same DB_* env vars as the handler
// _postgres_test.go files and skips when nothing is listening.
func openPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	env := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	host, port := env("DB_HOST", "localhost"), env("DB_PORT", "5432")
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		t.Skipf("skipping: no Postgres reachable at %s:%s", host, port)
	}
	_ = conn.Close()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
		host, port, env("DB_USER", "postgres"), env("DB_PASSWORD", "postgres"),
		env("DB_NAME", "volunteer_media_test"), env("DB_SSLMODE", "disable"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Skipf("skipping: could not connect to postgres (%v)", err)
	}
	if err := db.AutoMigrate(&models.RateLimitCounter{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	return db
}

// TestHit_Postgres_ConcurrentReplicasShareOneLimit simulates several
// replicas - separate Limiters with separate connection pools - racing on
// one key. Exactly Limit hits may succeed.
func TestHit_Postgres_ConcurrentReplicasShareOneLimit(t *testing.T) {
	const replicas, hitsPerReplica, limit = 3, 20, 7
	rule := Rule{Name: "pgtest:" + t.Name(), Limit: limit, Window: time.Hour}
	subject := fmt.Sprintf("subject-%d", time.Now().UnixNano())

	limiters := make([]*Limiter, replicas)
	for i := range limiters {
		limiters[i] = New(openPostgres(t))
	}
	t.Cleanup(func() {
		limiters[0].db.Exec("DELETE FROM rate_limit_counters WHERE key = ?", counterKey(rule.Name, subject))
	})

	var allowed atomic.Int64
	var wg sync.WaitGroup
	for _, l := range limiters {
		for i := 0; i < hitsPerReplica; i++ {
			wg.Add(1)
			go func(l *Limiter) {
				defer wg.Done()
				ok, _, err := l.Hit(context.Background(), rule, subject)
				if err != nil {
					t.Errorf("Hit: %v", err)
					return
				}
				if ok {
					allowed.Add(1)
				}
			}(l)
		}
	}
	wg.Wait()

	if got := allowed.Load(); got != limit {
		t.Fatalf("%d hits allowed across %d replicas, want exactly %d", got, replicas, limit)
	}
}
