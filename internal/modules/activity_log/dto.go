package activity_log

import (
	"encoding/json"
	"time"
)

type LogEntry struct {
	ID         string    `json:"id"`
	StoreID    string    `json:"store_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name"`
	Action     string    `json:"action"`
	Module     string    `json:"module"`
	ResourceID string    `json:"resource_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`

	// Derived at read time from (action, module) — see taxonomy.go. Never stored.
	Severity string `json:"severity"`
	Category string `json:"category"`

	// Field-level {before, after} diff captured for restore-eligible edits made
	// after the change-capture upgrade. Nil/omitted for older or non-eligible rows.
	Changes json.RawMessage `json:"changes,omitempty"`
}

type ListResponse struct {
	Items []LogEntry `json:"items"`
	Total int64      `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

type ListQuery struct {
	StoreID    string
	Module     string
	Action     string
	UserID     string
	ResourceID string // exact match — used to pull the full activity chain of one record
	Severity   string // derived filter — translated to (module, action) predicates
	Category   string // derived filter — translated to (module, action) predicates
	Search     string // free text over user_name / module / action / resource_id only
	DateFrom   string
	DateTo     string
	Page       int
	Limit      int
}
