// 知识点题集相关类型

export type QuizCategory = 'plant' | 'article' | 'pest'

export interface QuizSet {
  id: number
  name: string
  category: QuizCategory
  description: string
  latest_version: number
  question_count: number
  created_at: string
}

export interface QuizQuestionAdmin {
  id: number
  category: QuizCategory
  title: string
  options: string[]
  answer_index: number
  explanation: string
  created_at: string
  updated_at: string
}

export interface AttemptQuestion {
  question_id: number
  title: string
  options: string[]
  answer_index?: number // 仅交卷后返回（版本快照中的固定答案）
  explanation?: string // 仅交卷后返回
  selected?: number // 交卷后返回用户选择；未作答为 -1
  correct?: boolean // 仅交卷后返回
}

export type AttemptStatus = 'open' | 'done'

export interface Attempt {
  id: number
  attempt_no: string
  set_id: number
  set_name: string
  version_id: number
  version: number
  status: AttemptStatus
  score: number
  total_count: number
  started_at: string
  submitted_at?: string
  questions: AttemptQuestion[]
}

export interface SubmitResult {
  attempt: Attempt
  first_submitted: boolean
}

export interface ReviewItem {
  question_id: number
  title: string
  options: string[]
  answer_index: number
  explanation: string
  correct_streak: number
  updated_at: string
}

export interface ReviewAnswerResult {
  correct: boolean
  correct_streak: number
  resolved: boolean
  answer_index: number
}

export type SetProgressStatus = 'not_started' | 'open' | 'done'

export interface SetProgress {
  set_id: number
  version_id: number
  version: number
  status: SetProgressStatus
  score: number
  total_count: number
  attempt_id: number
  attempt_no: string
}

export interface QuizProgress {
  sets: SetProgress[]
  review_count: number
  total_score: number
  total_questions: number
}
