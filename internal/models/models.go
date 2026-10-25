package models

import "time"

// Admin represents a console operator. Email uniqueness is case-insensitive.
type Admin struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string    `gorm:"type:text;not null;uniqueIndex:idx_admins_email_lower,expression:lower(email)" json:"email"`
	PasswordHash string    `gorm:"type:text;not null" json:"-"`
	Active       bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (Admin) TableName() string { return "admins" }

// User is an external application's user reference, not a profile record.
type User struct {
	ID         int64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ExternalID string        `gorm:"type:text;not null;uniqueIndex:idx_users_external_id" json:"external_id"`
	CreatedAt  time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	Tokens     []DeviceToken `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (User) TableName() string { return "users" }

// DeviceToken stores one unique FCM token and its activation state.
type DeviceToken struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;index:idx_device_tokens_active_cursor,where:active = true" json:"id"`
	UserID    int64     `gorm:"not null;index:idx_device_tokens_user_id" json:"user_id"`
	Token     string    `gorm:"type:text;not null;uniqueIndex:idx_device_tokens_token" json:"token"`
	Platform  string    `gorm:"type:varchar(16);not null" json:"platform"`
	Active    bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (DeviceToken) TableName() string { return "device_tokens" }

// Notification contains the immutable content sent to every active token.
type Notification struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Title     string    `gorm:"type:text;not null;check:ck_notifications_title_length,char_length(title) BETWEEN 1 AND 120" json:"title"`
	Body      string    `gorm:"type:text;not null;check:ck_notifications_body_length,char_length(body) BETWEEN 1 AND 2000" json:"body"`
	CreatedBy int64     `gorm:"not null;index:idx_notifications_created_by" json:"created_by"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_notifications_created_at" json:"created_at"`
	Admin     Admin     `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
}

func (Notification) TableName() string { return "notifications" }

// NotificationJob stores durable worker state for a notification.
type NotificationJob struct {
	NotificationID int64        `gorm:"primaryKey;autoIncrement:false" json:"notification_id"`
	Status         string       `gorm:"type:varchar(16);not null;default:pending;check:ck_notification_jobs_status,status IN ('pending','processing','done','failed');index:idx_notification_jobs_claim,priority:1" json:"status"`
	HighWatermark  int64        `gorm:"not null;default:0" json:"high_watermark"`
	Cursor         int64        `gorm:"not null;default:0" json:"cursor"`
	Total          int64        `gorm:"not null;default:0" json:"total"`
	Sent           int64        `gorm:"not null;default:0" json:"sent"`
	Failed         int64        `gorm:"not null;default:0" json:"failed"`
	Invalid        int64        `gorm:"not null;default:0" json:"invalid"`
	Attempts       int          `gorm:"not null;default:0" json:"attempts"`
	NextAttemptAt  time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_notification_jobs_claim,priority:2" json:"next_attempt_at"`
	LeaseOwner     *string      `gorm:"type:text" json:"lease_owner,omitempty"`
	LeaseUntil     *time.Time   `gorm:"index:idx_notification_jobs_claim,priority:3" json:"lease_until,omitempty"`
	LastError      *string      `gorm:"type:text" json:"last_error,omitempty"`
	StartedAt      *time.Time   `json:"started_at,omitempty"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	Notification   Notification `gorm:"foreignKey:NotificationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (NotificationJob) TableName() string { return "notification_jobs" }

// NotificationBatch records each FCM call, including calls interrupted in flight.
type NotificationBatch struct {
	NotificationID int64        `gorm:"primaryKey;autoIncrement:false" json:"notification_id"`
	BatchNo        int64        `gorm:"primaryKey;autoIncrement:false" json:"batch_no"`
	TokenCount     int          `gorm:"not null;default:0" json:"token_count"`
	Success        int          `gorm:"not null;default:0" json:"success"`
	Failure        int          `gorm:"not null;default:0" json:"failure"`
	Invalid        int          `gorm:"not null;default:0" json:"invalid"`
	Status         string       `gorm:"type:varchar(16);not null;default:sending;check:ck_notification_batches_status,status IN ('sending','done','failed');index:idx_notification_batches_status" json:"status"`
	Error          *string      `gorm:"type:text" json:"error,omitempty"`
	StartedAt      time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP" json:"started_at"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	Notification   Notification `gorm:"foreignKey:NotificationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
}

func (NotificationBatch) TableName() string { return "notification_batches" }
