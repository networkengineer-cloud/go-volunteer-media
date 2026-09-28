package authz

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestPolicyMatrix pins down, for every action, exactly which roles may
// perform it. Changing a row here is a product decision - review it as one.
func TestPolicyMatrix(t *testing.T) {
	roles := []Role{RoleNone, RoleMember, RoleGroupAdmin, RoleSiteAdmin}
	// want[i] is whether roles[i] is allowed.
	matrix := map[Action][4]bool{
		ViewGroup:         {false, true, true, true},
		PostContent:       {false, true, true, true},
		ManageOwnSchedule: {false, true, true, true},

		ManageAnimals:       {false, false, true, true},
		ManageContent:       {false, false, true, true},
		ModerateContent:     {false, false, true, true},
		ManageSchedule:      {false, false, true, true},
		ManageMembers:       {false, false, true, true},
		ViewMemberDetails:   {false, false, true, true},
		ManageGroupSettings: {false, false, true, true},
		Announce:            {false, false, true, true},

		ModerateMedia:          {false, false, false, true},
		ConfigureGroupFeatures: {false, false, false, true},
	}

	if len(matrix) != len(policy) {
		t.Fatalf("matrix covers %d actions, policy has %d: add the new action to this test", len(matrix), len(policy))
	}
	for action, want := range matrix {
		if _, ok := policy[action]; !ok {
			t.Errorf("%s is in the test matrix but not in the policy", action)
		}
		for i, role := range roles {
			if got := Allows(role, action); got != want[i] {
				t.Errorf("Allows(%s, %s) = %v, want %v", role, action, got, want[i])
			}
		}
	}
}

func TestAllowsUnknownActionDenied(t *testing.T) {
	for _, role := range []Role{RoleNone, RoleMember, RoleGroupAdmin, RoleSiteAdmin} {
		if Allows(role, Action("no.such.action")) {
			t.Errorf("unknown action allowed for %s", role)
		}
	}
}

type fixture struct {
	db                                   *gorm.DB
	groupA, groupB, groupC, deletedGroup models.Group
	member, groupAdmin, siteAdmin        models.User
	outsider, deletedUser, adminOfBoth   models.User
	memberOfDeleted, orphan, peerMember  models.User
	adminOfC, memberOfTwoGroups          models.User
}

func setup(t *testing.T) *fixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return setupWith(t, db)
}

