<template>
  <div class="page" v-loading="loading">
    <template v-if="attempt">
      <el-page-header @back="goBack" class="header">
        <template #content>
          <span class="title">{{ attempt.set_name }}</span>
          <el-tag size="small" type="info" class="ver-tag">作答版本 v{{ attempt.version }}（题面与答案已固定）</el-tag>
        </template>
      </el-page-header>

      <el-alert
        v-if="isDone"
        :title="`本卷得分：${attempt.score} / ${attempt.total_count}`"
        :type="attempt.score === attempt.total_count ? 'success' : 'warning'"
        :closable="false"
        show-icon
        class="result-alert"
      >
        <template #default>
          按 v{{ attempt.version }} 版本判分，错题已加入复习清单，连续答对 2 次后退出。
        </template>
      </el-alert>
      <el-alert
        v-else
        title="交卷后才会显示正确答案与解析；同一题集只认首次交卷结果，网络失败可直接重试，成绩不会重复累计。"
        type="info"
        :closable="false"
        class="result-alert"
      />

      <QuizCard
        v-for="(q, i) in attempt.questions"
        :key="q.question_id"
        :question="q"
        :index="i"
        :readonly="isDone"
        :show-result="isDone"
        :model-value="answers[q.question_id]"
        @update:model-value="(v: number) => setAnswer(q.question_id, v)"
      />

      <div class="actions" v-if="!isDone">
        <el-button @click="goBack">稍后再做</el-button>
        <el-button type="primary" size="large" :loading="submitting" @click="onSubmit">
          交卷（已答 {{ answeredCount }}/{{ attempt.total_count }}）
        </el-button>
      </div>
      <div class="actions done-actions" v-else>
        <el-button @click="goBack">返回题集</el-button>
        <el-button type="warning" @click="$router.push('/quiz/review')">去复习错题</el-button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import QuizCard from '@/components/common/QuizCard.vue'
import { getAttempt, submitAttempt } from '@/api/quiz'
import { useQuizStore } from '@/stores/quizStore'
import type { Attempt } from '@/types/quiz'

const route = useRoute()
const router = useRouter()
const quizStore = useQuizStore()

const attempt = ref<Attempt | null>(null)
const answers = ref<Record<number, number>>({})
const loading = ref(true)
const submitting = ref(false)

const isDone = computed(() => attempt.value?.status === 'done')
const answeredCount = computed(() => Object.keys(answers.value).length)

// 进行中的作答按尝试号在本地暂存，刷新或"稍后再做"后可恢复已选项。
function draftKey(attemptNo: string) {
  return `gb_quiz_draft_${attemptNo}`
}

onMounted(load)

// 自动保存草稿（仅进行中尝试）。
watch(answers, (val) => {
  if (attempt.value && attempt.value.status === 'open') {
    localStorage.setItem(draftKey(attempt.value.attempt_no), JSON.stringify(val))
  }
}, { deep: true })

async function load() {
  const attemptNo = String(route.params.attemptNo)
  try {
    attempt.value = await getAttempt(attemptNo)
    if (attempt.value.status === 'done') {
      // 已完成的尝试：回填用户原选择，只读查看原卷。
      const saved: Record<number, number> = {}
      for (const q of attempt.value.questions) {
        if (q.selected !== undefined && q.selected >= 0) saved[q.question_id] = q.selected
      }
      answers.value = saved
    } else {
      // 进行中：恢复本地暂存的草稿。
      const raw = localStorage.getItem(draftKey(attemptNo))
      if (raw) {
        try {
          answers.value = JSON.parse(raw)
        } catch {
          answers.value = {}
        }
      }
    }
  } finally {
    loading.value = false
  }
}

function setAnswer(qid: number, v: number) {
  answers.value[qid] = v
}

async function onSubmit() {
  if (!attempt.value) return
  if (answeredCount.value < attempt.value.total_count) {
    try {
      await ElMessageBox.confirm(
        `还有 ${attempt.value.total_count - answeredCount.value} 题未作答，未作答按答错处理，确认交卷？`,
        '确认交卷',
        { type: 'warning', confirmButtonText: '确认交卷', cancelButtonText: '继续作答' },
      )
    } catch {
      return
    }
  }
  submitting.value = true
  try {
    // attempt_no 固定不变：写失败/超时后用户可再次点击，服务端只认首次结果。
    const result = await submitAttempt(attempt.value.attempt_no, { ...answers.value })
    if (!result.first_submitted) {
      ElMessage.info('本次交卷此前已成功提交，展示的是首次结果')
    } else {
      ElMessage.success(`交卷成功，得分 ${result.attempt.score} / ${result.attempt.total_count}`)
    }
    attempt.value = result.attempt
    const saved: Record<number, number> = {}
    for (const q of result.attempt.questions) {
      if (q.selected !== undefined && q.selected >= 0) saved[q.question_id] = q.selected
    }
    answers.value = saved
    localStorage.removeItem(draftKey(attempt.value.attempt_no))
    // 同步首页/复习页共享的进度与错题数。
    await quizStore.fetchProgress(true)
  } finally {
    submitting.value = false
  }
}

function goBack() {
  router.push('/quiz')
}
</script>

<style scoped>
.page { max-width: 900px; margin: 0 auto; }
.header { margin-bottom: 12px; }
.title { font-weight: 700; font-size: 16px; margin-right: 8px; }
.ver-tag { margin-left: 4px; }
.result-alert { margin: 12px 0; }
.actions { margin: 20px 0; text-align: center; }
.done-actions { display: flex; gap: 12px; justify-content: center; }
</style>
