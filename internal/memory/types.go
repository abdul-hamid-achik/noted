package memory

import (
	"time"
)

// Memory represents a stored memory with metadata
type Memory struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Category   string    `json:"category"`
	Importance int       `json:"importance"`
	Source     string    `json:"source,omitempty"`
	SourceRef  string    `json:"source_ref,omitempty"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Tags       []string  `json:"tags,omitempty"`
	Score      float64   `json:"score,omitempty"` // For search results
}

// RememberInput contains parameters for creating a new memory
type RememberInput struct {
	Content    string
	Title      string // Optional, generated from content if empty
	Category   string // Default: "fact"
	Importance int    // 1-5, default: 3
	TTL        time.Duration // Optional TTL
	Source     string // Optional source identifier
	SourceRef  string // Optional source reference
}

// RecallInput contains parameters for recalling memories
type RecallInput struct {
	Query        string
	Limit        int    // Default: 5
	Category     string // Optional filter
	UseSemantic  bool   // Prefer semantic search if available
}

// RecallResult contains the results of a recall operation
type RecallResult struct {
	Query    string    `json:"query"`
	Method   string    `json:"method"` // "semantic" or "keyword"
	Count    int       `json:"count"`
	Memories []Memory  `json:"memories"`
}

// ForgetInput contains parameters for forgetting memories
type ForgetInput struct {
	OlderThanDays   int
	ImportanceBelow int
	Category        string
	Query           string // Text search to match
	ID              int64  // Specific ID to delete
	DryRun          bool   // Default: true
}

// ForgetResult contains the results of a forget operation
type ForgetResult struct {
	DryRun     bool       `json:"dry_run"`
	Deleted    int        `json:"deleted"`
	WouldDelete int       `json:"would_delete,omitempty"`
	Memories   []Memory   `json:"memories"`
	Criteria   ForgetInput `json:"criteria,omitempty"`
}

// ValidCategories lists all valid memory categories
var ValidCategories = []string{
	"user-pref", // User preferences
	"project",   // Project information
	"decision",  // Decisions made
	"fact",      // Factual information
	"todo",      // Action items
}

// IsValidCategory checks if a category is valid
func IsValidCategory(cat string) bool {
	for _, c := range ValidCategories {
		if c == cat {
			return true
		}
	}
	return false
}
