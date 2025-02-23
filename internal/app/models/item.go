package models

import "time"

// ItemType represents the type of content
type ItemType string

const (
	TypeNote       ItemType = "note"
	TypeBookmark   ItemType = "bookmark"
	TypeTask       ItemType = "task"
	TypeWorkstream ItemType = "workstream"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusTodo TaskStatus = "todo"
	TaskStatusDone TaskStatus = "done"
)

// Item represents a content item in the system
type Item struct {
	ID       string    `json:"id"`
	Type     ItemType  `json:"type"`
	Tags     []string  `json:"tags"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`

	// Type-specific fields
	URL    string     `json:"url,omitempty"`    // for bookmarks
	Status TaskStatus `json:"status,omitempty"` // for tasks
	Items  []string   `json:"items,omitempty"`  // for workstreams
}

// NewItem creates a new item with the given type and ID
func NewItem(itemType ItemType, id string) *Item {
	now := time.Now().UTC()
	return &Item{
		ID:       id,
		Type:     itemType,
		Created:  now,
		Modified: now,
		Tags:     make([]string, 0),
	}
}
