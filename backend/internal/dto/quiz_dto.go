package dto

import "time"

// QuizQuestionRequest 是管理员维护题库题目的请求。
type QuizQuestionRequest struct {
	Category    string   `json:"category" binding:"required"`
	Title       string   `json:"title" binding:"required"`
	Options     []string `json:"options" binding:"required,min=2"`
	AnswerIndex int      `json:"answer_index"` // 可为 0（第一个选项），由 service 校验范围
	Explanation string   `json:"explanation"`
}

// QuizQuestionDTO 是题目的对外结构（题库管理用）。
type QuizQuestionDTO struct {
	ID          uint      `json:"id"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Options     []string  `json:"options"`
	AnswerIndex int       `json:"answer_index"`
	Explanation string    `json:"explanation"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// QuizSetDTO 是题集列表/详情结构，附带当前最新版本号。
type QuizSetDTO struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Category      string    `json:"category"`
	Description   string    `json:"description"`
	LatestVersion int       `json:"latest_version"`
	QuestionCount int       `json:"question_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// SnapshotQuestion 是版本快照中单题的不可变内容。
type SnapshotQuestion struct {
	QuestionID  uint     `json:"question_id"`
	Title       string   `json:"title"`
	Options     []string `json:"options"`
	AnswerIndex int      `json:"answer_index"`
	Explanation string   `json:"explanation"`
}

// StartAttemptRequest 携带客户端生成的尝试号；写失败后按同一尝试号重试。
type StartAttemptRequest struct {
	SetID     uint   `json:"set_id" binding:"required"`
	AttemptNo string `json:"attempt_no" binding:"required"`
}

// SubmitAttemptRequest 携带尝试号与整份作答。
type SubmitAttemptRequest struct {
	AttemptNo string       `json:"attempt_no" binding:"required"`
	Answers   map[uint]int `json:"answers" binding:"required"` // question_id -> 选项下标
}

// AttemptQuestion 是答题页看到的单题：未交卷时不返回正确答案。
type AttemptQuestion struct {
	QuestionID  uint     `json:"question_id"`
	Title       string   `json:"title"`
	Options     []string `json:"options"`
	AnswerIndex *int     `json:"answer_index,omitempty"` // 仅交卷后返回
	Explanation string   `json:"explanation,omitempty"`  // 仅交卷后返回
	Selected    *int     `json:"selected,omitempty"`     // 交卷后返回用户选择
	Correct     *bool    `json:"correct,omitempty"`      // 仅交卷后返回
}

// AttemptDTO 是一次尝试的完整视图（答题页/复习进度共用）。
type AttemptDTO struct {
	ID          uint              `json:"id"`
	AttemptNo   string            `json:"attempt_no"`
	SetID       uint              `json:"set_id"`
	SetName     string            `json:"set_name"`
	VersionID   uint              `json:"version_id"`
	Version     int               `json:"version"`
	Status      string            `json:"status"`
	Score       int               `json:"score"`
	TotalCount  int               `json:"total_count"`
	StartedAt   time.Time         `json:"started_at"`
	SubmittedAt *time.Time        `json:"submitted_at,omitempty"`
	Questions   []AttemptQuestion `json:"questions"`
}

// SubmitResult 是交卷结果。重复交卷时 first_submitted=false，表示只认首次结果。
type SubmitResult struct {
	Attempt        *AttemptDTO `json:"attempt"`
	FirstSubmitted bool        `json:"first_submitted"`
}

// ReviewAnswerRequest 是复习页单题作答，answer_no 为客户端幂等键。
type ReviewAnswerRequest struct {
	AnswerNo   string `json:"answer_no" binding:"required"`
	QuestionID uint   `json:"question_id" binding:"required"`
	Selected   int    `json:"selected"`
}

// ReviewItemDTO 是复习清单中的一道错题。
type ReviewItemDTO struct {
	QuestionID    uint      `json:"question_id"`
	Title         string    `json:"title"`
	Options       []string  `json:"options"`
	AnswerIndex   int       `json:"answer_index"`
	Explanation   string    `json:"explanation"`
	CorrectStreak int       `json:"correct_streak"` // 已连续答对次数，达到 2 退出
	UpdatedAt     time.Time `json:"updated_at"`
}

// ReviewAnswerResult 是复习作答判定结果（重试返回同一结果，不重复累计）。
type ReviewAnswerResult struct {
	Correct       bool `json:"correct"`
	CorrectStreak int  `json:"correct_streak"`
	Resolved      bool `json:"resolved"` // 连续答对两次，已退出复习清单
	AnswerIndex   int  `json:"answer_index"`
}

// SetProgress 是单个题集的进度，首页与复习页读取同一份结果。
type SetProgress struct {
	SetID      uint   `json:"set_id"`
	VersionID  uint   `json:"version_id"`
	Version    int    `json:"version"`
	Status     string `json:"status"` // open/done/not_started
	Score      int    `json:"score"`
	TotalCount int    `json:"total_count"`
	AttemptID  uint   `json:"attempt_id"`
	AttemptNo  string `json:"attempt_no"`
}

// QuizProgress 是当前用户的测验总进度。
type QuizProgress struct {
	Sets           []SetProgress `json:"sets"`
	ReviewCount    int           `json:"review_count"`
	TotalScore     int           `json:"total_score"` // 各题集首次完成成绩之和（重考不累计）
	TotalQuestions int           `json:"total_questions"`
}
