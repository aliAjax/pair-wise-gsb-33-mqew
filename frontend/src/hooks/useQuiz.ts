import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { startQuizAttempt, getQuizAttempt, submitQuizAttempt } from '@/api/quiz'
import type { QuizAttemptDetail } from '@/types/api'

// useQuiz 管理一次题集作答：开始时由服务端固定题面/选项/答案版本（快照），
// 提交按尝试号幂等——写入失败后以同一尝试号重试，成绩不重复累计。
export function useQuiz() {
  const attempt = ref<QuizAttemptDetail | null>(null)
  const answers = ref<Record<number, number>>({})
  const result = ref<QuizAttemptDetail | null>(null)
  const submitting = ref(false)
  const submitFailed = ref(false)

  async function start(setId: number) {
    attempt.value = await startQuizAttempt(setId)
    answers.value = {}
    result.value = null
    submitFailed.value = false
    return attempt.value
  }

  async function loadAttempt(attemptId: number) {
    const detail = await getQuizAttempt(attemptId)
    if (detail.status === 'submitted') {
      result.value = detail
    } else {
      attempt.value = detail
      answers.value = {}
      result.value = null
    }
    return detail
  }

  async function submitOnce(silent: boolean) {
    const res = await submitQuizAttempt(attempt.value!.attempt_id, { ...answers.value }, silent)
    return res.attempt
  }

  async function submit(): Promise<boolean> {
    if (!attempt.value || submitting.value) return false
    submitting.value = true
    submitFailed.value = false
    try {
      try {
        // 首次提交（静默，失败不打扰用户）
        result.value = await submitOnce(true)
      } catch {
        // 写入失败后按同一尝试号重试一次；服务端只认首次结果，成绩不重复累计
        result.value = await submitOnce(false)
      }
      return true
    } catch {
      submitFailed.value = true
      ElMessage.error('交卷失败，请点击重试（同一尝试号重试，成绩不会重复累计）')
      return false
    } finally {
      submitting.value = false
    }
  }

  function reset() {
    attempt.value = null
    answers.value = {}
    result.value = null
    submitFailed.value = false
  }

  return { attempt, answers, result, submitting, submitFailed, start, loadAttempt, submit, reset }
}
