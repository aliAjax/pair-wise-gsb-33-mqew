<template>
  <div class="page" v-loading="loading">
    <el-page-header @back="$router.push('/quiz')" content="错题复习" class="header" />
    <el-alert
      title="复习规则：每道错题需要连续答对 2 次才会退出清单；中途答错则重新计数。判分以题库当前答案为准。"
      type="info"
      :closable="false"
      show-icon
      class="rule"
    />

    <el-empty v-if="!loading && items.length === 0" description="暂无待复习错题，太棒了！">
      <el-button type="primary" @click="$router.push('/quiz')">返回题集</el-button>
    </el-empty>

    <el-card v-for="(item, index) in items" :key="item.question_id" class="review-card">
      <div class="q-head">
        <span class="q-title">{{ index + 1 }}. {{ item.title }}</span>
        <el-tag size="small" :type="item.correct_streak > 0 ? 'success' : 'info'">
          已连对 {{ item.correct_streak }}/2
        </el-tag>
      </div>

      <el-radio-group v-model="picks[item.question_id]" :disabled="locked[item.question_id]">
        <el-radio v-for="(opt, i) in item.options" :key="i" :value="i" class="option">{{ opt }}</el-radio>
      </el-radio-group>

      <div class="row-actions">
        <el-button
          type="primary"
          size="small"
          :loading="submitting[item.question_id]"
          :disabled="picks[item.question_id] === undefined || locked[item.question_id]"
          @click="onAnswer(item)"
        >提交答案</el-button>
      </div>

      <el-alert
        v-if="results[item.question_id]"
        class="feedback"
        :title="results[item.question_id].correct ? '回答正确' : '回答错误'"
        :type="results[item.question_id].correct ? 'success' : 'error'"
        :closable="false"
        show-icon
      >
        <template #default>
          <div v-if="results[item.question_id].resolved">
            已连续答对 2 次，本题退出复习清单 🎉
          </div>
          <div v-else-if="results[item.question_id].correct">
            当前连对 {{ results[item.question_id].correct_streak }}/2，再答对 1 次即可退出。
          </div>
          <div v-else>
            连对计数已清零。正确答案：{{ item.options[results[item.question_id].answer_index] }}
            <div v-if="item.explanation" class="explain">解析：{{ item.explanation }}</div>
          </div>
        </template>
      </el-alert>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { answerReview, genAttemptNo } from '@/api/quiz'
import { useQuizStore } from '@/stores/quizStore'
import type { ReviewAnswerResult, ReviewItem } from '@/types/quiz'

const quizStore = useQuizStore()
const items = ref<ReviewItem[]>([])
const loading = ref(true)

// 每题当前选择、提交中状态、结果反馈、以及是否已在本轮答对（锁定等待刷新）。
const picks = reactive<Record<number, number>>({})
const submitting = reactive<Record<number, boolean>>({})
const locked = reactive<Record<number, boolean>>({})
const results = reactive<Record<number, ReviewAnswerResult>>({})

onMounted(load)

async function load() {
  loading.value = true
  try {
    items.value = await quizStore.fetchReview()
  } finally {
    loading.value = false
  }
}

async function onAnswer(item: ReviewItem) {
  const qid = item.question_id
  const selected = picks[qid]
  if (selected === undefined) return
  // answer_no 在一次提交动作内固定；网络失败时按同一号重试，服务端不重复累计。
  const answerNo = pendingNos[qid] || genAttemptNo()
  pendingNos[qid] = answerNo
  submitting[qid] = true
  try {
    const result = await answerReview(answerNo, qid, selected)
    results[qid] = result
    if (result.correct) {
      locked[qid] = true
      if (result.resolved) {
        ElMessage.success('本题已退出复习清单')
        setTimeout(() => {
          load()
        }, 900)
      } else {
        // 答对一次：允许继续答第二次，清空选择并解锁。
        setTimeout(() => {
          delete picks[qid]
          delete locked[qid]
        }, 900)
      }
    } else {
      // 答错：计数清零，保留选择与反馈，允许重新作答。
      delete pendingNos[qid]
    }
  } catch {
    // 写失败不换号，用户再次点击仍以同一 answer_no 重试。
  } finally {
    submitting[qid] = false
  }
}

// 每题挂起的幂等作答号：只有产生新的一次作答意图（答错后）才换新号。
const pendingNos = reactive<Record<number, string>>({})
</script>

<style scoped>
.page { max-width: 900px; margin: 0 auto; }
.header { margin-bottom: 12px; }
.rule { margin-bottom: 16px; }
.review-card { margin-bottom: 14px; }
.q-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.q-title { font-weight: 700; }
.option { display: block; margin: 6px 0; }
.row-actions { margin-top: 10px; }
.feedback { margin-top: 10px; }
.explain { margin-top: 4px; color: #3c8d5c; }
</style>
