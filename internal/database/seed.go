package database

import (
	"fmt"
	"time"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/logging"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedData populates the database with demo data for testing and demonstrations
// If force is true, it will seed data even if users already exist
func SeedData(db *gorm.DB, force bool) error {
	logging.Info("Starting database seeding...")

	// Check if data already exists
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount > 0 && !force {
		logging.Info("Database already contains users - skipping seed data (use --force to override)")
		return nil
	}

	// Seed users
	users, err := seedUsers(db)
	if err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	// Get groups
	var groups []models.Group
	if err := db.Find(&groups).Error; err != nil {
		return fmt.Errorf("failed to fetch groups: %w", err)
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

	// Seed announcements
	if err := seedAnnouncements(db, users); err != nil {
		return fmt.Errorf("failed to seed announcements: %w", err)
	}

	logging.Info("Database seeding completed successfully")
	return nil
}

// seedUsers creates demo users
func seedUsers(db *gorm.DB) ([]models.User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	users := []models.User{
		{
			Username:                  "admin",
			Email:                     "admin@demo.local",
			Password:                  string(hashedPassword),
			IsAdmin:                   true,
			EmailNotificationsEnabled: true,
		},
		{
			Username:                  "sarah_volunteer",
			Email:                     "sarah@demo.local",
			Password:                  string(hashedPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: true,
		},
		{
			Username:                  "mike_foster",
			Email:                     "mike@demo.local",
			Password:                  string(hashedPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
		},
		{
			Username:                  "emma_cats",
			Email:                     "emma@demo.local",
			Password:                  string(hashedPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: true,
		},
		{
			Username:                  "jake_modsquad",
			Email:                     "jake@demo.local",
			Password:                  string(hashedPassword),
			IsAdmin:                   false,
			EmailNotificationsEnabled: false,
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

// assignUsersToGroups assigns demo users to appropriate groups
func assignUsersToGroups(db *gorm.DB, users []models.User, groups []models.Group) error {
	// Find specific groups
	var dogsGroup, catsGroup, modsquadGroup models.Group
	for _, g := range groups {
		switch g.Name {
		case "dogs":
			dogsGroup = g
		case "cats":
			catsGroup = g
		case "modsquad":
			modsquadGroup = g
		}
	}

	// Admin gets access to all groups
	if err := db.Model(&users[0]).Association("Groups").Append(&dogsGroup, &catsGroup, &modsquadGroup); err != nil {
		return err
	}

	// sarah_volunteer - dogs and modsquad
	if err := db.Model(&users[1]).Association("Groups").Append(&dogsGroup, &modsquadGroup); err != nil {
		return err
	}

	// mike_foster - dogs
	if err := db.Model(&users[2]).Association("Groups").Append(&dogsGroup); err != nil {
		return err
	}

	// emma_cats - cats
	if err := db.Model(&users[3]).Association("Groups").Append(&catsGroup); err != nil {
		return err
	}

	// jake_modsquad - modsquad
	if err := db.Model(&users[4]).Association("Groups").Append(&modsquadGroup); err != nil {
		return err
	}

	logging.Info("Assigned users to groups")
	return nil
}

// seedAnimals creates demo animals
func seedAnimals(db *gorm.DB, groups []models.Group) ([]models.Animal, error) {
	var dogsGroupID, catsGroupID uint
	for _, g := range groups {
		if g.Name == "dogs" {
			dogsGroupID = g.ID
		} else if g.Name == "cats" {
			catsGroupID = g.ID
		}
	}

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)
	fiveDaysAgo := now.AddDate(0, 0, -5)
	tenDaysAgo := now.AddDate(0, 0, -10)
	fifteenDaysAgo := now.AddDate(0, 0, -15)
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	animals := []models.Animal{
		{
			GroupID:          dogsGroupID,
			Name:             "Buddy",
			Species:          "Dog",
			Breed:            "Golden Retriever",
			Age:              4,
			Description:      "Friendly and energetic golden retriever who loves to play fetch. Great with kids and other dogs. House trained and knows basic commands.",
			Status:           "available",
			ArrivalDate:      &thirtyDaysAgo,
			LastStatusChange: &thirtyDaysAgo,
		},
		{
			GroupID:          dogsGroupID,
			Name:             "Luna",
			Species:          "Dog",
			Breed:            "German Shepherd Mix",
			Age:              2,
			Description:      "Smart and loyal companion. Luna is learning her manners and would benefit from an experienced dog owner. She's very food motivated and loves training.",
			Status:           "foster",
			ArrivalDate:      &fifteenDaysAgo,
			FosterStartDate:  &fiveDaysAgo,
			LastStatusChange: &fiveDaysAgo,
		},
		{
			GroupID:          dogsGroupID,
			Name:             "Charlie",
			Species:          "Dog",
			Breed:            "Beagle",
			Age:              5,
			Description:      "Sweet beagle with a curious nose. Charlie is calm, affectionate, and gets along well with everyone. Perfect family dog.",
			Status:           "available",
			ArrivalDate:      &tenDaysAgo,
			LastStatusChange: &tenDaysAgo,
		},
		{
			GroupID:          dogsGroupID,
			Name:             "Max",
			Species:          "Dog",
			Breed:            "Labrador Retriever",
			Age:              3,
			Description:      "High-energy lab who needs plenty of exercise and mental stimulation. Max loves water, fetching, and being part of family activities.",
			Status:           "available",
			ArrivalDate:      &fiveDaysAgo,
			LastStatusChange: &fiveDaysAgo,
		},
		{
			GroupID:          dogsGroupID,
			Name:             "Rocky",
			Species:          "Dog",
			Breed:            "Pit Bull Terrier",
			Age:              4,
			Description:      "Currently in bite quarantine following an incident. Rocky is working with our behavior team and showing good progress. Evaluation pending.",
			Status:           "bite_quarantine",
			ArrivalDate:      &fifteenDaysAgo,
			QuarantineStartDate: &twoDaysAgo,
			LastStatusChange: &twoDaysAgo,
		},
		{
			GroupID:          catsGroupID,
			Name:             "Mittens",
			Species:          "Cat",
			Breed:            "Domestic Shorthair",
			Age:              6,
			Description:      "Independent but affectionate cat who enjoys lap time on her own terms. Mittens is great for someone looking for a calm companion.",
			Status:           "available",
			ArrivalDate:      &thirtyDaysAgo,
			LastStatusChange: &thirtyDaysAgo,
		},
		{
			GroupID:          catsGroupID,
			Name:             "Oliver",
			Species:          "Cat",
			Breed:            "Maine Coon Mix",
			Age:              2,
			Description:      "Playful and social cat who loves interactive toys and attention. Oliver would do well in an active home with lots of playtime.",
			Status:           "foster",
			ArrivalDate:      &tenDaysAgo,
			FosterStartDate:  &twoDaysAgo,
			LastStatusChange: &twoDaysAgo,
		},
		{
			GroupID:          catsGroupID,
			Name:             "Whiskers",
			Species:          "Cat",
			Breed:            "Siamese",
			Age:              8,
			Description:      "Talkative and affectionate senior cat. Whiskers is looking for a quiet home where he can be the center of attention.",
			Status:           "available",
			ArrivalDate:      &fifteenDaysAgo,
			LastStatusChange: &fifteenDaysAgo,
		},
		{
			GroupID:          catsGroupID,
			Name:             "Luna (cat)",
			Species:          "Cat",
			Breed:            "Domestic Longhair",
			Age:              1,
			Description:      "Energetic kitten who loves to explore and play. Luna is social and would do well with another young cat or as an only pet with lots of attention.",
			Status:           "available",
			ArrivalDate:      &fiveDaysAgo,
			LastStatusChange: &fiveDaysAgo,
		},
	}

	for i := range animals {
		if err := db.Create(&animals[i]).Error; err != nil {
			return nil, err
		}
		logging.WithField("animal_name", animals[i].Name).Info("Created demo animal")
	}

	return animals, nil
}

// seedComments creates demo comments on animals
func seedComments(db *gorm.DB, users []models.User, animals []models.Animal) error {
	// Get comment tags
	var behaviorTag, medicalTag models.CommentTag
	db.Where("name = ?", "behavior").First(&behaviorTag)
	db.Where("name = ?", "medical").First(&medicalTag)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	threeDaysAgo := now.AddDate(0, 0, -3)

	comments := []models.AnimalComment{
		{
			AnimalID:  animals[0].ID, // Buddy
			UserID:    users[1].ID,   // sarah_volunteer
			Content:   "Buddy had a great walk today! He's very well-behaved on leash and loves meeting new people.",
			CreatedAt: yesterday,
		},
		{
			AnimalID:  animals[1].ID, // Luna (dog)
			UserID:    users[1].ID,   // sarah_volunteer
			Content:   "Luna is doing great in foster! She's settling in well and learning quickly. Her foster family reports she's housetrained and sleeps through the night.",
			CreatedAt: now,
		},
		{
			AnimalID:  animals[4].ID, // Rocky
			UserID:    users[4].ID,   // jake_modsquad
			Content:   "Rocky completed his first evaluation session today. He responded well to calm handling and showed no signs of aggression. Will continue monitoring.",
			CreatedAt: yesterday,
		},
		{
			AnimalID:  animals[5].ID, // Mittens
			UserID:    users[3].ID,   // emma_cats
			Content:   "Mittens seems a bit stressed today. Gave her some extra quiet time and she's doing better. Will monitor tomorrow.",
			Tags:      []models.CommentTag{behaviorTag},
			CreatedAt: threeDaysAgo,
		},
		{
			AnimalID:  animals[6].ID, // Oliver
			UserID:    users[3].ID,   // emma_cats
			Content:   "Oliver went to foster today! His foster family is experienced with Maine Coons and he seemed excited to explore his new home.",
			CreatedAt: now,
		},
		{
			AnimalID:  animals[7].ID, // Whiskers
			UserID:    users[3].ID,   // emma_cats
			Content:   "Whiskers has been sneezing a bit. Started him on lysine supplement. Vet check scheduled for tomorrow.",
			Tags:      []models.CommentTag{medicalTag},
			CreatedAt: yesterday,
		},
		{
			AnimalID:  animals[2].ID, // Charlie
			UserID:    users[2].ID,   // mike_foster
			Content:   "Charlie is such a sweetheart! He's been getting along great with the other dogs during playtime.",
			CreatedAt: now,
		},
	}

	for i := range comments {
		if err := db.Create(&comments[i]).Error; err != nil {
			return err
		}
		logging.WithField("animal_id", comments[i].AnimalID).Info("Created demo comment")
	}

	return nil
}

// seedUpdates creates demo group updates
func seedUpdates(db *gorm.DB, users []models.User, groups []models.Group) error {
	var dogsGroupID, catsGroupID uint
	for _, g := range groups {
		if g.Name == "dogs" {
			dogsGroupID = g.ID
		} else if g.Name == "cats" {
			catsGroupID = g.ID
		}
	}

	now := time.Now()
	twoDaysAgo := now.AddDate(0, 0, -2)
	fiveDaysAgo := now.AddDate(0, 0, -5)

	updates := []models.Update{
		{
			GroupID:   dogsGroupID,
			UserID:    users[1].ID, // sarah_volunteer
			Title:     "Great Adoption Weekend!",
			Content:   "We had three successful adoptions this weekend! Thank you to everyone who helped with meet and greets. Let's keep the momentum going!",
			CreatedAt: twoDaysAgo,
		},
		{
			GroupID:   dogsGroupID,
			UserID:    users[2].ID, // mike_foster
			Title:     "Reminder: Upcoming Training Session",
			Content:   "Don't forget about our group training session this Saturday at 10am. We'll be working on loose-leash walking and basic commands. All dogs welcome!",
			CreatedAt: now,
		},
		{
			GroupID:   catsGroupID,
			UserID:    users[3].ID, // emma_cats
			Title:     "Kitten Season Update",
			Content:   "We're expecting several litters in the coming weeks. If you're interested in fostering kittens, please let us know ASAP! We'll need bottles feeders and experienced foster homes.",
			CreatedAt: fiveDaysAgo,
		},
		{
			GroupID:   catsGroupID,
			UserID:    users[3].ID, // emma_cats
			Title:     "Cat Room Cleanup Day",
			Content:   "Thank you to everyone who came out for the cat room cleanup yesterday! The space looks amazing and the cats are loving their refreshed environment.",
			CreatedAt: now,
		},
	}

	for i := range updates {
		if err := db.Create(&updates[i]).Error; err != nil {
			return err
		}
		logging.WithField("title", updates[i].Title).Info("Created demo update")
	}

	return nil
}

// seedAnnouncements creates demo site-wide announcements
func seedAnnouncements(db *gorm.DB, users []models.User) error {
	now := time.Now()
	threeDaysAgo := now.AddDate(0, 0, -3)

	announcements := []models.Announcement{
		{
			UserID:    users[0].ID, // admin
			Title:     "Welcome to the Volunteer Portal!",
			Content:   "Thank you for being part of our volunteer community! This platform helps us coordinate animal care, share updates, and stay connected. Feel free to explore and reach out if you have any questions.",
			SendEmail: false,
			CreatedAt: threeDaysAgo,
		},
		{
			UserID:    users[0].ID, // admin
			Title:     "Holiday Schedule Change",
			Content:   "Please note that the shelter will have modified hours during the upcoming holiday weekend. Volunteer shifts have been adjusted accordingly. Check your email for your updated schedule.",
			SendEmail: true,
			CreatedAt: now,
		},
	}

	for i := range announcements {
		if err := db.Create(&announcements[i]).Error; err != nil {
			return err
		}
		logging.WithField("title", announcements[i].Title).Info("Created demo announcement")
	}

	return nil
}
