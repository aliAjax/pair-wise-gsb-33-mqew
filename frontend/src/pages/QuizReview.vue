<template>
  <div class="page">
    <div class="header">
      <h1>错题复习</h1>
      <el-button @click="$router.push('/quiz')">返回测验</el-button>
    </div>

    <el-card class="progress-card">
      <div class="progress-text">
        <span>待复习 <b>{{ progress.pending }}</b> 题</span>
        <span>已掌握 <b>{{ progress.resolved }}</b> 题</span>
        <span class="rule">错题连续答对 {{ REVIEW_RESOLVE_THRESHOLD }} 次后移出复习清单</span>
      </div>
      <el-progress
        :percentage="progressPercent"
        :status="progress.pending === 0 ? 'success' : undefined"
        :stroke-width="14"
      />
    </el-card>

    <el-empty v-if="!loading && items.length === 0" description="太棒了，复习清单已清空" />

    <el-card v-for="item in items" :key="item.question_id" class="review-card">
      <div class="q-title">
        {{ item.question }}
        <el-tag size="small" type="warning">已连对 {{ item.consecutive_correct }} / {{ REVIEW_RESOLVE_THRESHOLD }} 次</el-tag>
      </div>
      <el-radio-group v-model="selections[item.question_id]" :disabled="!!feedbacks[item.question_id]">
        <el-radio v-for="(opt, i) in item.options" :key="i" :value="i" class="option">
          {{ opt }}
          <template v-if="feedbacks[item.question_id]">
            <el-tag v-if="i === feedbacks[item.question_id]!.answer" type="success" size="small">正确答案</el-tag>
            <el-tag v-else-if="selections[item.question_id] === i" type="danger" size="small">你的选择</el-tag>
          </template>
        </el-radio>
      </el-radio-group>

      <div v-if="feedbacks[item.question_id]" class="feedback">
        <el-alert
          :type="feedbacks[item.question_id]!.correct ? 'success' : 'error'"
          :closable="false" show-icon
          :title="feedbackTitle(item.question_id)"
        />
        <p v-if="feedbacks[item.question_id]!.explanation" class="explanation">
          解析：{{ feedbacks[item.question_id]!.explanation }}
        </p>
      </div>

      <div class="ops">
        <el-button
          v-if="!feedbacks[item.question_id]"
          type="primary"
          size="small"
          :disabled="selections[item.question_id] === undefined"
          :loading="answeringId === item.question_id"
          @click="answer(item)"
        >提交答案</el-button>
        <el-button v-else size="small" @click="next(item.question_id)">知道了</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listQuizReview, answerQuizReview, getQuizReviewProgress } from '@/api/quiz'
import { REVIEW_RESOLVE_THRESHOLD } from '@/constants/quiz'
import type { QuizReviewItem, QuizReviewAnswerResult, QuizReviewProgress } from '@/types/api'

const items = ref<QuizReviewItem[]>([])
const progress = ref<QuizReviewProgress>({ pending: 0, resolved: 0 })
const selections = ref<Record<number, number | undefined>>({})
const feedbacks = ref<Record<number, QuizReviewAnswerResult | undefined>>({})
const answeringId = ref(0)
const loading = ref(true)

const progressPercent = computed(() => {
  const total = progress.value.pending + progress.value.resolved
  if (total === 0) return 100
  return Math.round((progress.value.resolved / total) * 100)
})

onMounted(async () => {
  await Promise.all([loadReview(), loadProgress()])
  loading.value = false
})

async function loadReview() {
  items.value = await listQuizReview()
}

// 复习页与首页读取同一进度结果
async function loadProgress() {
  progress.value = await getQuizReviewProgress()
}

async function answer(item: QuizReviewItem) {
  const selected = selections.value[item.question_id]
  if (selected === undefined) return
  answeringId.value = item.question_id
  try {
    const res = await answerQuizReview(item.question_id, selected)
    feedbacks.value[item.question_id] = res
    item.consecutive_correct = res.consecutive_correct
    if (res.resolved) {
      ElMessage.success('已连续答对两次，该题移出复习清单')
    }
    await loadProgress()
  } finally {
    answeringId.value = 0
  }
}

function feedbackTitle(questionId: number): string {
  const f = feedbacks.value[questionId]
  if (!f) return ''
  if (!f.correct) return '答错了，连对次数已重置，请对照解析巩固'
  if (f.resolved) return '回答正确，已移出复习清单'
  return `回答正确，再连续答对 ${REVIEW_RESOLVE_THRESHOLD - f.consecutive_correct} 次即可移出`
}

function next(questionId: number) {
  const f = feedbacks.value[questionId]
  delete feedbacks.value[questionId]
  delete selections.value[questionId]
  if (f?.resolved) {
    items.value = items.value.filter((i) => i.question_id !== questionId)
  }
}
</script>

<style scoped>
.page { max-width: 900px; margin: 0 auto; }
.header { display: flex; align-items: center; justify-content: space-between; }
.progress-card { margin-bottom: 16px; }
.progress-text { display: flex; gap: 24px; margin-bottom: 8px; color: #666; }
.progress-text b { color: #3c8d5c; }
.rule { margin-left: auto; font-size: 12px; color: #999; }
.review-card { margin-bottom: 12px; }
.q-title { font-weight: 700; margin-bottom: 8px; display: flex; gap: 8px; align-items: center; }
.option { display: block; margin: 6px 0; }
.feedback { margin-top: 8px; }
.explanation { color: #3c8d5c; margin: 8px 0 0; }
.ops { margin-top: 12px; }
</style>
