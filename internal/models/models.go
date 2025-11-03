package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID                        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"-"`
	Username                  string         `gorm:"uniqueIndex;not null" json:"username"`
	Email                     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password                  string         `gorm:"not null" json:"-"`
	IsAdmin                   bool           `gorm:"default:false" json:"is_admin"`
	DefaultGroupID            *uint          `gorm:"index" json:"default_group_id"`
	Groups                    []Group        `gorm:"many2many:user_groups;" json:"groups,omitempty"`
	FailedLoginAttempts       int            `gorm:"default:0" json:"-"`
	LockedUntil               *time.Time     `json:"-"`
	ResetToken                string         `json:"-"`
	ResetTokenExpiry          *time.Time     `json:"-"`
	EmailNotificationsEnabled bool           `gorm:"default:false" json:"email_notifications_enabled"`
}

// Group represents a volunteer group (dogs, cats, modsquad, etc.)
type Group struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"uniqueIndex;not null" json:"name"`
	Description  string         `json:"description"`
	ImageURL     string         `json:"image_url"`
	HeroImageURL string         `json:"hero_image_url"`
	HasProtocols bool           `gorm:"default:false" json:"has_protocols"` // Enable protocols feature for this group
	Users        []User         `gorm:"many2many:user_groups;" json:"users,omitempty"`
	Animals      []Animal       `gorm:"foreignKey:GroupID" json:"animals,omitempty"`
	Updates      []Update       `gorm:"foreignKey:GroupID" json:"updates,omitempty"`
	Protocols    []Protocol     `gorm:"foreignKey:GroupID" json:"protocols,omitempty"`
}

// Animal represents an animal in a group
type Animal struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
	GroupID             uint           `gorm:"not null;index:idx_animal_group_status" json:"group_id"`
	Name                string         `gorm:"not null" json:"name"`
	Species             string         `json:"species"`
	Breed               string         `json:"breed"`
	Age                 int            `json:"age"`
	Description         string         `json:"description"`
	ImageURL            string         `json:"image_url"`
	Status              string         `gorm:"default:'available';index:idx_animal_group_status" json:"status"` // available, foster, bite_quarantine, archived
	ArrivalDate         *time.Time     `json:"arrival_date"`                                                     // When animal first became available
	FosterStartDate     *time.Time     `json:"foster_start_date"`                                                // When animal went to foster
	QuarantineStartDate *time.Time     `json:"quarantine_start_date"`                                            // When bite quarantine started
	ArchivedDate        *time.Time     `json:"archived_date"`                                                    // When animal was archived
	LastStatusChange    *time.Time     `json:"last_status_change"`                                               // Timestamp of last status change
}

// LengthOfStay returns the number of days since the animal's arrival date
func (a *Animal) LengthOfStay() int {
	if a.ArrivalDate == nil {
		return 0
	}
	return int(time.Since(*a.ArrivalDate).Hours() / 24)
}

// CurrentStatusDuration returns the number of days since the last status change
func (a *Animal) CurrentStatusDuration() int {
	if a.LastStatusChange == nil {
		return 0
	}
	return int(time.Since(*a.LastStatusChange).Hours() / 24)
}

// QuarantineEndDate calculates when the 10-day bite quarantine ends
// The quarantine cannot end on Saturday or Sunday, so it adjusts forward to Monday
func (a *Animal) QuarantineEndDate() *time.Time {
	if a.QuarantineStartDate == nil {
		return nil
	}
	
	// Calculate 10 days from start date
	endDate := a.QuarantineStartDate.AddDate(0, 0, 10)
	
	// Check if end date falls on weekend and adjust to next Monday
	for endDate.Weekday() == time.Saturday || endDate.Weekday() == time.Sunday {
		endDate = endDate.AddDate(0, 0, 1)
	}
	
	return &endDate
}

// Update represents a post/update in a group
type Update struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"index:idx_update_group_created" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	GroupID   uint           `gorm:"not null;index:idx_update_group_created" json:"group_id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Title     string         `gorm:"not null" json:"title"`
	Content   string         `gorm:"not null" json:"content"`
	ImageURL  string         `json:"image_url"`
	User      User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Announcement represents a site-wide announcement/update
type Announcement struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"index:idx_announcement_created" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Title     string         `gorm:"not null" json:"title"`
	Content   string         `gorm:"not null" json:"content"`
	SendEmail bool           `gorm:"default:false" json:"send_email"`
	User      User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// AnimalComment represents a comment on an animal (social media style)
type AnimalComment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"index:idx_comment_animal_created" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	AnimalID  uint           `gorm:"not null;index:idx_comment_animal_created" json:"animal_id"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Content   string         `gorm:"not null" json:"content"`
	ImageURL  string         `json:"image_url"`
	Tags      []CommentTag   `gorm:"many2many:animal_comment_tags;" json:"tags,omitempty"`
	User      User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// CommentTag represents a tag that can be applied to comments
type CommentTag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"uniqueIndex;not null" json:"name"`
	Color     string         `gorm:"default:'#6b7280'" json:"color"` // Hex color for UI display
	IsSystem  bool           `gorm:"default:false" json:"is_system"` // True for behavior/medical tags
}

// SiteSetting represents configurable site settings
type SiteSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Key       string    `gorm:"uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
}

// Protocol represents a protocol/procedure for a group
type Protocol struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	GroupID    uint           `gorm:"not null;index" json:"group_id"`
	Title      string         `gorm:"not null" json:"title"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	ImageURL   string         `json:"image_url"`
	OrderIndex int            `gorm:"default:0" json:"order_index"` // For custom ordering
}
