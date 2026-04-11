package model

import (
	"time"

	"github.com/google/uuid"
)

type Lesson struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Difficulty  int       `json:"difficulty"`
	ReadTimeMin int       `json:"read_time_min"`
	Tags        []string  `json:"tags"`
	IsPublished bool      `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LessonListItem is a lighter representation without full content.
type LessonListItem struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Difficulty  int       `json:"difficulty"`
	ReadTimeMin int       `json:"read_time_min"`
	Tags        []string  `json:"tags"`
}

type LessonFilter struct {
	Category string
	Page     int
	Limit    int
}
