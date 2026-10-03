package dto

import "time"

// QuizSubmitRequest carries the chosen options of an attempt. Map keys are
// snapshot question ids rendered as strings.
type QuizSubmitRequest struct {
	Answers map[string]int `json:"answers" binding:"required"`
}

// QuizReviewAnswerRequest carries one review answer for a wrong question.
type QuizReviewAnswerRequest struct {
	QuestionID uint `json:"question_id" binding:"required"`
	Selected   int  `json:"selected" binding:"gte=0"`
}

// QuizQuestionInput is one question in a bank create/replace payload.
type QuizQuestionInput struct {
	Question    string   `json:"question" binding:"required,max=512"`
	Options     []string `json:"options" binding:"required,min=2,max=8,dive,required,max=128"`
	Answer      int      `json:"answer" binding:"gte=0"`
	Explanation string   `json:"explanation" binding:"omitempty,max=512"`
}

// QuizSetCreateRequest creates a quiz set together with its first bank version.
type QuizSetCreateRequest struct {
	Title       string              `json:"title" binding:"required,max=128"`
	Category    string              `json:"category" binding:"required,oneof=plant article pest"`
	Description string              `json:"description" binding:"omitempty,max=255"`
	Questions   []QuizQuestionInput `json:"questions" binding:"required,min=1,dive"`
}

// QuizSetQuestionsUpdateRequest replaces the bank and bumps the version.
type QuizSetQuestionsUpdateRequest struct {
	Questions []QuizQuestionInput `json:"questions" binding:"required,min=1,dive"`
}

// QuizSetItem is a quiz set card with the current user's progress.
type QuizSetItem struct {
	ID            uint   `json:"id"`
	Title         string `json:"title"`
	Category      string `json:"category"`
	CategoryText  string `json:"category_text"`
	Description   string `json:"description"`
	Version       int    `json:"version"`
	QuestionCount int64  `json:"question_count"`
	MyStatus      string `json:"my_status"`
	MyStatusText  string `json:"my_status_text"`
	MyAttemptID   uint   `json:"my_attempt_id"`
	BestScore     int    `json:"best_score"`
	BestTotal     int    `json:"best_total"`
}

// QuizQuestionView is a question while answering: no answer leaks.
type QuizQuestionView struct {
	QuestionID uint     `json:"question_id"`
	Question   string   `json:"question"`
	Options    []string `json:"options"`
}

// QuizQuestionResultView is a question after submission, with the frozen
// answer of the attempt's version plus the user's choice. Answer stays nil
// while the attempt is in progress so the bank answer never leaks.
type QuizQuestionResultView struct {
	QuestionID  uint     `json:"question_id"`
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	Answer      *int     `json:"answer"`
	Explanation string   `json:"explanation"`
	Selected    *int     `json:"selected"`
	Correct     bool     `json:"correct"`
}

// QuizAttemptDetail describes an attempt. Questions carry no answers while
// the attempt is still in progress.
type QuizAttemptDetail struct {
	AttemptID   uint                     `json:"attempt_id"`
	QuizSetID   uint                     `json:"quiz_set_id"`
	AttemptNo   int                      `json:"attempt_no"`
	SetVersion  int                      `json:"set_version"`
	Status      string                   `json:"status"`
	StatusText  string                   `json:"status_text"`
	Score       int                      `json:"score"`
	Total       int                      `json:"total"`
	SubmittedAt *time.Time               `json:"submitted_at"`
	Questions   []QuizQuestionResultView `json:"questions"`
}

// QuizReviewItem is a pending wrong question in the review list.
type QuizReviewItem struct {
	QuestionID         uint     `json:"question_id"`
	QuizSetID          uint     `json:"quiz_set_id"`
	Question           string   `json:"question"`
	Options            []string `json:"options"`
	ConsecutiveCorrect int      `json:"consecutive_correct"`
}

// QuizReviewAnswerResult reports the outcome of one review answer.
type QuizReviewAnswerResult struct {
	Correct            bool   `json:"correct"`
	ConsecutiveCorrect int    `json:"consecutive_correct"`
	Resolved           bool   `json:"resolved"`
	Answer             int    `json:"answer"`
	Explanation        string `json:"explanation"`
}

// QuizReviewProgress is the single progress source read by both the review
// page and the home page.
type QuizReviewProgress struct {
	Pending  int64 `json:"pending"`
	Resolved int64 `json:"resolved"`
}
