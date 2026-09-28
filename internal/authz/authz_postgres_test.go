package authz

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These tests run the role and scope queries against real Postgres, where
// "groups" is a keyword and soft-delete joins behave as in production. They
// use the same DB_* env vars as the handler _postgres_test.go files and skip
// when nothing is listening. Everything runs in a transaction that is rolled
// back.
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
	tx := db.Begin()
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

func TestGroupRole_Postgres(t *testing.T) {
	f := setupWith(t, openPostgres(t))
	ctx := context.Background()

	for _, tc := range []struct {
		name  string
		s     Subject
		group uint
		want  Role
	}{
		{"member", subj(f.member), f.groupA.ID, RoleMember},
		{"group admin", subj(f.groupAdmin), f.groupA.ID, RoleGroupAdmin},
		{"outsider", subj(f.outsider), f.groupA.ID, RoleNone},
		{"soft-deleted user", subj(f.deletedUser), f.groupA.ID, RoleNone},
		{"soft-deleted group", subj(f.groupAdmin), f.deletedGroup.ID, RoleNone},
	} {
		got, err := GroupRole(ctx, f.db, tc.s, tc.group)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s: GroupRole = %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestGroupsWhereAndCheckManageUser_Postgres(t *testing.T) {
	f := setupWith(t, openPostgres(t))
	ctx := context.Background()

	scope, err := GroupsWhere(ctx, f.db, subj(f.groupAdmin), ManageAnimals)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope.GroupIDs) != 1 || !scope.Contains(f.groupA.ID) {
		t.Errorf("scope = %+v, want only group A", scope)
	}

	target := f.member
	if err := f.db.Preload("Groups").First(&target, f.member.ID).Error; err != nil {
		t.Fatal(err)
	}
	denial, err := CheckManageUser(ctx, f.db, subj(f.groupAdmin), &target)
	if err != nil || denial != UserAllowed {
		t.Errorf("CheckManageUser = %d, %v; want allowed", denial, err)
	}
}
