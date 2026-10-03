import request from '@/utils/request'
import type {
  QuizSetItem,
  QuizAttemptDetail,
  QuizReviewItem,
  QuizReviewAnswerResult,
  QuizReviewProgress,
} from '@/types/api'

export function listQuizSets() {
  return request.get<never, QuizSetItem[]>('/quiz/sets')
}

export function startQuizAttempt(setId: number) {
  return request.post<never, QuizAttemptDetail>(`/quiz/sets/${setId}/attempts`)
}

export function getQuizAttempt(attemptId: number) {
  return request.get<never, QuizAttemptDetail>(`/quiz/attempts/${attemptId}`)
}

export function submitQuizAttempt(attemptId: number, answers: Record<number, number>, silent = false) {
  return request.post<never, { message: string; attempt: QuizAttemptDetail }>(
    `/quiz/attempts/${attemptId}/submit`,
    { answers },
    { silent },
  )
}

export function listQuizReview() {
  return request.get<never, QuizReviewItem[]>('/quiz/review')
}

export function answerQuizReview(questionId: number, selected: number) {
  return request.post<never, QuizReviewAnswerResult>('/quiz/review/answer', {
    question_id: questionId,
    selected,
  })
}

// 复习页与首页共用同一进度结果
export function getQuizReviewProgress() {
  return request.get<never, QuizReviewProgress>('/quiz/review/progress')
}
