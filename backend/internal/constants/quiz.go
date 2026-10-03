package constants

// 知识点题集题目分类（品种 / 文章 / 病虫害）。
const (
	QuizCategoryPlant   = "plant"
	QuizCategoryArticle = "article"
	QuizCategoryPest    = "pest"
)

// ValidQuizCategories returns all accepted quiz question categories.
func ValidQuizCategories() []string {
	return []string{QuizCategoryPlant, QuizCategoryArticle, QuizCategoryPest}
}

// IsValidQuizCategory reports whether the category is known.
func IsValidQuizCategory(c string) bool {
	for _, v := range ValidQuizCategories() {
		if v == c {
			return true
		}
	}
	return false
}

// 错题需连续答对两次才退出复习清单。
const QuizReviewPassStreak = 2