// setupWith migrates and seeds the fixture into db (SQLite here, a
// rolled-back Postgres transaction in authz_postgres_test.go).
func setupWith(t *testing.T, db *gorm.DB) *fixture {
	t.Helper()
	if err := db.AutoMigrate(&models.User{}, &models.Group{}, &models.UserGroup{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	f := &fixture{db: db}

	// Names are prefixed so the fixture can't collide with the default
	// groups and seed users already in a shared Postgres test database
	// (authz_postgres_test.go runs this inside a rolled-back transaction).
	mkGroup := func(name string) models.Group {
		g := models.Group{Name: "authz-test-" + name}
		if err := db.Create(&g).Error; err != nil {
			t.Fatalf("create group: %v", err)
		}
		return g
	}
	mkUser := func(name string, siteAdmin bool) models.User {
		u := models.User{Username: "authz-test-" + name, Email: "authz-test-" + name + "@example.com", Password: "x", IsAdmin: siteAdmin}
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		return u
	}
	join := func(u models.User, g models.Group, admin bool) {
		if err := db.Create(&models.UserGroup{UserID: u.ID, GroupID: g.ID, IsGroupAdmin: admin}).Error; err != nil {
			t.Fatalf("join: %v", err)
		}
	}

	f.groupA = mkGroup("dogs")
	f.groupB = mkGroup("cats")
	f.groupC = mkGroup("modsquad")
	f.deletedGroup = mkGroup("retired")

	f.member = mkUser("member", false)
	f.groupAdmin = mkUser("groupadmin", false)
	f.siteAdmin = mkUser("siteadmin", true)
	f.outsider = mkUser("outsider", false)
	f.deletedUser = mkUser("deleted", false)
	f.adminOfBoth = mkUser("adminofboth", false)
	f.memberOfDeleted = mkUser("memberofdeleted", false)
	f.orphan = mkUser("orphan", false)
	f.peerMember = mkUser("peer", false)
	// adminOfC shares group A with groupAdmin (a plain member there) but is a
	// group admin of group C, which groupAdmin does not administer - the
	// lateral-escalation shape TestCheckManageUser guards against.
	f.adminOfC = mkUser("adminofc", false)
	// memberOfTwoGroups is a plain member (never a group admin anywhere) of
	// both group A and group B, to confirm the new rule only looks at the
	// target's *admin* groups, not every group the target merely belongs to.
	f.memberOfTwoGroups = mkUser("memberoftwogroups", false)

	join(f.member, f.groupA, false)
	join(f.peerMember, f.groupB, false)
	join(f.groupAdmin, f.groupA, true)
	join(f.groupAdmin, f.deletedGroup, true)
	join(f.outsider, f.groupB, false)
	join(f.deletedUser, f.groupA, true)
	join(f.adminOfBoth, f.groupA, true)
	join(f.adminOfBoth, f.groupB, true)
	join(f.memberOfDeleted, f.deletedGroup, false)
	join(f.siteAdmin, f.groupB, false)
	join(f.adminOfC, f.groupA, false)
	join(f.adminOfC, f.groupC, true)
	join(f.memberOfTwoGroups, f.groupA, false)
	join(f.memberOfTwoGroups, f.groupB, false)

	if err := db.Delete(&f.deletedGroup).Error; err != nil {
		t.Fatalf("soft-delete group: %v", err)
	}
	if err := db.Delete(&f.deletedUser).Error; err != nil {
		t.Fatalf("soft-delete user: %v", err)
	}
	return f
}

func subj(u models.User) Subject { return Subject{UserID: u.ID, IsSiteAdmin: u.IsAdmin} }

func TestGroupRole(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	cases := []struct {
		name  string
		s     Subject
		group uint
		want  Role
	}{
		{"member of the group", subj(f.member), f.groupA.ID, RoleMember},
		{"group admin of the group", subj(f.groupAdmin), f.groupA.ID, RoleGroupAdmin},
		{"member of another group only", subj(f.outsider), f.groupA.ID, RoleNone},
		{"site admin, not a member", subj(f.siteAdmin), f.groupA.ID, RoleSiteAdmin},
		{"site admin, also a member", subj(f.siteAdmin), f.groupB.ID, RoleSiteAdmin},
		{"soft-deleted user keeps no role", subj(f.deletedUser), f.groupA.ID, RoleNone},
		{"admin of a soft-deleted group has no role there", subj(f.groupAdmin), f.deletedGroup.ID, RoleNone},
		{"member of a soft-deleted group has no role there", subj(f.memberOfDeleted), f.deletedGroup.ID, RoleNone},
		{"nonexistent group", subj(f.member), 9999, RoleNone},
		{"zero group ID", subj(f.member), 0, RoleNone},
		{"zero user ID", Subject{}, f.groupA.ID, RoleNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GroupRole(ctx, f.db, tc.s, tc.group)
			if err != nil {
				t.Fatalf("GroupRole: %v", err)
			}
			if got != tc.want {
				t.Errorf("GroupRole = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestCan(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	check := func(s Subject, a Action, g uint, want bool) {
		t.Helper()
		got, err := Can(ctx, f.db, s, a, g)
		if err != nil {
			t.Fatalf("Can: %v", err)
		}
		if got != want {
			t.Errorf("Can(user %d, %s, group %d) = %v, want %v", s.UserID, a, g, got, want)
		}
	}

	check(subj(f.member), ViewGroup, f.groupA.ID, true)
	check(subj(f.member), ManageAnimals, f.groupA.ID, false)
	check(subj(f.groupAdmin), ManageAnimals, f.groupA.ID, true)
	check(subj(f.groupAdmin), ConfigureGroupFeatures, f.groupA.ID, false)
	check(subj(f.outsider), ViewGroup, f.groupA.ID, false)
	check(subj(f.siteAdmin), ConfigureGroupFeatures, f.groupA.ID, true)
}

func TestCanReturnsDBError(t *testing.T) {
	f := setup(t)
	sqlDB, _ := f.db.DB()
	sqlDB.Close()

	allowed, err := Can(context.Background(), f.db, subj(f.member), ViewGroup, f.groupA.ID)
	if err == nil {
		t.Fatal("expected an error from a closed database")
	}
	if allowed {
		t.Fatal("a database error must never allow")
	}
}

func testContext(userID any, isAdmin any) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	if userID != nil {
		c.Set("user_id", userID)
	}
	if isAdmin != nil {
		c.Set("is_admin", isAdmin)
	}
	return c
}

func TestCallerCan(t *testing.T) {
	f := setup(t)

	if !CallerCan(testContext(f.member.ID, false), f.db, ViewGroup, f.groupA.ID) {
		t.Error("member should view their group")
	}
	if CallerCan(testContext(nil, true), f.db, ViewGroup, f.groupA.ID) {
		t.Error("no user_id in context must deny, even with is_admin set")
	}
	if CallerCan(testContext(int(f.member.ID), false), f.db, ViewGroup, f.groupA.ID) {
		t.Error("a user_id of the wrong type must deny")
	}
	if CallerCan(testContext(f.outsider.ID, "true"), f.db, ViewGroup, f.groupA.ID) {
		t.Error("a non-bool is_admin must not grant site admin")
	}

	sqlDB, _ := f.db.DB()
	sqlDB.Close()
	if CallerCan(testContext(f.member.ID, false), f.db, ViewGroup, f.groupA.ID) {
		t.Error("a database error must deny")
	}
}

func TestCallerRole(t *testing.T) {
	f := setup(t)
	if got := CallerRole(testContext(f.groupAdmin.ID, false), f.db, f.groupA.ID); got != RoleGroupAdmin {
		t.Errorf("CallerRole = %s, want group_admin", got)
	}
	if got := CallerRole(testContext(nil, nil), f.db, f.groupA.ID); got != RoleNone {
		t.Errorf("CallerRole without caller = %s, want none", got)
	}
}

func TestParseGroupID(t *testing.T) {
	for raw, want := range map[string]bool{"1": true, "42": true, "0": false, "-1": false, "abc": false, "": false, "99999999999": false} {
		if _, ok := ParseGroupID(raw); ok != want {
			t.Errorf("ParseGroupID(%q) ok = %v, want %v", raw, ok, want)
		}
	}
}

func TestGroupsWhere(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	scope, err := GroupsWhere(ctx, f.db, subj(f.siteAdmin), ManageAnimals)
	if err != nil || !scope.All || scope.Empty() {
		t.Fatalf("site admin scope = %+v, %v; want All", scope, err)
	}

	scope, err = GroupsWhere(ctx, f.db, subj(f.adminOfBoth), ManageAnimals)
	if err != nil {
		t.Fatal(err)
	}
	if scope.All || !scope.Contains(f.groupA.ID) || !scope.Contains(f.groupB.ID) || len(scope.GroupIDs) != 2 {
		t.Errorf("admin of both scope = %+v", scope)
	}

	// The soft-deleted group the group admin also administers is excluded.
	scope, err = GroupsWhere(ctx, f.db, subj(f.groupAdmin), ManageAnimals)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope.GroupIDs) != 1 || !scope.Contains(f.groupA.ID) || scope.Contains(f.deletedGroup.ID) {
		t.Errorf("group admin scope = %+v, want only group A", scope)
	}

	// A member-level action spans every group the user belongs to...
	scope, err = GroupsWhere(ctx, f.db, subj(f.member), ViewGroup)
	if err != nil || !scope.Contains(f.groupA.ID) {
		t.Errorf("member view scope = %+v, %v", scope, err)
	}
	// ...an admin-level one spans none of them.
	scope, err = GroupsWhere(ctx, f.db, subj(f.member), ManageAnimals)
	if err != nil || !scope.Empty() {
		t.Errorf("member manage scope = %+v, %v; want empty", scope, err)
	}
	// Site-admin-only actions are empty for everyone else.
	scope, err = GroupsWhere(ctx, f.db, subj(f.groupAdmin), ConfigureGroupFeatures)
	if err != nil || !scope.Empty() {
		t.Errorf("group admin features scope = %+v, %v; want empty", scope, err)
	}
	// Soft-deleted users get nothing.
	scope, err = GroupsWhere(ctx, f.db, subj(f.deletedUser), ManageAnimals)
	if err != nil || !scope.Empty() {
		t.Errorf("deleted user scope = %+v, %v; want empty", scope, err)
	}
}

func TestCheckManageUser(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	load := func(u models.User) *models.User {
		var out models.User
		if err := f.db.Preload("Groups").First(&out, u.ID).Error; err != nil {
			t.Fatalf("load user: %v", err)
		}
		return &out
	}

	cases := []struct {
		name   string
		caller models.User
		target models.User
		want   UserDenial
	}{
		{"site admin manages anyone", f.siteAdmin, f.member, UserAllowed},
		{"site admin manages a user with no groups", f.siteAdmin, f.orphan, UserAllowed},
		{"group admin manages a member of their group", f.groupAdmin, f.member, UserAllowed},
		{"group admin cannot manage a site admin", f.adminOfBoth, f.siteAdmin, DenyTargetIsSiteAdmin},
		{"group admin cannot manage a user with no groups", f.groupAdmin, f.orphan, DenyTargetHasNoGroups},
		{"group admin cannot manage a member of another group", f.groupAdmin, f.outsider, DenyNoSharedAdminGroup},
		{"plain member cannot manage a fellow member", f.member, f.groupAdmin, DenyNoSharedAdminGroup},
		{"admin of a soft-deleted group gets nothing from it", f.groupAdmin, f.memberOfDeleted, DenyTargetHasNoGroups},
		{"admin of two groups manages a member of the second", f.adminOfBoth, f.peerMember, UserAllowed},

		// Lateral privilege escalation (the vulnerability this table now
		// guards against): groupAdmin shares group A with adminOfC (a plain
		// member there), but adminOfC is a group admin of group C, which
		// groupAdmin does not administer. Taking over adminOfC's account
		// (password reset, etc.) would hand groupAdmin admin rights in group
		// C, so this must be denied even though the two share a group.
		{"blocked: caller cannot manage a target who admins a group the caller doesn't", f.groupAdmin, f.adminOfC, DenyTargetAdminsOtherGroup},
		// Still allowed: a plain volunteer who happens to belong to two
		// groups is manageable by an admin of just one of them - the new
		// rule only restricts targets who are themselves group admins
		// elsewhere, not targets who are merely members elsewhere.
		{"still allowed: volunteer who is a plain member of two groups", f.groupAdmin, f.memberOfTwoGroups, UserAllowed},
		// Allowed: adminOfBoth administers every group adminOfC administers
		// admin rights in (group C is not one of them, group A is shared,
		// and the only group adminOfC administers is group C - so use a
		// target whose sole admin group, group A, is one adminOfBoth also
		// administers).
		{"allowed: caller administers every group the target administers", f.adminOfBoth, f.groupAdmin, UserAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CheckManageUser(ctx, f.db, subj(tc.caller), load(tc.target))
			if err != nil {
				t.Fatalf("CheckManageUser: %v", err)
			}
			if got != tc.want {
				t.Errorf("CheckManageUser = %d, want %d", got, tc.want)
			}
		})
	}

	if _, err := CheckManageUser(ctx, f.db, Subject{}, load(f.member)); err == nil {
		t.Error("expected ErrNoSubject for an empty subject")
	}
}

func TestRoleString(t *testing.T) {
	for r, want := range map[Role]string{RoleNone: "none", RoleMember: "member", RoleGroupAdmin: "group_admin", RoleSiteAdmin: "site_admin", Role(9): "unknown(9)"} {
		if got := r.String(); got != want {
			t.Errorf("Role(%d).String() = %q, want %q", int(r), got, want)
		}
	}
}
