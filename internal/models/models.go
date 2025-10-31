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
	Groups                    []Group        `gorm:"many2many:user_groups;" json:"groups,omitempty"`
	FailedLoginAttempts       int            `gorm:"default:0" json:"-"`
	LockedUntil               *time.Time     `json:"-"`
	ResetToken                string         `json:"-"`
	ResetTokenExpiry          *time.Time     `json:"-"`
	EmailNotificationsEnabled bool           `gorm:"default:false" json:"email_notifications_enabled"`
}

// Group represents a volunteer group (dogs, cats, modsquad, etc.)
type Group struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	Users       []User         `gorm:"many2many:user_groups;" json:"users,omitempty"`
	Animals     []Animal       `gorm:"foreignKey:GroupID" json:"animals,omitempty"`
	Updates     []Update       `gorm:"foreignKey:GroupID" json:"updates,omitempty"`
}

// Animal represents an animal in a group
type Animal struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	GroupID     uint           `gorm:"not null;index:idx_animal_group_status" json:"group_id"`
	Name        string         `gorm:"not null" json:"name"`
	Species     string         `json:"species"`
	Breed       string         `json:"breed"`
	Age         int            `json:"age"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	Status      string         `gorm:"default:'available';index:idx_animal_group_status" json:"status"` // available, adopted, fostered
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
