package activity_log

import "time"

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
}

type ListResponse struct {
	Items []LogEntry `json:"items"`
	Total int64      `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

type ListQuery struct {
	StoreID  string
	Module   string
	Action   string
	UserID   string
	DateFrom string
	DateTo   string
	Page     int
	Limit    int
}
