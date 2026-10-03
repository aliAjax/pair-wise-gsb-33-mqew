export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface UserInfo {
  id: number
  username: string
  email: string
  nickname: string
  avatar: string
  bio: string
  role: 'user' | 'admin'
  created_at: string
}

export interface CareReminder {
  id: number
  user_id: number
  plant_species_id: number
  task_title: string
  remind_date: string
  frequency: string
  status: 'pending' | 'done' | 'overdue'
  created_at: string
}

export interface UserGarden {
  id: number
  user_id: number
  plant_species_id: number
  nickname: string
  owned_since: string
  location: string
  care_reminder_id: number
  created_at: string
}

export interface DiseasePest {
  id: number
  plant_species_id: number
  name: string
  symptoms: string
  cause: string
  treatment: string
  recommended_medicine: string
  images: string
  keywords: string
  created_at: string
}

export interface Question {
  id: number
  user_id: number
  title: string
  content: string
  images: string
  status: string
  created_at: string
}

export interface Answer {
  id: number
  question_id: number
  user_id: number
  content: string
  is_best: boolean
  like_count: number
  created_at: string
}

export type QuizMyStatus = 'not_started' | 'in_progress' | 'completed'

export interface QuizSetItem {
  id: number
  title: string
  category: string
  category_text: string
  description: string
  version: number
  question_count: number
  my_status: QuizMyStatus
  my_status_text: string
  my_attempt_id: number
  best_score: number
  best_total: number
}

export interface QuizQuestionResult {
  question_id: number
  question: string
  options: string[]
  answer: number | null
  explanation: string
  selected: number | null
  correct: boolean
}

export interface QuizAttemptDetail {
  attempt_id: number
  quiz_set_id: number
  attempt_no: number
  set_version: number
  status: 'in_progress' | 'submitted'
  status_text: string
  score: number
  total: number
  submitted_at: string | null
  questions: QuizQuestionResult[]
}

export interface QuizReviewItem {
  question_id: number
  quiz_set_id: number
  question: string
  options: string[]
  consecutive_correct: number
}

export interface QuizReviewAnswerResult {
  correct: boolean
  consecutive_correct: number
  resolved: boolean
  answer: number
  explanation: string
}

export interface QuizReviewProgress {
  pending: number
  resolved: number
}
