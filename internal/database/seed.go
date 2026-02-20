package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeedData populates the database with demo data for testing and demonstrations
// If force is true, it will seed data even if users already exist
func SeedData(db *gorm.DB, force bool) error {
	logging.Info("Starting database seeding...")

	// Check if data already exists
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount > 0 && !force {
		if err := ensureSandboxMembership(db); err != nil {
			return fmt.Errorf("failed to ensure sandbox memberships: %w", err)
		}
		logging.Info("Database already contains users - ensured sandbox membership; skipping seed data (use --force to override)")
		return nil
	}

	// If force is true, delete existing data
	if force && userCount > 0 {
		logging.Info("Force flag set - deleting existing data...")

		// Delete in reverse order of foreign key dependencies
		if err := db.Exec("DELETE FROM animal_comment_tags").Error; err != nil {
			return fmt.Errorf("failed to delete animal_comment_tags: %w", err)
		}
		if err := db.Exec("DELETE FROM animal_animal_tags").Error; err != nil {
			return fmt.Errorf("failed to delete animal_animal_tags: %w", err)
		}
		if err := db.Exec("DELETE FROM animal_name_histories").Error; err != nil {
			return fmt.Errorf("failed to delete animal_name_histories: %w", err)
		}
		if err := db.Exec("DELETE FROM animal_comments").Error; err != nil {
			return fmt.Errorf("failed to delete animal_comments: %w", err)
		}
		if err := db.Exec("DELETE FROM animal_images").Error; err != nil {
			return fmt.Errorf("failed to delete animal_images: %w", err)
		}
		if err := db.Exec("DELETE FROM animals").Error; err != nil {
			return fmt.Errorf("failed to delete animals: %w", err)
		}
		if err := db.Exec("DELETE FROM updates").Error; err != nil {
			return fmt.Errorf("failed to delete updates: %w", err)
		}
		if err := db.Exec("DELETE FROM announcements").Error; err != nil {
			return fmt.Errorf("failed to delete announcements: %w", err)
		}
		if err := db.Exec("DELETE FROM protocols").Error; err != nil {
			return fmt.Errorf("failed to delete protocols: %w", err)
		}
		if err := db.Exec("DELETE FROM user_groups").Error; err != nil {
			return fmt.Errorf("failed to delete user_groups: %w", err)
		}
		if err := db.Exec("DELETE FROM users").Error; err != nil {
			return fmt.Errorf("failed to delete users: %w", err)
		}

		logging.Info("Existing data deleted successfully")
	}

	// Seed users
	users, err := seedUsers(db)
	if err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	// Ensure activity-sandbox group exists for testing
	if err := ensureSandboxGroup(db); err != nil {
		return fmt.Errorf("failed to ensure sandbox group: %w", err)
	}

	// Get groups and update with images
	var groups []models.Group
	if err := db.Find(&groups).Error; err != nil {
		return fmt.Errorf("failed to fetch groups: %w", err)
	}

	// Update ModSquad group with Unsplash images
	if err := updateGroupImages(db, groups); err != nil {
		return fmt.Errorf("failed to update group images: %w", err)
	}

	// Assign users to groups
	if err := assignUsersToGroups(db, users, groups); err != nil {
		return fmt.Errorf("failed to assign users to groups: %w", err)
	}

	// Seed animals
	animals, err := seedAnimals(db, groups)
	if err != nil {
		return fmt.Errorf("failed to seed animals: %w", err)
	}

	// Seed comments
	if err := seedComments(db, users, animals); err != nil {
		return fmt.Errorf("failed to seed comments: %w", err)
	}

	// Seed updates
	if err := seedUpdates(db, users, groups); err != nil {
		return fmt.Errorf("failed to seed updates: %w", err)
	}

	// Seed announcements (email disabled by default; opt-in)
	if err := seedAnnouncements(db, users); err != nil {
		return fmt.Errorf("failed to seed announcements: %w", err)
	}

	// Seed protocols
	if err := seedProtocols(db, users, groups); err != nil {
		return fmt.Errorf("failed to seed protocols: %w", err)
	}

	// Update site settings with hero image
	if err := updateSiteSettings(db); err != nil {
		return fmt.Errorf("failed to update site settings: %w", err)
	}

	logging.Info("Database seeding completed successfully")
	return nil
}

