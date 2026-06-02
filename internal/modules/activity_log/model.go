package activity_log

import "time"

type ActivityLog struct {
	ID         string    `gorm:"primaryKey;type:varchar(30)"`
	StoreID    string    `gorm:"type:varchar(30);not null;index"`
	UserID     string    `gorm:"type:varchar(30);not null;default:''"`
	UserName   string    `gorm:"type:varchar(255);not null;default:''"`
	Action     string    `gorm:"type:varchar(100);not null;default:''"` // create, update, delete, pay, cancel, …
	Module     string    `gorm:"type:varchar(100);not null;default:''"` // product, sale, document, …
	ResourceID string    `gorm:"type:varchar(30);not null;default:''"`
	Method     string    `gorm:"type:varchar(10);not null;default:''"`
	Path       string    `gorm:"type:text;not null;default:''"`
	IPAddress  string    `gorm:"type:varchar(50);not null;default:''"`
	CreatedAt  time.Time `gorm:"not null;default:now()"`
}

func (ActivityLog) TableName() string { return "activity_logs" }
