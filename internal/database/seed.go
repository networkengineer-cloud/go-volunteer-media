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

// Seed announcements
if err := seedAnnouncements(db, users); err != nil {
return fmt.Errorf("failed to seed announcements: %w", err)
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
// Hash password (minimum 8 characters for frontend validation)
hashedPassword, err := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
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
Username:                  "sarah_modsquad",
Email:                     "sarah@demo.local",
Password:                  string(hashedPassword),
IsAdmin:                   false,
EmailNotificationsEnabled: true,
},
{
Username:                  "mike_modsquad",
Email:                     "mike@demo.local",
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
{
Username:                  "lisa_modsquad",
Email:                     "lisa@demo.local",
Password:                  string(hashedPassword),
IsAdmin:                   false,
EmailNotificationsEnabled: true,
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
func assignUsersToGroups(db *gorm.DB, users []models.User, groups []models.Group) error {
// Find ModSquad group (primary group for demo)
var modsquadGroup models.Group
for _, g := range groups {
if g.Name == "modsquad" {
modsquadGroup = g
break
}
}

// All users get access to ModSquad (primary group for first few months)
for i := range users {
if err := db.Model(&users[i]).Association("Groups").Append(&modsquadGroup); err != nil {
return err
}
}

logging.Info("Assigned all users to ModSquad group")
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
GroupID:          modsquadGroupID,
Name:             "Rocky",
Species:          "Dog",
Breed:            "Pit Bull Terrier",
Age:              4,
Description:      "Currently in bite quarantine following an incident. Rocky is working with our behavior team and showing excellent progress with positive reinforcement training. Evaluation pending completion of quarantine period.",
Status:           "bite_quarantine",
ImageURL:         "https://images.unsplash.com/photo-1551717743-49959800b1f6?w=800&q=80", // Pit Bull
ArrivalDate:      &twentyDaysAgo,
QuarantineStartDate: &twoDaysAgo,
LastStatusChange: &twoDaysAgo,
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

for i := range animals {
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

comments := []models.AnimalComment{
{
AnimalID:  animals[0].ID, // Buddy (Golden Retriever)
UserID:    users[1].ID,   // sarah_modsquad
Content:   "Buddy had an amazing walk today! He's so well-behaved on leash and absolutely loves meeting new people at the park. Several people asked about adopting him!",
CreatedAt: yesterday,
},
{
AnimalID:  animals[1].ID, // Luna (German Shepherd Mix)
UserID:    users[2].ID,   // mike_modsquad
Content:   "Luna is doing fantastic in her foster home! She's settling in beautifully and learning quickly. Foster family reports she's fully housetrained and sleeps through the night without any issues.",
CreatedAt: now,
},
{
AnimalID:  animals[4].ID, // Rocky (Pit Bull)
UserID:    users[3].ID,   // jake_modsquad
Content:   "Rocky completed his first behavioral evaluation session today. He responded wonderfully to calm handling and showed no signs of aggression. Continuing with positive reinforcement training. Great progress!",
Tags:      []models.CommentTag{behaviorTag},
CreatedAt: yesterday,
},
{
AnimalID:  animals[2].ID, // Charlie (Beagle)
UserID:    users[4].ID,   // lisa_modsquad
Content:   "Charlie is such a sweetheart! He's been getting along great with the other dogs during playtime. His gentle nature makes him perfect for a family with kids.",
CreatedAt: twoDaysAgo,
},
{
AnimalID:  animals[3].ID, // Max (Lab)
UserID:    users[1].ID,   // sarah_modsquad
Content:   "Took Max to the lake today for some swimming practice! He's a natural in the water and had an absolute blast. He definitely needs an active family who can give him plenty of exercise.",
CreatedAt: threeDaysAgo,
},
{
AnimalID:  animals[5].ID, // Daisy (Border Collie Mix)
UserID:    users[2].ID,   // mike_modsquad
Content:   "Daisy learned three new tricks today - she's incredibly smart! Working on frisbee catching next. This girl would excel at agility competitions.",
Tags:      []models.CommentTag{behaviorTag},
CreatedAt: yesterday,
},
{
AnimalID:  animals[6].ID, // Cooper (Australian Shepherd)
UserID:    users[4].ID,   // lisa_modsquad
Content:   "Cooper went to his foster home today! The foster family has a large yard and lots of experience with herding breeds. He seemed excited to explore his new space!",
CreatedAt: now,
},
{
AnimalID:  animals[7].ID, // Bella (Husky)
UserID:    users[3].ID,   // jake_modsquad
Content:   "Bella had a vet checkup today - everything looks great! Her coat is shiny and healthy. She does have a lot of energy and would benefit from long daily runs or hikes.",
Tags:      []models.CommentTag{medicalTag},
CreatedAt: twoDaysAgo,
},
{
AnimalID:  animals[8].ID, // Zeus (Great Dane)
UserID:    users[1].ID,   // sarah_modsquad
Content:   "Zeus is the gentlest giant! Despite his massive size, he's so careful around people and just wants to cuddle. He tried to sit on my lap during training - it was adorable!",
CreatedAt: yesterday,
},
{
AnimalID:  animals[9].ID, // Rosie (Corgi)
UserID:    users[2].ID,   // mike_modsquad
Content:   "Rosie is an absolute star! She's great with commands and loves showing off her tricks. Her short little legs and big personality have won everyone's hearts here at ModSquad.",
CreatedAt: fourDaysAgo,
},
{
AnimalID:  animals[0].ID, // Buddy (Golden Retriever)
UserID:    users[3].ID,   // jake_modsquad
Content:   "Buddy's training continues to impress. He now knows sit, stay, down, come, and leave it. He's ready for his forever home!",
Tags:      []models.CommentTag{behaviorTag},
CreatedAt: threeDaysAgo,
},
{
AnimalID:  animals[4].ID, // Rocky (Pit Bull)
UserID:    users[4].ID,   // lisa_modsquad
Content:   "Rocky had another great session today. He's showing more confidence and trust. His tail wags are coming more frequently now - it's beautiful to see his transformation!",
Tags:      []models.CommentTag{behaviorTag},
CreatedAt: now,
},
}

for i := range comments {
if err := db.Create(&comments[i]).Error; err != nil {
return err
}
logging.WithField("animal_id", comments[i].AnimalID).Info("Created ModSquad demo comment")
}

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
UserID:    users[1].ID, // sarah_modsquad
Title:     "Amazing Adoption Weekend!",
Content:   "What an incredible weekend for ModSquad! We had THREE successful adoptions - Ranger, Scout, and Pepper all found their forever homes! 🎉 Thank you to everyone who helped with meet and greets, applications, and home checks. Our teamwork made this possible. Let's keep this momentum going!",
CreatedAt: threeDaysAgo,
},
{
GroupID:   modsquadGroupID,
UserID:    users[2].ID, // mike_modsquad
Title:     "Training Workshop This Saturday",
Content:   "Don't forget about our ModSquad training workshop this Saturday at 10am! We'll be working on loose-leash walking, recall commands, and proper greeting behaviors. All dogs welcome, regardless of skill level. Bring treats and water for your pup! See you there!",
CreatedAt: yesterday,
},
{
GroupID:   modsquadGroupID,
UserID:    users[3].ID, // jake_modsquad
Title:     "New Foster Homes Needed!",
Content:   "ModSquad is looking for experienced foster volunteers! We have several dogs coming in next month who need temporary homes while they await adoption. If you've fostered before or are interested in learning, please reach out. Training and supplies provided. Foster families save lives!",
CreatedAt: fiveDaysAgo,
},
{
GroupID:   modsquadGroupID,
UserID:    users[4].ID, // lisa_modsquad
Title:     "Fundraiser Success - Thank You!",
Content:   "Our recent fundraising event was a huge success! We raised $3,500 for ModSquad medical expenses and supplies. Special thanks to everyone who donated, volunteered, and spread the word. This money will directly help dogs like Rocky get the behavioral support they need and provide medical care for incoming rescues. You all are amazing! ❤️",
CreatedAt: oneWeekAgo,
},
{
GroupID:   modsquadGroupID,
UserID:    users[1].ID, // sarah_modsquad
Title:     "Volunteer Orientation Next Month",
Content:   "Are you interested in joining the ModSquad team? We're hosting a volunteer orientation session on the first Saturday of next month! Learn about our mission, meet current volunteers, and discover how you can help. No experience necessary - just a love for dogs and willingness to learn. Email us to RSVP!",
CreatedAt: now,
},
{
GroupID:   modsquadGroupID,
UserID:    users[2].ID, // mike_modsquad
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
SendEmail: true,
CreatedAt: twoDaysAgo,
},
{
UserID:    users[0].ID, // admin
Title:     "Photo Day - Help Needed!",
Content:   "We're planning a professional photo day for all our available dogs next month! High-quality photos significantly increase adoption rates. We need volunteers to help with dog prep, handling during photo sessions, and treats/rewards. If you have photography skills or just want to help, please sign up! Let's help our dogs put their best paw forward. 📸",
SendEmail: true,
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

// updateSiteSettings updates site-wide settings with Unsplash hero image
func updateSiteSettings(db *gorm.DB) error {
// Update hero image for home page
var heroSetting models.SiteSetting
if err := db.Where("key = ?", "hero_image_url").First(&heroSetting).Error; err == nil {
// Beautiful hero image of happy dogs for the home page
heroSetting.Value = "https://images.unsplash.com/photo-1601758228041-f3b2795255f1?w=1920&q=80"
if err := db.Save(&heroSetting).Error; err != nil {
return fmt.Errorf("failed to update hero_image_url setting: %w", err)
}
logging.Info("Updated site-wide hero image with Unsplash image")
}

return nil
}
