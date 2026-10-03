<template>
  <div class="page">
    <!-- 题集列表 -->
    <template v-if="view === 'sets'">
      <div class="header">
        <h1>养护知识小测验</h1>
        <el-button type="success" plain @click="$router.push('/quiz/review')">错题复习</el-button>
      </div>
      <el-alert
        title="题集来自品种库、养护文章与病虫害手册；开始答题后题面与答案版本即被固定，题库更新不影响进行中与已完成的测验"
        type="info" :closable="false" show-icon class="tip" />
      <el-row :gutter="16">
        <el-col v-for="s in sets" :key="s.id" :xs="24" :sm="12" :md="8">
          <el-card class="set-card">
            <div class="set-title">
              {{ s.title }}
              <el-tag :type="categoryTag(s.category)" size="small">{{ s.category_text }}</el-tag>
            </div>
            <p class="desc">{{ s.description }}</p>
            <p class="meta">
              共 {{ s.question_count }} 题 · 题库 v{{ s.version }} ·
              <el-tag size="small" :type="statusTag(s.my_status)">{{ s.my_status_text }}</el-tag>
            </p>
            <p v-if="s.my_status === 'completed'" class="best">最佳成绩：{{ s.best_score }} / {{ s.best_total }}</p>
            <div class="ops">
              <el-button v-if="s.my_status === 'not_started'" type="primary" @click="begin(s)">开始答题</el-button>
              <el-button v-else-if="s.my_status === 'in_progress'" type="primary" @click="begin(s)">继续答题</el-button>
              <template v-else>
                <el-button @click="viewResult(s)">查看成绩</el-button>
                <el-button type="primary" @click="begin(s)">重新挑战</el-button>
              </template>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </template>

    <!-- 答题中 -->
    <template v-else-if="view === 'quiz' && attempt">
      <h1>答题中 <small>尝试 #{{ attempt.attempt_no }} · 题库 v{{ attempt.set_version }}</small></h1>
      <el-alert :title="`共 ${attempt.total} 题，题面与答案已按当前版本固定`" type="info" :closable="false" show-icon class="tip" />
      <QuizCard
        v-for="(q, i) in attempt.questions"
        :key="q.question_id"
        :question="q"
        :index="i"
        :submitted="false"
        :model-value="answers[q.question_id]"
        @update:model-value="answers[q.question_id] = $event"
      />
      <el-alert
        v-if="submitFailed"
        title="交卷失败，请检查网络后重试；重试使用同一尝试号，成绩不会重复累计"
        type="error" :closable="false" show-icon class="tip" />
      <div class="actions">
        <el-button @click="backToSets">返回题集</el-button>
        <el-button type="primary" size="large" :loading="submitting" @click="submitQuiz">
          {{ submitFailed ? '重试交卷' : '交卷' }}
        </el-button>
      </div>
    </template>

    <!-- 成绩 -->
    <template v-else-if="view === 'result' && result">
      <h1>测验成绩</h1>
      <el-result
        :icon="result.score * 2 >= result.total ? 'success' : 'warning'"
        :title="`得分：${result.score} / ${result.total}`"
      >
        <template #sub-title>
          尝试 #{{ result.attempt_no }} · 按题库 v{{ result.set_version }} 判分 · 答错的题已加入复习清单
        </template>
        <template #extra>
          <el-button type="primary" @click="restart">再答一次</el-button>
          <el-button type="success" plain @click="$router.push('/quiz/review')">去复习错题</el-button>
          <el-button @click="backToSets">返回题集</el-button>
        </template>
      </el-result>
      <QuizCard
        v-for="(q, i) in result.questions"
        :key="q.question_id"
        :question="q"
        :index="i"
        :submitted="true"
        :model-value="q.selected ?? undefined"
        @update:model-value="() => {}"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import QuizCard from '@/components/common/QuizCard.vue'
import { listQuizSets } from '@/api/quiz'
import { useQuiz } from '@/hooks/useQuiz'
import { QuizCategoryTagType, type QuizCategory, type QuizMyStatus } from '@/constants/quiz'
import type { QuizSetItem } from '@/types/api'

const sets = ref<QuizSetItem[]>([])
const view = ref<'sets' | 'quiz' | 'result'>('sets')
const { attempt, answers, result, submitting, submitFailed, start, loadAttempt, submit, reset } = useQuiz()
const currentSetId = ref(0)

onMounted(loadSets)

async function loadSets() {
  sets.value = await listQuizSets()
}

function categoryTag(c: string) {
  return QuizCategoryTagType[(c as QuizCategory)] ?? 'info'
}
function statusTag(s: QuizMyStatus) {
  return s === 'completed' ? 'success' : s === 'in_progress' ? 'warning' : 'info'
}

async function begin(s: QuizSetItem) {
  currentSetId.value = s.id
  reset()
  await start(s.id)
  view.value = 'quiz'
}

async function viewResult(s: QuizSetItem) {
  currentSetId.value = s.id
  reset()
  await loadAttempt(s.my_attempt_id)
  view.value = 'result'
}

async function submitQuiz() {
  if (await submit()) {
    view.value = 'result'
  }
}

async function restart() {
  if (!currentSetId.value) return
  reset()
  await start(currentSetId.value)
  view.value = 'quiz'
}

async function backToSets() {
  reset()
  view.value = 'sets'
  await loadSets()
}
</script>

<style scoped>
.page { max-width: 900px; margin: 0 auto; }
.header { display: flex; align-items: center; justify-content: space-between; }
.tip { margin-bottom: 16px; }
.set-card { margin-bottom: 16px; }
.set-title { font-weight: 700; display: flex; justify-content: space-between; align-items: center; }
.desc { color: #888; font-size: 13px; min-height: 36px; }
.meta { color: #666; font-size: 13px; }
.best { color: #3c8d5c; font-weight: 600; }
.ops { margin-top: 8px; }
.actions { margin: 20px 0; text-align: center; }
h1 small { font-size: 14px; color: #888; font-weight: 400; }
</style>
