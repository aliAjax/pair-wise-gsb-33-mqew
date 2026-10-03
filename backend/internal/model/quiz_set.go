package model

import "time"

// QuizSet is a knowledge-point question set organized from plant species,
// care articles and disease/pest content. Version increments whenever the
// question bank is replaced; attempts started earlier keep their snapshot.
type QuizSet struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:128;not null" json:"title"`
	Category    string    `gorm:"size:16;not null;index" json:"category"`
	Description string    `gorm:"size:255" json:"description"`
	Version     int       `gorm:"not null;default:1" json:"version"`
	Status      string    `gorm:"size:16;not null;default:published" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// QuizSet status values.
const (
	QuizSetStatusPublished = "published"
	QuizSetStatusArchived  = "archived"
)
