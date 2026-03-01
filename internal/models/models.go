package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Default site settings values - used by database migrations and as fallbacks
const (
	DefaultSiteName        = "MyHAWS"
	DefaultSiteShortName   = "MyHAWS"
	DefaultSiteDescription = "MyHAWS Volunteer Portal - Internal volunteer management system"
)

// User represents a user in the system
type User struct {
	ID                        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"-"`
	Username                  string         `gorm:"uniqueIndex;not null" json:"username"`
	FirstName                 string         `gorm:"default:''" json:"first_name"`
	LastName                  string         `gorm:"default:''" json:"last_name"`
	Email                     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password                  string         `gorm:"not null" json:"-"`
	IsAdmin                   bool           `gorm:"default:false" json:"is_admin"`
	PhoneNumber               string         `gorm:"default:''" json:"phone_number"`
	HideEmail                 bool           `gorm:"default:false" json:"hide_email"`        // User can hide email from non-admins
	HidePhoneNumber           bool           `gorm:"default:false" json:"hide_phone_number"` // User can hide phone from non-admins
	DefaultGroupID            *uint          `gorm:"index" json:"default_group_id"`
	Groups                    []Group        `gorm:"many2many:user_groups;" json:"groups,omitempty"`
	FailedLoginAttempts       int            `gorm:"default:0" json:"-"`
	LockedUntil               *time.Time     `json:"-"`
	LastLogin                 *time.Time     `json:"-"`
	ResetToken                string         `json:"-"`
	ResetTokenExpiry          *time.Time     `json:"-"`
	ResetTokenLookup          string         `gorm:"index;default:''" json:"-"` // Plaintext prefix for indexed token lookup
	SetupToken                string         `json:"-"`                         // Separate field for initial password setup (invite flow)
	SetupTokenExpiry          *time.Time     `json:"-"`
	SetupTokenLookup          string         `gorm:"index;default:''" json:"-"` // Plaintext prefix for indexed token lookup
	RequiresPasswordSetup     bool           `gorm:"default:false" json:"-"`    // Flag to prevent login before password setup
	EmailNotificationsEnabled bool           `gorm:"default:false" json:"email_notifications_enabled"`
	ShowLengthOfStay          bool           `gorm:"default:false" json:"show_length_of_stay"`
}

// Group represents a volunteer group (dogs, cats, modsquad, etc.)
type Group struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Name           string         `gorm:"uniqueIndex;not null" json:"name"`
	Description    string         `json:"description"`
	ImageURL       string         `json:"image_url"`
	HeroImageURL   string         `json:"hero_image_url"`
	HasProtocols   bool           `gorm:"column:has_protocols;default:false" json:"has_protocols"`     // Enable protocols feature for this group
	GroupMeBotID   string         `gorm:"column:groupme_bot_id" json:"groupme_bot_id"`                 // GroupMe Bot ID for sending messages
	GroupMeEnabled bool           `gorm:"column:groupme_enabled;default:false" json:"groupme_enabled"` // Enable GroupMe integration for this group
	Users          []User         `gorm:"many2many:user_groups;" json:"users,omitempty"`
	Animals        []Animal       `gorm:"foreignKey:GroupID" json:"animals,omitempty"`
	Updates        []Update       `gorm:"foreignKey:GroupID" json:"updates,omitempty"`
	Protocols      []Protocol     `gorm:"foreignKey:GroupID" json:"protocols,omitempty"`
}

// Animal represents an animal in a group
type Animal struct {
	ID                             uint                `gorm:"primaryKey" json:"id"`
	CreatedAt                      time.Time           `json:"created_at"`
	UpdatedAt                      time.Time           `json:"updated_at"`
	DeletedAt                      gorm.DeletedAt      `gorm:"index" json:"-"`
	GroupID                        uint                `gorm:"not null;index:idx_animal_group_status" json:"group_id"`
	Name                           string              `gorm:"not null" json:"name"`
	Species                        string              `json:"species"`
	Breed                          string              `json:"breed"`
	Age                            int                 `json:"age"`
	EstimatedBirthDate             *time.Time          `json:"estimated_birth_date"` // Estimated date of birth for real-time age calculation
	Description                    string              `json:"description"`
	TrainerNotes                   string              `json:"trainer_notes"` // Optional notes for trainer meetings
	ImageURL                       string              `json:"image_url"`
	Status                         string              `gorm:"default:'available';index:idx_animal_group_status" json:"status"` // available, foster, bite_quarantine, archived
	ArrivalDate                    *time.Time          `json:"arrival_date"`                                                    // When animal first became available
	FosterStartDate                *time.Time          `json:"foster_start_date"`                                               // When animal went to foster
	QuarantineStartDate            *time.Time          `json:"quarantine_start_date"`                                           // When bite quarantine started
	ArchivedDate                   *time.Time          `json:"archived_date"`                                                   // When animal was archived
	LastStatusChange               *time.Time          `json:"last_status_change"`                                              // Timestamp of last status change
	ReturnCount                    int                 `gorm:"default:0" json:"return_count"`                                   // Number of times animal has returned to shelter after being archived
	IsReturned                     bool                `gorm:"default:false" json:"is_returned"`                                // Indicates if archived animal is a return (not all archived animals are returns)
	ProtocolDocumentURL            string              `json:"protocol_document_url"`                                           // URL to protocol document (PDF/DOCX)
	ProtocolDocumentName           string              `json:"protocol_document_name"`                                          // Original filename of protocol document
	ProtocolDocumentData           []byte              `gorm:"type:bytea" json:"-"`                                             // Binary data of protocol document (null when using Azure)
	ProtocolDocumentType           string              `json:"protocol_document_type"`                                          // MIME type of protocol document
	ProtocolDocumentSize           int                 `json:"protocol_document_size"`                                          // Size in bytes
	ProtocolDocumentUserID         *uint               `json:"protocol_document_user_id"`                                       // User who uploaded the protocol document
	ProtocolDocumentProvider       string              `gorm:"default:'postgres'" json:"-"`                                     // Storage backend: "postgres" or "azure"
	ProtocolDocumentBlobIdentifier string              `json:"-"`                                                               // Azure blob identifier (UUID without extension)
	ProtocolDocumentBlobExtension  string              `json:"-"`                                                               // File extension (e.g., ".pdf", ".docx") for blob storage
	Tags                           []AnimalTag         `gorm:"many2many:animal_animal_tags;" json:"tags,omitempty"`             // Tags associated with this animal
	NameHistory                    []AnimalNameHistory `gorm:"foreignKey:AnimalID" json:"name_history,omitempty"`               // History of name changes for this animal
	Images                         []AnimalImage       `gorm:"foreignKey:AnimalID" json:"images,omitempty"`                     // Images uploaded for this animal
}

// AgeDisplay computes the animal's age in years and months from EstimatedBirthDate.
// Falls back to (Age, 0) when EstimatedBirthDate is nil.
func (a *Animal) AgeDisplay() (years int, months int) {
	if a.EstimatedBirthDate == nil {
		return a.Age, 0
	}
	now := time.Now()
	bd := *a.EstimatedBirthDate
	years = now.Year() - bd.Year()
	months = int(now.Month()) - int(bd.Month())
	if now.Day() < bd.Day() {
		months--
	}
	if months < 0 {
		years--
		months += 12
	}
	if years < 0 {
		years = 0
		months = 0
	}
	return years, months
}

// AgeYearsFromBirthDate computes whole years from EstimatedBirthDate for backward compatibility.
func (a *Animal) AgeYearsFromBirthDate() int {
	y, _ := a.AgeDisplay()
	return y
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
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `gorm:"index:idx_update_group_created" json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	GroupID     uint           `gorm:"not null;index:idx_update_group_created" json:"group_id"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	Title       string         `gorm:"not null" json:"title"`
	Content     string         `gorm:"not null" json:"content"`
	ImageURL    string         `json:"image_url"`
	SendEmail   bool           `gorm:"default:false" json:"send_email"` // Records whether email dispatch was requested at creation time
	SendGroupMe bool           `gorm:"default:false" json:"send_groupme"`
	User        User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Announcement represents a site-wide announcement/update
type Announcement struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `gorm:"index:idx_announcement_created" json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	UserID      uint           `gorm:"not null;index" json:"user_id"`
	Title       string         `gorm:"not null" json:"title"`
	Content     string         `gorm:"not null" json:"content"`
	SendEmail   bool           `gorm:"default:false" json:"send_email"`
	SendGroupMe bool           `gorm:"default:false" json:"send_groupme"`
	User        User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// AnimalComment represents a comment on an animal (social media style)
type AnimalComment struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time        `gorm:"index:idx_comment_animal_created" json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt gorm.DeletedAt   `gorm:"index" json:"-"`
	AnimalID  uint             `gorm:"not null;index:idx_comment_animal_created" json:"animal_id"`
	UserID    uint             `gorm:"not null;index" json:"user_id"`
	Content   string           `gorm:"not null" json:"content"`
	ImageURL  string           `json:"image_url"`
	Metadata  *SessionMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	Tags      []CommentTag     `gorm:"many2many:animal_comment_tags;" json:"tags,omitempty"`
	User      User             `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// CommentHistory stores the history of comment edits
// Each entry is a snapshot of a previous version of the comment
type CommentHistory struct {
	ID        uint             `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time        `gorm:"index:idx_comment_history_comment" json:"created_at"`
	CommentID uint             `gorm:"not null;index:idx_comment_history_comment" json:"comment_id"`
	Content   string           `gorm:"not null" json:"content"`
	ImageURL  string           `json:"image_url"`
	Metadata  *SessionMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	EditedBy  uint             `gorm:"not null" json:"edited_by"` // User who authored this historical version
	User      User             `gorm:"foreignKey:EditedBy" json:"user,omitempty"`
}

// SessionMetadata stores structured session report data
type SessionMetadata struct {
	SessionGoal      string `json:"session_goal,omitempty"`
	SessionOutcome   string `json:"session_outcome,omitempty"`
	BehaviorNotes    string `json:"behavior_notes,omitempty"`
	MedicalNotes     string `json:"medical_notes,omitempty"`
	SessionRating    int    `json:"session_rating,omitempty"` // 1-5 (Poor, Fair, Okay, Good, Great)
	OtherNotes       string `json:"other_notes,omitempty"`
	SessionStartTime string `json:"session_start_time,omitempty"` // "HH:MM" 24-hour format
	SessionEndTime   string `json:"session_end_time,omitempty"`   // "HH:MM" 24-hour format
}

// Scan implements sql.Scanner interface to convert database value to SessionMetadata
func (sm *SessionMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &sm)
}

// Value implements driver.Valuer interface to convert SessionMetadata to database value
func (sm *SessionMetadata) Value() (driver.Value, error) {
	if sm == nil {
		return nil, nil
	}
	return json.Marshal(sm)
}

// CommentTag represents a tag that can be applied to comments
// Tags are group-specific - each group has its own set of tags
type CommentTag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	GroupID   uint           `gorm:"index;uniqueIndex:idx_comment_tag_group_name" json:"group_id"` // Group this tag belongs to - NOT NULL enforced via raw SQL after migration
	Name      string         `gorm:"not null;uniqueIndex:idx_comment_tag_group_name" json:"name"`
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
	GroupID    uint           `gorm:"not null;index:idx_protocols_group_order" json:"group_id"`
	Title      string         `gorm:"not null" json:"title"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	ImageURL   string         `json:"image_url"`
	OrderIndex int            `gorm:"default:0;index:idx_protocols_group_order" json:"order_index"` // For custom ordering
}

// AnimalTag represents a tag that can be applied to animals
// Tags are group-specific - each group has its own set of tags
type AnimalTag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	GroupID   uint           `gorm:"index;uniqueIndex:idx_animal_tag_group_name" json:"group_id"` // Group this tag belongs to - NOT NULL enforced via raw SQL after migration
	Name      string         `gorm:"not null;uniqueIndex:idx_animal_tag_group_name" json:"name"`
	Category  string         `gorm:"not null" json:"category"`       // "behavior" or "walker_status"
	Color     string         `gorm:"default:'#6b7280'" json:"color"` // Hex color for UI display
}