// seedUsers creates demo users focused on ModSquad volunteers
func seedUsers(db *gorm.DB) ([]models.User, error) {
	// Hash passwords (minimum 8 characters for frontend validation)
	// Admin/Group Admins keep demo1234 password
	adminPassword, err := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	// Volunteers use volunteer2026! password
	volunteerPassword, err := bcrypt.GenerateFromPassword([]byte("volunteer2026!"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	users := []models.User{
		{
			Username:                  "admin",
			Email:                     "admin@demo.local",
			Password:                  string(adminPassword),
			IsAdmin:                   true,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "(555) 100-0001",
			HideEmail:                 false,
			HidePhoneNumber:           false,
		},
		{
			Username:                  "merry",
			Email:                     "merry@demo.local",
			Password:                  string(adminPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "(555) 100-0002",
			HideEmail:                 false,
			HidePhoneNumber:           true, // Merry has hidden phone number
		},
		{
			Username:                  "sophia",
			Email:                     "sophia@demo.local",
			Password:                  string(adminPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "(555) 100-0003",
			HideEmail:                 true, // Sophia has hidden email
			HidePhoneNumber:           false,
		},
		{
			Username:                  "terry",
			Email:                     "terry@demo.local",
			Password:                  string(volunteerPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "(555) 100-0004",
			HideEmail:                 false,
			HidePhoneNumber:           false,
		},
		{
			Username:                  "alex",
			Email:                     "alex@demo.local",
			Password:                  string(volunteerPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "(555) 100-0005",
			HideEmail:                 false,
			HidePhoneNumber:           false,
		},
		{
			Username:                  "jordan",
			Email:                     "jordan@demo.local",
			Password:                  string(volunteerPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "(555) 100-0006",
			HideEmail:                 false,
			HidePhoneNumber:           false,
		},
		{
			Username:                  "casey",
			Email:                     "casey@demo.local",
			Password:                  string(volunteerPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "(555) 100-0007",
			HideEmail:                 false,
			HidePhoneNumber:           false,
		},
		{
			Username:                  "taylor",
			Email:                     "taylor@demo.local",
			Password:                  string(volunteerPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "(555) 100-0008",
			HideEmail:                 false,
			HidePhoneNumber:           false,
		},
	}

	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			return nil, err
		}
		logging.WithField("username", users[i].Username).Info("Created demo user")
	}

	return users, nil
}

// updateGroupImages updates groups with Unsplash images for icons and hero banners
func updateGroupImages(db *gorm.DB, groups []models.Group) error {
	for i := range groups {
		switch groups[i].Name {
		case "modsquad":
			// ModSquad group images - professional dog training/behavioral focus
			groups[i].ImageURL = "https://images.unsplash.com/photo-1548199973-03cce0bbc87b?w=400&q=80"      // Dog icon
			groups[i].HeroImageURL = "https://images.unsplash.com/photo-1548199973-03cce0bbc87b?w=1920&q=80" // Hero banner
		case "dogs":
			// Dogs group images - general dog group
			groups[i].ImageURL = "https://images.unsplash.com/photo-1537151608828-ea2b11777ee8?w=400&q=80"      // Happy dogs icon
			groups[i].HeroImageURL = "https://images.unsplash.com/photo-1537151608828-ea2b11777ee8?w=1920&q=80" // Happy dogs banner
		case "cats":
			// Cats group images - general cat group
			groups[i].ImageURL = "https://images.unsplash.com/photo-1514888286974-6c03e2ca1dba?w=400&q=80"      // Cat icon
			groups[i].HeroImageURL = "https://images.unsplash.com/photo-1514888286974-6c03e2ca1dba?w=1920&q=80" // Cat banner
		}

		if err := db.Save(&groups[i]).Error; err != nil {
			return fmt.Errorf("failed to update group %s images: %w", groups[i].Name, err)
		}
		logging.WithField("group_name", groups[i].Name).Info("Updated group with Unsplash images")
	}

	return nil
}

// assignUsersToGroups assigns demo users to ModSquad group (primary focus)
// It also sets group admin status for merry and sophia, and enrolls all users
// into the activity-sandbox group (kept empty for automated tests).
func assignUsersToGroups(db *gorm.DB, users []models.User, groups []models.Group) error {
	// Find ModSquad group (primary group for demo)
	var modsquadGroup models.Group
	var sandboxGroup *models.Group
	for _, g := range groups {
		switch g.Name {
		case "modsquad":
			modsquadGroup = g
		case "activity-sandbox":
			groupCopy := g
			sandboxGroup = &groupCopy
		}
	}

	// All users get access to ModSquad (primary group for first few months)
	// merry and sophia are group admins
	for i := range users {
		if err := db.Model(&users[i]).Association("Groups").Append(&modsquadGroup); err != nil {
			return err
		}

		// Enroll all users in the empty sandbox group for deterministic E2E checks.
		if sandboxGroup != nil {
			if err := db.Model(&users[i]).Association("Groups").Append(sandboxGroup); err != nil {
				return err
			}
		}

		// Set group admin status for merry and sophia
		if users[i].Username == "merry" || users[i].Username == "sophia" {
			// Update the UserGroup record to set IsGroupAdmin = true
			if err := db.Model(&models.UserGroup{}).
				Where("user_id = ? AND group_id = ?", users[i].ID, modsquadGroup.ID).
				Update("is_group_admin", true).Error; err != nil {
				return fmt.Errorf("failed to set group admin for %s: %w", users[i].Username, err)
			}
			logging.WithField("username", users[i].Username).Info("Set user as group admin for ModSquad")
		}
	}

	logging.Info("Assigned all users to ModSquad group")
	return nil
}

// ensureSandboxGroup creates the activity-sandbox group if it doesn't exist
// This group is used for automated testing and kept empty
// Uses upsert to avoid duplicate-key errors under concurrent seed runs
func ensureSandboxGroup(db *gorm.DB) error {
	group := models.Group{
		Name:         "activity-sandbox",
		Description:  "Empty group reserved for automated tests",
		HasProtocols: false,
	}

	// Use upsert to avoid TOCTOU race condition
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"description", "has_protocols"}),
	}).Create(&group).Error; err != nil {
		return fmt.Errorf("failed to ensure activity-sandbox group: %w", err)
	}

	logging.WithField("group_name", group.Name).Debug("Ensured activity-sandbox group exists")
	return nil
}

// ensureSandboxMembership backfills access to the empty activity-sandbox group for existing demo users
// when the main seed routine is skipped due to pre-existing data.
func ensureSandboxMembership(db *gorm.DB) error {
	// First ensure the sandbox group exists (creates if missing via upsert)
	if err := ensureSandboxGroup(db); err != nil {
		return err
	}

	// Fetch the group (guaranteed to exist after ensureSandboxGroup)
	var sandboxGroup models.Group
	if err := db.Where("name = ?", "activity-sandbox").First(&sandboxGroup).Error; err != nil {
		// Should not happen after successful ensureSandboxGroup, but handle gracefully
		return fmt.Errorf("failed to fetch activity-sandbox group after creation: %w", err)
	}

	usernames := []string{"admin", "merry", "sophia", "terry", "alex", "jordan", "casey", "taylor"}
	for _, username := range usernames {
		var user models.User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}

		userGroup := models.UserGroup{UserID: user.ID, GroupID: sandboxGroup.ID}
		if err := db.Where("user_id = ? AND group_id = ?", user.ID, sandboxGroup.ID).FirstOrCreate(&userGroup).Error; err != nil {
			return err
		}
	}

	return nil
}

// seedAnimals creates demo animals for ModSquad group with Unsplash images
func seedAnimals(db *gorm.DB, groups []models.Group) ([]models.Animal, error) {
	var modsquadGroupID uint
	for _, g := range groups {
		if g.Name == "modsquad" {
			modsquadGroupID = g.ID
			break
		}
	}

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)
	fiveDaysAgo := now.AddDate(0, 0, -5)
	tenDaysAgo := now.AddDate(0, 0, -10)
	fifteenDaysAgo := now.AddDate(0, 0, -15)
	twentyDaysAgo := now.AddDate(0, 0, -20)
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	// ModSquad-focused dogs with Unsplash images
	animals := []models.Animal{
		{
			GroupID:          modsquadGroupID,
			Name:             "Buddy",
			Species:          "Dog",
			Breed:            "Golden Retriever",
			Age:              4,
			Description:      "Friendly and energetic golden retriever who loves to play fetch. Great with kids and other dogs. House trained and knows basic commands. Buddy is the perfect family companion!",
			Status:           "available",
			ImageURL:         "https://images.unsplash.com/photo-1633722715463-d30f4f325e24?w=800&q=80", // Golden Retriever
			ArrivalDate:      &thirtyDaysAgo,
			LastStatusChange: &thirtyDaysAgo,
		},
		{
			GroupID:          modsquadGroupID,
			Name:             "Luna",
			Species:          "Dog",
			Breed:            "German Shepherd Mix",
			Age:              2,
			Description:      "Smart and loyal companion. Luna is learning her manners and would benefit from an experienced dog owner. She's very food motivated and loves training sessions with our volunteers.",
			Status:           "foster",
			ImageURL:         "https://images.unsplash.com/photo-1568572933382-74d440642117?w=800&q=80", // German Shepherd
			ArrivalDate:      &twentyDaysAgo,
			FosterStartDate:  &fiveDaysAgo,
			LastStatusChange: &fiveDaysAgo,
		},
		{
			GroupID:          modsquadGroupID,
			Name:             "Charlie",
			Species:          "Dog",
			Breed:            "Beagle",
			Age:              5,
			Description:      "Sweet beagle with a curious nose and endless enthusiasm for life. Charlie is calm, affectionate, and gets along well with everyone. Perfect family dog who loves gentle walks and cuddles.",
			Status:           "available",
			ImageURL:         "https://images.unsplash.com/photo-1505628346881-b72b27e84530?w=800&q=80", // Beagle
			ArrivalDate:      &fifteenDaysAgo,
			LastStatusChange: &fifteenDaysAgo,
		},
		{
			GroupID:          modsquadGroupID,
			Name:             "Max",
			Species:          "Dog",
			Breed:            "Labrador Retriever",
			Age:              3,
			Description:      "High-energy chocolate lab who needs plenty of exercise and mental stimulation. Max loves water, fetching, and being part of family activities. He's a loyal friend who will bring joy to any active home.",
			Status:           "available",
			ImageURL:         "https://images.unsplash.com/photo-1579270183931-b2fd69f83db4?w=800&q=80", // Chocolate Lab
			ArrivalDate:      &tenDaysAgo,
			LastStatusChange: &tenDaysAgo,
		},
		{
			GroupID:             modsquadGroupID,
			Name:                "Rocky",
			Species:             "Dog",
			Breed:               "Pit Bull Terrier",
			Age:                 4,
			Description:         "Currently in bite quarantine following an incident. Rocky is working with our behavior team and showing excellent progress with positive reinforcement training. Evaluation pending completion of quarantine period.",
			Status:              "bite_quarantine",
			ImageURL:            "https://images.unsplash.com/photo-1551717743-49959800b1f6?w=800&q=80", // Pit Bull
			ArrivalDate:         &twentyDaysAgo,
			QuarantineStartDate: &twoDaysAgo,
			LastStatusChange:    &twoDaysAgo,
		},
		{
			GroupID:          modsquadGroupID,
			Name:             "Daisy",
			Species:          "Dog",
			Breed:            "Border Collie Mix",
			Age:              3,
			Description:      "Incredibly intelligent and eager to please. Daisy excels at agility and loves learning new tricks. She needs an active family who can provide mental and physical stimulation daily.",
			Status:           "available",
			ImageURL:         "https://images.unsplash.com/photo-1587300003388-59208cc962cb?w=800&q=80", // Border Collie
			ArrivalDate:      &tenDaysAgo,
			LastStatusChange: &tenDaysAgo,
		},
		{
			GroupID:          modsquadGroupID,
			Name:             "Cooper",
			Species:          "Dog",
			Breed:            "Australian Shepherd",
			Age:              2,
			Description:      "Beautiful Australian Shepherd with stunning blue merle coat. Cooper is energetic, smart, and loves to work. He'd be perfect for hiking, running, or dog sports. Currently in foster care and thriving!",
			Status:           "foster",
			ImageURL:         "https://images.unsplash.com/photo-1568393691622-c7ba131d63b4?w=800&q=80", // Australian Shepherd
			ArrivalDate:      &fifteenDaysAgo,
			FosterStartDate:  &fiveDaysAgo,
			LastStatusChange: &fiveDaysAgo,
		},
		{
			GroupID:          modsquadGroupID,
			Name:             "Bella",
			Species:          "Dog",
			Breed:            "Husky Mix",
			Age:              4,
			Description:      "Gorgeous husky mix with striking blue eyes. Bella loves cool weather, long runs, and howling along to music. She's independent but affectionate and needs an experienced owner who understands the breed.",
			Status:           "available",
			ImageURL:         "https://images.unsplash.com/photo-1605568427561-40dd23c2acea?w=800&q=80", // Husky
			ArrivalDate:      &twentyDaysAgo,
			LastStatusChange: &twentyDaysAgo,
		},
		{
			GroupID:          modsquadGroupID,
			Name:             "Zeus",
			Species:          "Dog",
			Breed:            "Great Dane",
			Age:              5,
			Description:      "Gentle giant who thinks he's a lap dog! Zeus is calm, affectionate, and excellent with children. Despite his size, he has a moderate energy level and just wants to be near his people. House trained and crate trained.",
			Status:           "available",
			ImageURL:         "https://images.unsplash.com/photo-1534361960057-19889db9621e?w=800&q=80", // Great Dane
			ArrivalDate:      &fiveDaysAgo,
			LastStatusChange: &fiveDaysAgo,
		},
		{
			GroupID:          modsquadGroupID,
			Name:             "Rosie",
			Species:          "Dog",
			Breed:            "Corgi Mix",
			Age:              3,
			Description:      "Adorable corgi mix with short legs and a big personality! Rosie is playful, smart, and loves being the center of attention. She's great with kids and gets along well with other dogs.",
			Status:           "available",
			ImageURL:         "https://images.unsplash.com/photo-1612536409413-0e95d00c7ab5?w=800&q=80", // Corgi
			ArrivalDate:      &tenDaysAgo,
			LastStatusChange: &tenDaysAgo,
		},
	}

	// Fetch animal tags for assignment
	var (
		friendlyTag      models.AnimalTag
		shyTag           models.AnimalTag
		reactiveTag      models.AnimalTag
		resourceGuarding models.AnimalTag
		dualWalkerTag    models.AnimalTag
		experiencedOnly  models.AnimalTag
	)

	db.Where("name = ?", "friendly").First(&friendlyTag)
	db.Where("name = ?", "shy").First(&shyTag)
	db.Where("name = ?", "reactive").First(&reactiveTag)
	db.Where("name = ?", "resource guarding").First(&resourceGuarding)
	db.Where("name = ?", "dual walker").First(&dualWalkerTag)
	db.Where("name = ?", "experienced only").First(&experiencedOnly)

	// Assign tags to animals based on their characteristics
	animalTags := map[int][]models.AnimalTag{
		0: {friendlyTag},                    // Buddy - friendly golden retriever
		1: {experiencedOnly, dualWalkerTag}, // Luna - needs experienced walker, dual walker
		2: {friendlyTag},                    // Charlie - calm and friendly beagle
		3: {friendlyTag},                    // Max - high-energy but friendly lab
		4: {reactiveTag, experiencedOnly},   // Rocky - in bite quarantine
		5: {friendlyTag},                    // Daisy - intelligent and eager border collie
		6: {dualWalkerTag},                  // Cooper - energetic aussie shepherd
		7: {experiencedOnly},                // Bella - independent husky
		8: {friendlyTag},                    // Zeus - gentle giant
		9: {friendlyTag},                    // Rosie - playful corgi
	}

	for i := range animals {
		// Assign tags if they exist for this animal
		if tags, ok := animalTags[i]; ok {
			animals[i].Tags = tags
		}

		if err := db.Create(&animals[i]).Error; err != nil {
			return nil, err
		}
		logging.WithField("animal_name", animals[i].Name).Info("Created ModSquad demo animal")
	}

	return animals, nil
}

// seedComments creates demo comments on ModSquad animals
func seedComments(db *gorm.DB, users []models.User, animals []models.Animal) error {
	// Get comment tags
	var behaviorTag, medicalTag, generalTag models.CommentTag
	db.Where("name = ?", "behavior").First(&behaviorTag)
	db.Where("name = ?", "medical").First(&medicalTag)
	db.Where("name = ?", "general").First(&generalTag)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	twoDaysAgo := now.AddDate(0, 0, -2)
	threeDaysAgo := now.AddDate(0, 0, -3)
	fourDaysAgo := now.AddDate(0, 0, -4)

	// Generate comments for long-term animals (simulate 6+ months)
	var allComments []models.AnimalComment

	// Buddy - Long-term resident with 35+ comments over 6 months
	for i := 0; i < 35; i++ {
		daysAgo := 180 - (i * 5) // Spread over 6 months
		userIdx := (i % 3) + 1   // Rotate between merry, sophia, terry
		commentDate := now.AddDate(0, 0, -daysAgo)

		commentTexts := []string{
			"Buddy had a great walk today! He's so friendly with everyone we meet.",
			"Worked on recall training with Buddy. He's making excellent progress!",
			"Buddy enjoyed his playtime in the yard today. Still full of energy!",
			"Vet checkup went well. Buddy is healthy and ready for adoption!",
			"Buddy met some potential adopters today. They loved his gentle nature.",
			"Training session complete. Buddy is very food-motivated and eager to learn.",
			"Buddy had a bath today and looks so handsome! Great temperament during grooming.",
			"Socialization with other dogs went well. Buddy is very playful.",
			"Buddy's favorite toy is the tennis ball. He could fetch all day!",
			"Another wonderful day with Buddy. He deserves a loving forever home.",
		}

		comment := models.AnimalComment{
			AnimalID:  animals[0].ID,
			UserID:    users[userIdx].ID,
			Content:   commentTexts[i%len(commentTexts)],
			CreatedAt: commentDate,
		}

		if i%7 == 0 {
			comment.Tags = []models.CommentTag{behaviorTag}
		} else if i%11 == 0 {
			comment.Tags = []models.CommentTag{medicalTag}
		}

		allComments = append(allComments, comment)
	}

	// Rocky - Long-term resident with 40+ comments (behavioral rehabilitation)
	for i := 0; i < 40; i++ {
		daysAgo := 200 - (i * 5) // Spread over ~7 months
		userIdx := (i % 3) + 1
		commentDate := now.AddDate(0, 0, -daysAgo)

		commentTexts := []string{
			"Rocky showed great improvement in today's training session.",
			"Behavioral eval: Rocky is responding well to positive reinforcement.",
			"Rocky's confidence is growing every day. Wonderful progress!",
			"Worked on leash manners with Rocky. He's getting much better!",
			"Rocky had a calm, relaxed day today. His transformation is amazing.",
			"Socialization session went very well. Rocky is becoming more trusting.",
			"Rocky enjoys his puzzle toys. Great mental stimulation for him.",
			"Another successful training milestone for Rocky today!",
			"Rocky's gentle nature is really shining through now.",
			"So proud of Rocky's progress. He's ready for a patient, experienced home.",
		}

		comment := models.AnimalComment{
			AnimalID:  animals[4].ID, // Rocky
			UserID:    users[userIdx].ID,
			Content:   commentTexts[i%len(commentTexts)],
			CreatedAt: commentDate,
		}

		if i%5 == 0 {
			comment.Tags = []models.CommentTag{behaviorTag}
		}

		allComments = append(allComments, comment)
	}

	// Regular comments for other animals (with session report metadata)
	comments := []models.AnimalComment{
		{
			AnimalID:  animals[1].ID, // Luna
			UserID:    users[4].ID,   // alex
			Content:   "Luna is doing fantastic in her foster home! She's settling in beautifully and learning quickly.",
			CreatedAt: yesterday,
			Metadata: &models.SessionMetadata{
				SessionGoal:    "Loose leash walking in the neighborhood",
				SessionOutcome: "Maintained slack leash for 80% of the 20-minute walk",
				BehaviorNotes:  "Mild reactivity to bikes; recovered with treat scatter",
				SessionRating:  4,
				OtherNotes:     "Try front-clip harness for next walk",
			},
		},
		{
			AnimalID:  animals[2].ID, // Charlie
			UserID:    users[5].ID,   // jordan
			Content:   "Charlie is such a sweetheart! He's been getting along great with the other dogs during playtime.",
			CreatedAt: twoDaysAgo,
			Metadata: &models.SessionMetadata{
				SessionGoal:    "Calm greetings with new dogs",
				SessionOutcome: "Approached with loose body; held sit for 5 seconds",
				BehaviorNotes:  "Brief excitement; settled quickly with cue",
				SessionRating:  5,
				OtherNotes:     "Practice with volunteer dogs in quieter area",
			},
		},
		{
			AnimalID:  animals[3].ID, // Max
			UserID:    users[3].ID,   // terry
			Content:   "Took Max to the lake today for some swimming practice! He's a natural in the water.",
			CreatedAt: threeDaysAgo,
			Metadata: &models.SessionMetadata{
				SessionGoal:    "Energy outlet and recall near water",
				SessionOutcome: "Returned on first cue 4/5 times",
				BehaviorNotes:  "High arousal; improved with structured breaks",
				SessionRating:  4,
				OtherNotes:     "Use long-line for safety near shoreline",
			},
		},
		{
			AnimalID:  animals[5].ID, // Daisy
			UserID:    users[6].ID,   // casey
			Content:   "Daisy learned three new tricks today - she's incredibly smart! Would excel at agility.",
			Tags:      []models.CommentTag{behaviorTag},
			CreatedAt: yesterday,
			Metadata: &models.SessionMetadata{
				SessionGoal:    "Shaping focus during agility warmup",
				SessionOutcome: "Held focus through three obstacle reps",
				BehaviorNotes:  "High engagement; brief sniffing when new dogs entered",
				SessionRating:  5,
			},
		},
		{
			AnimalID:  animals[6].ID, // Cooper
			UserID:    users[7].ID,   // taylor
			Content:   "Cooper went to his foster home today! The family has a large yard and herding breed experience.",
			CreatedAt: now,
		},
		{
			AnimalID:  animals[7].ID, // Bella
			UserID:    users[4].ID,   // alex
			Content:   "Bella had a vet checkup today - everything looks great! She has lots of energy.",
			Tags:      []models.CommentTag{medicalTag},
			CreatedAt: twoDaysAgo,
			Metadata: &models.SessionMetadata{
				SessionGoal:    "Post-vet decompression walk",
				SessionOutcome: "Settled after 10 minutes; loose body language",
				MedicalNotes:   "Vet cleared for light exercise only",
				SessionRating:  3,
			},
		},
		{
			AnimalID:  animals[8].ID, // Zeus
			UserID:    users[5].ID,   // jordan
			Content:   "Zeus is the gentlest giant! Despite his size, he's so careful and just wants to cuddle.",
			CreatedAt: yesterday,
		},
		{
			AnimalID:  animals[8].ID, // Zeus (additional comment)
			UserID:    users[1].ID,   // merry
			Content:   "Had another great session with Zeus today! He's making excellent progress with his manners.",
			CreatedAt: twoDaysAgo,
			Metadata: &models.SessionMetadata{
				SessionGoal:    "Manners with visitors and calm settling",
				SessionOutcome: "Settled on mat within 3 minutes; remained for 10",
				BehaviorNotes:  "Seeks contact; redirects easily to mat cue",
				SessionRating:  5,
			},
		},
		{
			AnimalID:  animals[9].ID, // Rosie
			UserID:    users[6].ID,   // casey
			Content:   "Rosie is an absolute star! Her short legs and big personality have won everyone's hearts.",
			CreatedAt: fourDaysAgo,
		},
		{
			AnimalID:  animals[9].ID, // Rosie (additional comment)
			UserID:    users[7].ID,   // taylor
			Content:   "Took Rosie for a walk around the shelter grounds today. She's doing great with her leash manners!",
			CreatedAt: threeDaysAgo,
			Metadata: &models.SessionMetadata{
				SessionGoal:    "Loose-leash walking around shelter",
				SessionOutcome: "Minimal pulling; checked-in frequently",
				BehaviorNotes:  "Excitable greeting; improved with 5-second rule",
				SessionRating:  4,
			},
		},
	}

	// Combine all comments
	allComments = append(allComments, comments...)

	for i := range allComments {
		if err := db.Create(&allComments[i]).Error; err != nil {
			return err
		}
	}
	logging.WithField("total_comments", len(allComments)).Info("Created ModSquad demo comments")

	return nil
}

// seedUpdates creates demo ModSquad group updates
func seedUpdates(db *gorm.DB, users []models.User, groups []models.Group) error {
	var modsquadGroupID uint
	for _, g := range groups {
		if g.Name == "modsquad" {
			modsquadGroupID = g.ID
			break
		}
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	threeDaysAgo := now.AddDate(0, 0, -3)
	fiveDaysAgo := now.AddDate(0, 0, -5)
	oneWeekAgo := now.AddDate(0, 0, -7)

	updates := []models.Update{
		{
			GroupID:   modsquadGroupID,
			UserID:    users[1].ID, // merry
			Title:     "Amazing Adoption Weekend!",
			Content:   "What an incredible weekend for ModSquad! We had THREE successful adoptions - Ranger, Scout, and Pepper all found their forever homes! 🎉 Thank you to everyone who helped with meet and greets, applications, and home checks. Our teamwork made this possible. Let's keep this momentum going!",
			CreatedAt: threeDaysAgo,
		},
		{
			GroupID:   modsquadGroupID,
			UserID:    users[2].ID, // sophia
			Title:     "Training Workshop This Saturday",
			Content:   "Don't forget about our ModSquad training workshop this Saturday at 10am! We'll be working on loose-leash walking, recall commands, and proper greeting behaviors. All dogs welcome, regardless of skill level. Bring treats and water for your pup! See you there!",
			CreatedAt: yesterday,
		},
		{
			GroupID:   modsquadGroupID,
			UserID:    users[3].ID, // terry
			Title:     "New Foster Homes Needed!",
			Content:   "ModSquad is looking for experienced foster volunteers! We have several dogs coming in next month who need temporary homes while they await adoption. If you've fostered before or are interested in learning, please reach out. Training and supplies provided. Foster families save lives!",
			CreatedAt: fiveDaysAgo,
		},
		{
			GroupID:   modsquadGroupID,
			UserID:    users[1].ID, // merry
			Title:     "Fundraiser Success - Thank You!",
			Content:   "Our recent fundraising event was a huge success! We raised $3,500 for ModSquad medical expenses and supplies. Special thanks to everyone who donated, volunteered, and spread the word. This money will directly help dogs like Rocky get the behavioral support they need and provide medical care for incoming rescues. You all are amazing! ❤️",
			CreatedAt: oneWeekAgo,
		},
		{
			GroupID:   modsquadGroupID,
			UserID:    users[2].ID, // sophia
			Title:     "Volunteer Orientation Next Month",
			Content:   "Are you interested in joining the ModSquad team? We're hosting a volunteer orientation session on the first Saturday of next month! Learn about our mission, meet current volunteers, and discover how you can help. No experience necessary - just a love for dogs and willingness to learn. Email us to RSVP!",
			CreatedAt: now,
		},
		{
			GroupID:   modsquadGroupID,
			UserID:    users[3].ID, // terry
			Title:     "Dog Park Playdate Success!",
			Content:   "Yesterday's ModSquad playdate at the park was wonderful! Six of our dogs got to socialize and play together. Buddy, Max, and Daisy especially loved showing off their fetch skills. Thanks to everyone who came out - socialization is so important for our pups! Let's plan another one soon.",
			CreatedAt: yesterday,
		},
	}

	for i := range updates {
		if err := db.Create(&updates[i]).Error; err != nil {
			return err
		}
		logging.WithField("title", updates[i].Title).Info("Created ModSquad demo update")
	}

	return nil
}

// seedAnnouncements creates demo site-wide announcements for ModSquad
func seedAnnouncements(db *gorm.DB, users []models.User) error {
	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)
	fiveDaysAgo := now.AddDate(0, 0, -5)
	oneWeekAgo := now.AddDate(0, 0, -7)

	announcements := []models.Announcement{
		{
			UserID:    users[0].ID, // admin
			Title:     "Welcome to ModSquad Volunteer Portal!",
			Content:   "Welcome to the official ModSquad volunteer management portal! 🐕 This platform helps us coordinate dog care, share important updates, and stay connected as a team. You can view animal profiles, add comments, track foster placements, and stay informed about group activities. Feel free to explore all the features and reach out if you have any questions. Thank you for being part of ModSquad!",
			SendEmail: false,
			CreatedAt: oneWeekAgo,
		},
		{
			UserID:    users[0].ID, // admin
			Title:     "New Feature: Activity Feed",
			Content:   "We've just launched a new Activity Feed feature! You can now see all recent comments, status changes, and updates for your group in one convenient place. Check it out on your group page - it's a great way to stay up-to-date with everything happening at ModSquad.",
			SendEmail: false,
			CreatedAt: fiveDaysAgo,
		},
		{
			UserID:    users[0].ID, // admin
			Title:     "Important: Vet Appointment Reminders",
			Content:   "Reminder to all foster volunteers: Please ensure your foster dogs make it to their scheduled vet appointments. We'll send you email reminders 24 hours before each appointment. If you need to reschedule, please contact us at least 48 hours in advance. Thank you for your dedication to our pups' health!",
			SendEmail: false,
			CreatedAt: twoDaysAgo,
		},
		{
			UserID:    users[0].ID, // admin
			Title:     "Photo Day - Help Needed!",
			Content:   "We're planning a professional photo day for all our available dogs next month! High-quality photos significantly increase adoption rates. We need volunteers to help with dog prep, handling during photo sessions, and treats/rewards. If you have photography skills or just want to help, please sign up! Let's help our dogs put their best paw forward. 📸",
			SendEmail: false,
			CreatedAt: now,
		},
	}

	for i := range announcements {
		if err := db.Create(&announcements[i]).Error; err != nil {
			return err
		}
		logging.WithField("title", announcements[i].Title).Info("Created ModSquad demo announcement")
	}

	return nil
}

// seedProtocols creates demo protocols for ModSquad group
func seedProtocols(db *gorm.DB, users []models.User, groups []models.Group) error {
	// Find the ModSquad group (case-insensitive search)
	var modSquadGroup *models.Group
	for i := range groups {
		if groups[i].Name == "modsquad" || groups[i].Name == "ModSquad" {
			modSquadGroup = &groups[i]
			break
		}
	}

	if modSquadGroup == nil {
		logging.Warn("ModSquad group not found - skipping protocol seeding")
		return nil
	}

	protocols := []models.Protocol{
		{
			GroupID:    modSquadGroup.ID,
			Title:      "Daily Dog Care Routine",
			Content:    "Morning Care (7:00 AM - 9:00 AM):\n1. Check dog's overall health and behavior\n2. Provide fresh water in clean bowls\n3. Feed according to individual dietary plan (see animal profile)\n4. Take dog outside for bathroom break and short walk\n5. Clean any messes in kennel or living area\n\nMidday Care (12:00 PM - 2:00 PM):\n1. Brief bathroom break and playtime\n2. Provide fresh water if needed\n3. Monitor for any signs of distress or illness\n\nEvening Care (5:00 PM - 7:00 PM):\n1. Evening meal (if on twice-daily feeding schedule)\n2. Extended walk or play session (30-45 minutes)\n3. Fresh water and bathroom break\n4. Clean bowls and living area\n5. Log any concerns or observations in the system\n\nRemember: Every dog is unique! Always check their individual profile for specific dietary restrictions, medical needs, or behavioral notes.",
			OrderIndex: 1,
		},
		{
			GroupID:    modSquadGroup.ID,
			Title:      "Medication Administration",
			Content:    "Before Administering Medication:\n1. Verify you have the correct dog (check name tag and profile)\n2. Confirm the medication, dosage, and timing in the animal's profile\n3. Wash your hands thoroughly\n4. Gather supplies: medication, treats, pill pocket (if needed)\n\nAdministration Steps:\n1. Stay calm and speak in a soothing voice\n2. For pills: Hide in pill pocket, peanut butter, or cheese\n3. For liquids: Use provided syringe, aim for side of mouth\n4. Ensure dog swallows completely (gentle throat massage may help)\n5. Offer water immediately after\n6. Reward with praise and a healthy treat\n\nAfter Administration:\n1. Log medication given in the system immediately\n2. Note any difficulties or reactions\n3. Watch for 15-20 minutes for adverse reactions\n4. Contact vet immediately if vomiting, excessive drooling, or distress occurs\n\nIMPORTANT: Never skip or delay scheduled medications. If you cannot administer, notify a supervisor immediately.",
			OrderIndex: 2,
		},
		{
			GroupID:    modSquadGroup.ID,
			Title:      "New Foster Dog Intake",
			Content:    "First 24 Hours - Critical Adjustment Period:\n\n1. Quiet Introduction:\n- Bring dog to designated area away from other animals\n- Allow 30 minutes to decompress in quiet space\n- Provide water but wait 1-2 hours before first meal\n\n2. Initial Assessment:\n- Check for visible injuries or health concerns\n- Note temperament: nervous, friendly, fearful, energetic\n- Test basic commands: sit, stay, come\n- Observe bathroom habits and preferences\n\n3. Profile Setup:\n- Take clear photos (face, full body, any distinguishing marks)\n- Record all observations in the system\n- Note any supplies needed (specific food, toys, bedding)\n- Document initial weight\n\n4. First Week Guidelines:\n- Maintain consistent routine\n- Gradually introduce to other dogs (if applicable)\n- Monitor eating, drinking, and bathroom habits\n- Take notes on personality and quirks\n- Schedule vet appointment within 3-5 days\n\n5. Red Flags - Contact Supervisor Immediately:\n- Refusal to eat/drink for 24+ hours\n- Lethargy or unresponsiveness\n- Vomiting or diarrhea\n- Aggression toward people or other animals\n- Signs of injury or illness",
			OrderIndex: 3,
		},
		{
			GroupID:    modSquadGroup.ID,
			Title:      "Emergency Procedures",
			Content:    "In ANY emergency, remain calm and act quickly.\n\nSTEP 1: Assess the Situation\n- Is the dog breathing?\n- Is there visible injury or bleeding?\n- Is the dog conscious and responsive?\n- Are other animals or people at risk?\n\nSTEP 2: Immediate Actions\nFor Severe Bleeding:\n- Apply direct pressure with clean cloth\n- Elevate wound above heart if possible\n- Do not remove cloth if blood soaks through (add more on top)\n\nFor Choking:\n- Open mouth and look for visible obstruction\n- If visible, try to carefully remove\n- If not visible or cannot remove, perform Heimlich (small dogs: hold upside down, larger dogs: abdominal thrusts)\n\nFor Unconsciousness:\n- Check for breathing and pulse\n- Begin CPR if needed (30 compressions, 2 breaths)\n- Call emergency vet while performing CPR\n\nSTEP 3: Contact Emergency Services\nEmergency Vet: (555) 123-4567\nAfter Hours: (555) 123-4568\nSupervisor: [Contact from profile]\n\nSTEP 4: Transport\n- Use blanket or board as stretcher for injured dog\n- Keep dog warm\n- Minimize movement\n- Have someone call ahead to vet\n\nSTEP 5: Document\n- Take photos if safe to do so\n- Record time of incident\n- Note all actions taken\n- Update system ASAP after emergency is handled",
			OrderIndex: 4,
		},
		{
			GroupID:    modSquadGroup.ID,
			Title:      "Adoption Appointment Protocol",
			Content:    "Preparation (Day Before):\n1. Verify appointment in system\n2. Ensure dog is clean and groomed\n3. Prepare dog's profile printout with:\n   - Medical history\n   - Behavioral notes\n   - Dietary requirements\n   - Current medications\n4. Gather any supplies (collar, leash, toys) that will go with dog\n\nDay of Appointment:\n\n30 Minutes Before:\n- Bathroom break for dog\n- Check that meeting area is clean and prepared\n- Review adoption application notes\n- Have paperwork ready\n\nDuring Meeting:\n1. Allow adopters to approach dog at their pace\n2. Demonstrate dog's commands and behaviors\n3. Discuss dog's personality, quirks, and needs honestly\n4. Allow interaction time (15-30 minutes minimum)\n5. Answer all questions thoroughly\n6. If they have other pets, discuss slow introduction methods\n\nIf Adoption Proceeds:\n1. Complete all paperwork\n2. Provide copy of medical records\n3. Give dietary and medication instructions\n4. Provide emergency contact information\n5. Schedule follow-up check-in (1 week)\n6. Update dog's status in system immediately\n\nIf Adoption Doesn't Proceed:\n1. Thank adopters for their time\n2. Ask if they'd like to meet other dogs\n3. Update system with notes about the meeting\n4. No judgment - the right match is most important!\n\nRemember: Our goal is successful, lasting adoptions. It's better to wait for the right family than rush into a poor match.",
			OrderIndex: 5,
		},
	}

	for i := range protocols {
		if err := db.Create(&protocols[i]).Error; err != nil {
			return err
		}
		logging.WithField("title", protocols[i].Title).Info("Created ModSquad demo protocol")
	}

	return nil
}

// updateSiteSettings updates site-wide settings with Unsplash hero image for fresh databases
func updateSiteSettings(db *gorm.DB) error {
	// Only set the hero image if none has been configured yet (preserve any admin-uploaded image)
	var heroSetting models.SiteSetting
	if err := db.Where("key = ?", "hero_image_url").First(&heroSetting).Error; err == nil {
		if heroSetting.Value == "" {
			// Beautiful hero image of happy dogs for the home page
			heroSetting.Value = "https://images.unsplash.com/photo-1601758228041-f3b2795255f1?w=1920&q=80"
			if err := db.Save(&heroSetting).Error; err != nil {
				return fmt.Errorf("failed to update hero_image_url setting: %w", err)
			}
			logging.Info("Updated site-wide hero image with Unsplash image")
		} else {
			logging.Info("Skipping hero image update - custom image already configured")
		}
	}

	return nil
}
