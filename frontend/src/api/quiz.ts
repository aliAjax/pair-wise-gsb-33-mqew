import request from '@/utils/request'
import type {
  Attempt,
  QuizProgress,
  QuizQuestionAdmin,
  QuizSet,
  ReviewAnswerResult,
  ReviewItem,
  SubmitResult,
} from '@/types/quiz'

// 生成尝试号 / 复习作答号。
// 优先用 crypto.randomUUID；同一会话加序号后缀，保证写失败后可用同一号重试。
export function genAttemptNo(): string {
  const c = globalThis.crypto as Crypto | undefined
  if (c?.randomUUID) return c.randomUUID()
  return `attempt-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

// ---- 普通用户 ----

export function listQuizSets() {
  return request.get<never, QuizSet[]>('/quiz/sets')
}

export function getQuizProgress() {
  return request.get<never, QuizProgress>('/quiz/progress')
}

// 开始作答；同一 attempt_no 重试幂等返回同一次尝试。
export function startAttempt(setId: number, attemptNo: string) {
  return request.post<never, Attempt>('/quiz/attempts', { set_id: setId, attempt_no: attemptNo })
}

export function getAttempt(attemptNo: string) {
  return request.get<never, Attempt>(`/quiz/attempts/${attemptNo}`)
}

// 交卷；同一 attempt_no 重复提交只认首次结果，不重复累计成绩。
export function submitAttempt(attemptNo: string, answers: Record<number, number>) {
  return request.post<never, SubmitResult>('/quiz/attempts/submit', { attempt_no: attemptNo, answers })
}

export function listReview() {
  return request.get<never, ReviewItem[]>('/quiz/review')
}

// 复习页单题作答；answer_no 幂等，重试不重复累计连对次数。
export function answerReview(answerNo: string, questionId: number, selected: number) {
  return request.post<never, ReviewAnswerResult>('/quiz/review/answer', {
    answer_no: answerNo,
    question_id: questionId,
    selected,
  })
}

// ---- 管理员题库维护 ----

export function adminListQuestions(category?: string) {
  return request.get<never, QuizQuestionAdmin[]>('/quiz/admin/questions', { params: { category } })
}

export function adminCreateQuestion(payload: Omit<QuizQuestionAdmin, 'id' | 'created_at' | 'updated_at'>) {
  return request.post<never, QuizQuestionAdmin>('/quiz/admin/questions', payload)
}

export function adminUpdateQuestion(id: number, payload: Omit<QuizQuestionAdmin, 'id' | 'created_at' | 'updated_at'>) {
  return request.put<never, QuizQuestionAdmin>(`/quiz/admin/questions/${id}`, payload)
}
