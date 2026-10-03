// 与后端 internal/constants/quiz.go 保持一致
export type QuizCategory = 'plant' | 'article' | 'pest'

export const QuizCategoryMap: Record<QuizCategory, string> = {
  plant: '品种知识',
  article: '养护文章',
  pest: '病虫害防治',
}

export const QuizCategoryTagType: Record<QuizCategory, 'success' | 'primary' | 'warning'> = {
  plant: 'success',
  article: 'primary',
  pest: 'warning',
}

export type QuizMyStatus = 'not_started' | 'in_progress' | 'completed'

export const QuizMyStatusMap: Record<QuizMyStatus, string> = {
  not_started: '未开始',
  in_progress: '进行中',
  completed: '已完成',
}

// 错题连续答对多少次后移出复习清单（与后端 model.WrongQuestionResolveThreshold 一致）
export const REVIEW_RESOLVE_THRESHOLD = 2
