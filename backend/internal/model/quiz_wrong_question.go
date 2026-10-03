package model

import "time"

// Review list status values for wrong questions.
const (
	WrongQuestionPending  = "pending"
	WrongQuestionResolved = "resolved"
)

// WrongQuestionResolveThreshold is how many consecutive correct answers a
// wrong question needs before it leaves the review list.
const WrongQuestionResolveThreshold = 2

// QuizWrongQuestion tracks a question the user answered incorrectly. It keeps
// its own snapshot so review always replays the version the user failed on.
type QuizWrongQuestion struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	UserID             uint      `gorm:"uniqueIndex:uk_quiz_wrong_user_question;not null;index" json:"user_id"`
	QuizSetID          uint      `gorm:"not null" json:"quiz_set_id"`
	QuestionID         uint      `gorm:"uniqueIndex:uk_quiz_wrong_user_question;not null" json:"question_id"`
	Snapshot           string    `gorm:"type:json" json:"snapshot"`
	ConsecutiveCorrect int       `gorm:"not null;default:0" json:"consecutive_correct"`
	Status             string    `gorm:"size:16;not null;default:pending;index" json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
