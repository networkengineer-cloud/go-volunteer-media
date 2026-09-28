package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

// Regression tests for handler authorization that moved onto the central
// policy in internal/authz. Per-role policy coverage lives in
// internal/authz/authz_test.go; these check the handler wiring.

func TestBulkUpdateAnimals_GroupAdminCannotMoveAnimalsIntoForeignGroup(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "groupadmin", "ga@example.com", "password123", false)
	own := CreateTestGroup(t, db, "Dogs", "")
	foreign := CreateTestGroup(t, db, "Cats", "")
	AddUserToGroupWithAdmin(t, db, admin.ID, own.ID, true)
	animal := CreateTestAnimal(t, db, own.ID, "Rex", "Dog")

	send := func(groupID uint) *httptest.ResponseRecorder {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", admin.ID)
		c.Set("is_admin", false)
		body := fmt.Sprintf(`{"animal_ids":[%d],"group_id":%d}`, animal.ID, groupID)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/bulk-animals/bulk-update", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		BulkUpdateAnimals(db)(c)
		return w
	}

	if w := send(foreign.ID); w.Code != http.StatusForbidden {
		t.Fatalf("moving into a group the caller does not administer: expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var reloaded models.Animal
	db.First(&reloaded, animal.ID)
	if reloaded.GroupID != own.ID {
		t.Fatalf("animal moved to group %d despite the 403", reloaded.GroupID)
	}

	if w := send(own.ID); w.Code != http.StatusOK {
		t.Fatalf("moving within the caller's own group: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGroupAdminDeleteUser_GroupAdminDeletesMemberOfTheirGroup(t *testing.T) {
	db := SetupTestDB(t)
	admin := CreateTestUser(t, db, "groupadmin", "ga@example.com", "password123", false)
	member := CreateTestUser(t, db, "member", "m@example.com", "password123", false)
	outsider := CreateTestUser(t, db, "outsider", "o@example.com", "password123", false)
	siteAdmin := CreateTestUser(t, db, "siteadmin", "sa@example.com", "password123", true)
	dogs := CreateTestGroup(t, db, "Dogs", "")
	cats := CreateTestGroup(t, db, "Cats", "")
	AddUserToGroupWithAdmin(t, db, admin.ID, dogs.ID, true)
	AddUserToGroupWithAdmin(t, db, member.ID, dogs.ID, false)
	AddUserToGroupWithAdmin(t, db, outsider.ID, cats.ID, false)
	AddUserToGroupWithAdmin(t, db, siteAdmin.ID, dogs.ID, false)

	del := func(target uint) *httptest.ResponseRecorder {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", admin.ID)
		c.Set("is_admin", false)
		c.Params = gin.Params{{Key: "userId", Value: itoa(target)}}
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/users/"+itoa(target), nil)
		GroupAdminDeleteUser(db)(c)
		return w
	}

	for name, tc := range map[string]struct {
		target uint
		want   int
	}{
		"member of another group": {outsider.ID, http.StatusForbidden},
		"site admin in own group": {siteAdmin.ID, http.StatusForbidden},
		"member of own group":     {member.ID, http.StatusOK},
	} {
		t.Run(name, func(t *testing.T) {
			if w := del(tc.target); w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestDeleteAnimalImage_GroupAdminCannotDeleteOthersImage(t *testing.T) {
	// ModerateMedia is site-admin only (unchanged from before the central
	// policy): group admins moderate comments, not photos.
	db := SetupTestDB(t)
	if err := db.AutoMigrate(&models.AnimalImage{}); err != nil {
		t.Fatal(err)
	}
	admin := CreateTestUser(t, db, "groupadmin", "ga@example.com", "password123", false)
	uploader := CreateTestUser(t, db, "uploader", "u@example.com", "password123", false)
	group := CreateTestGroup(t, db, "Dogs", "")
	AddUserToGroupWithAdmin(t, db, admin.ID, group.ID, true)
	AddUserToGroupWithAdmin(t, db, uploader.ID, group.ID, false)
	animal := CreateTestAnimal(t, db, group.ID, "Rex", "Dog")
	img := models.AnimalImage{AnimalID: &animal.ID, UserID: uploader.ID, ImageURL: "/api/images/x"}
	if err := db.Create(&img).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", admin.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{
		{Key: "id", Value: itoa(group.ID)},
		{Key: "animalId", Value: itoa(animal.ID)},
		{Key: "imageId", Value: itoa(img.ID)},
	}
	c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
	DeleteAnimalImage(db, &mockStorageProvider{ProviderName: "postgres"})(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "You can only delete your own images" {
		t.Fatalf("unexpected error: %q", resp["error"])
	}
}

func TestGetUserProfile_GroupAdminSeesSkillTagsFromTheirGroupsOnly(t *testing.T) {
	db := SetupTestDB(t)
	if err := db.AutoMigrate(&models.UserSkillTag{}); err != nil {
		t.Fatal(err)
	}
	admin := CreateTestUser(t, db, "groupadmin", "ga@example.com", "password123", false)
	member := CreateTestUser(t, db, "member", "m@example.com", "password123", false)
	dogs := CreateTestGroup(t, db, "Dogs", "")
	cats := CreateTestGroup(t, db, "Cats", "")
	AddUserToGroupWithAdmin(t, db, admin.ID, dogs.ID, true)
	AddUserToGroupWithAdmin(t, db, member.ID, dogs.ID, false)
	AddUserToGroupWithAdmin(t, db, member.ID, cats.ID, false)

	dogTag := models.UserSkillTag{GroupID: dogs.ID, Name: "Leash trained"}
	catTag := models.UserSkillTag{GroupID: cats.ID, Name: "Litter duty"}
	db.Create(&dogTag)
	db.Create(&catTag)
	if err := db.Model(member).Association("SkillTags").Append(&dogTag, &catTag); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", admin.ID)
	c.Set("is_admin", false)
	c.Params = gin.Params{{Key: "id", Value: itoa(member.ID)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	GetUserProfile(db)(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		SkillTags []struct {
			Name string `json:"name"`
		} `json:"skill_tags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.SkillTags) != 1 || resp.SkillTags[0].Name != "Leash trained" {
		t.Fatalf("expected only the Dogs skill tag, got %+v", resp.SkillTags)
	}
}