// AnimalImage represents an image uploaded for an animal
type AnimalImage struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	AnimalID         *uint          `gorm:"index:idx_animal_image_animal;index:idx_animal_images_profile" json:"animal_id"` // Nullable for unlinked images
	UserID           uint           `gorm:"not null;index" json:"user_id"`
	ImageURL         string         `gorm:"not null" json:"image_url"`
	ImageData        []byte         `gorm:"type:bytea" json:"-"`           // Binary image data stored in DB (null when using Azure)
	MimeType         string         `gorm:"default:'image/jpeg'" json:"-"` // MIME type of the image
	Caption          string         `json:"caption"`
	IsProfilePicture bool           `gorm:"default:false;index:idx_animal_images_profile" json:"is_profile_picture"`
	Width            int            `json:"width"`
	Height           int            `json:"height"`
	FileSize         int            `json:"file_size"`                   // in bytes
	StorageProvider  string         `gorm:"default:'postgres'" json:"-"` // Storage backend: "postgres" or "azure"
	BlobIdentifier   string         `json:"-"`                           // Azure blob identifier (UUID without extension)
	BlobExtension    string         `json:"-"`                           // File extension (e.g., ".jpg", ".png") for blob storage
	User             User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Animal           Animal         `gorm:"foreignKey:AnimalID" json:"animal,omitempty"`
}

// AnimalNameHistory tracks name changes for an animal
type AnimalNameHistory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index:idx_name_history_animal" json:"created_at"`
	AnimalID  uint      `gorm:"not null;index:idx_name_history_animal" json:"animal_id"`
	OldName   string    `gorm:"not null" json:"old_name"`
	NewName   string    `gorm:"not null" json:"new_name"`
	ChangedBy uint      `gorm:"not null" json:"changed_by"` // User ID who made the change
}

// UserGroup represents the many-to-many relationship between users and groups
// with additional fields for group-level permissions
type UserGroup struct {
	UserID       uint      `gorm:"primaryKey;index:idx_user_groups_user_admin" json:"user_id"`
	GroupID      uint      `gorm:"primaryKey;index:idx_user_groups_group_id" json:"group_id"`
	CreatedAt    time.Time `json:"created_at"`
	IsGroupAdmin bool      `gorm:"default:false;index:idx_user_groups_user_admin" json:"is_group_admin"` // User has admin privileges for this specific group
	User         User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Group        Group     `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}
