package constants

// QuizCategory values. Mirrored in frontend src/constants/quiz.ts and used by
// quiz models, services, formatters, log templates and error messages.
const (
	QuizCategoryPlant   = "plant"
	QuizCategoryArticle = "article"
	QuizCategoryPest    = "pest"
)

// QuizMyStatus values describe the current user's progress on a quiz set.
const (
	QuizMyStatusNotStarted = "not_started"
	QuizMyStatusInProgress = "in_progress"
	QuizMyStatusCompleted  = "completed"
)
