package model

import "time"

// 知识点题集相关状态常量。
const (
	QuizSetActive   = "active"
	QuizAttemptOpen = "open"
	QuizAttemptDone = "done"
)

// QuizQuestion 是题库中的单道知识点题目（来源于品种、文章、病虫害内容）。
type QuizQuestion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Category    string    `gorm:"size:32;not null;index" json:"category"` // plant/article/pest
	Title       string    `gorm:"size:255;not null" json:"title"`         // 题面（固定快照的来源）
	Options     string    `gorm:"type:JSON;not null" json:"options"`      // JSON 数组：选项文本
	AnswerIndex int       `gorm:"not null" json:"answer_index"`           // 正确选项下标
	Explanation string    `gorm:"size:1024" json:"explanation"`           // 解析
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// QuizSet 是按知识点组织的题集（品种 / 文章 / 病虫害）。
type QuizSet struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Category    string    `gorm:"size:32;not null;index" json:"category"` // plant/article/pest
	Description string    `gorm:"size:512" json:"description"`
	Status      string    `gorm:"size:16;not null;default:active" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// QuizSetVersion 是题集的不可变版本。题面、选项与答案在发布时整体快照，
// 之后题库改动不会影响已发布版本；题库更新后级联发布新版本。
type QuizSetVersion struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	SetID       uint      `gorm:"index:idx_qsv_set_version,unique,priority:1;not null" json:"set_id"`
	Version     int       `gorm:"index:idx_qsv_set_version,unique,priority:2;not null" json:"version"`
	QuestionIDs string    `gorm:"type:JSON;not null" json:"question_ids"` // 版本包含的题目 id 顺序
	Snapshot    string    `gorm:"type:JSON;not null" json:"snapshot"`     // 题面/选项/答案/解析的整体快照
	CreatedAt   time.Time `json:"created_at"`
}

// QuizAttempt 是用户对某个题集某一版本的一次作答尝试。
type QuizAttempt struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index:idx_quiz_attempt_user_no,unique,priority:1;not null" json:"user_id"`
	AttemptNo   string     `gorm:"size:64;index:idx_quiz_attempt_user_no,unique,priority:2;not null" json:"attempt_no"` // 客户端生成的尝试号，写失败按此号幂等重试
	SetID       uint       `gorm:"index;not null" json:"set_id"`
	VersionID   uint       `gorm:"not null" json:"version_id"`
	Status      string     `gorm:"size:16;not null;default:open" json:"status"` // open/done
	Score       int        `gorm:"not null;default:0" json:"score"`
	TotalCount  int        `gorm:"not null;default:0" json:"total_count"`
	StartedAt   time.Time  `json:"started_at"`
	SubmittedAt *time.Time `json:"submitted_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// QuizAttemptAnswer 记录某次尝试中每道题的最终作答（交卷时整体写入）。
type QuizAttemptAnswer struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	AttemptID  uint      `gorm:"index:idx_quiz_attempt_answer,unique,priority:1;not null" json:"attempt_id"`
	QuestionID uint      `gorm:"index:idx_quiz_attempt_answer,unique,priority:2;not null" json:"question_id"`
	Selected   int       `gorm:"not null" json:"selected"`
	Correct    bool      `gorm:"not null;default:false" json:"correct"`
	CreatedAt  time.Time `json:"created_at"`
}

// QuizReviewAnswer 记录复习清单中的一次作答。以 (user_id, answer_no) 去重，
// 客户端写失败后携带同一 answer_no 重试不会重复累计连对次数。
type QuizReviewAnswer struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index:idx_quiz_review_answer_no,unique,priority:1;not null" json:"user_id"`
	AnswerNo   string    `gorm:"size:64;index:idx_quiz_review_answer_no,unique,priority:2;not null" json:"answer_no"`
	QuestionID uint      `gorm:"index;not null" json:"question_id"`
	Selected   int       `gorm:"not null" json:"selected"`
	Correct    bool      `gorm:"not null;default:false" json:"correct"`
	CreatedAt  time.Time `json:"created_at"`
}

// QuizReviewItem 是错题复习清单条目：错题需再答对两次才退出清单。
type QuizReviewItem struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index:idx_quiz_review_user_question,unique,priority:1;not null" json:"user_id"`
	QuestionID    uint      `gorm:"index:idx_quiz_review_user_question,unique,priority:2;not null" json:"question_id"`
	CorrectStreak int       `gorm:"not null;default:0" json:"correct_streak"` // 连续答对次数，达到 2 退出
	Resolved      bool      `gorm:"not null;default:false" json:"resolved"`
	LastWrongAt   time.Time `json:"last_wrong_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
