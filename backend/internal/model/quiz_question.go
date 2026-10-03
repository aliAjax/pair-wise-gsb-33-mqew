package model

import "time"

// QuizQuestion is a single question in the bank of a quiz set. Rows are
// immutable per version: a bank update inserts rows with the new version and
// keeps old rows so in-progress and submitted attempts remain explainable.
type QuizQuestion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	QuizSetID   uint      `gorm:"index:idx_quiz_questions_set_version;not null" json:"quiz_set_id"`
	Version     int       `gorm:"index:idx_quiz_questions_set_version;not null;default:1" json:"version"`
	Question    string    `gorm:"size:512;not null" json:"question"`
	Options     string    `gorm:"type:json;not null" json:"options"`
	Answer      int       `gorm:"not null" json:"answer"`
	Explanation string    `gorm:"size:512" json:"explanation"`
	Sort        int       `gorm:"not null;default:0" json:"sort"`
	CreatedAt   time.Time `json:"created_at"`
}
