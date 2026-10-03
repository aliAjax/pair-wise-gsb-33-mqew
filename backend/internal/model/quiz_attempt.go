package model

import "time"

// QuizAttempt status values.
const (
	QuizAttemptInProgress = "in_progress"
	QuizAttemptSubmitted  = "submitted"
)

// QuizSnapshotQuestion freezes question text, options and the answer at the
// moment an attempt starts, so grading and review stay on that version even
// after the question bank is updated.
type QuizSnapshotQuestion struct {
	QuestionID  uint     `json:"question_id"`
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	Answer      int      `json:"answer"`
	Explanation string   `json:"explanation"`
}

// QuizAttempt is one answering session of a quiz set. The unique key
// (user_id, quiz_set_id, attempt_no) makes submissions idempotent: a retry
// with the same attempt number never accumulates the score twice.
type QuizAttempt struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"uniqueIndex:uk_quiz_attempt_user_set_no;not null;index" json:"user_id"`
	QuizSetID   uint       `gorm:"uniqueIndex:uk_quiz_attempt_user_set_no;not null" json:"quiz_set_id"`
	AttemptNo   int        `gorm:"uniqueIndex:uk_quiz_attempt_user_set_no;not null" json:"attempt_no"`
	SetVersion  int        `gorm:"not null" json:"set_version"`
	Snapshot    string     `gorm:"type:json" json:"snapshot"`
	Answers     string     `gorm:"type:json" json:"answers"`
	Score       int        `gorm:"not null;default:0" json:"score"`
	Total       int        `gorm:"not null;default:0" json:"total"`
	Status      string     `gorm:"size:16;not null;default:in_progress;index" json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	SubmittedAt *time.Time `json:"submitted_at"`
}
