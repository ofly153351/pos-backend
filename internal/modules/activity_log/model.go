package activity_log

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// JSONText is a jsonb-friendly column value. Empty → SQL NULL; non-empty is sent to
// the driver as a string so PostgreSQL stores it as real jsonb (a raw []byte would
// otherwise be encoded as bytea by pgx). It also round-trips as raw JSON in API
// responses via MarshalJSON, so the captured diff reaches the frontend untouched.
type JSONText []byte

func (j JSONText) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *JSONText) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*j = nil
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = []byte(v)
	default:
		return fmt.Errorf("activity_log: cannot scan %T into JSONText", src)
	}
	return nil
}

func (j JSONText) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

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
	// Changes is a nullable JSONB diff ({kind, fields:{name:{before,after}}}) captured
	// at write time for restore-eligible edits. Migration 054 adds the column.
	Changes   JSONText  `gorm:"type:jsonb"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (ActivityLog) TableName() string { return "activity_logs" }
