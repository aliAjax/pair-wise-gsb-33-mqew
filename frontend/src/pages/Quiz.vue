<template>
  <div class="page">
    <h1>养护知识小测验</h1>
    <el-alert
      title="题目按知识点分册，题面、选项与答案在开始作答时固定为当时版本；题库更新只影响尚未开始的题集，进行中与已完成的作答始终按原版本查看。"
      type="info"
      :closable="false"
      show-icon
    />

    <el-row :gutter="16" class="summary" v-loading="loading">
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value">{{ quizStore.progress?.total_score ?? 0 }} / {{ quizStore.progress?.total_questions ?? 0 }}</div>
          <div class="stat-label">累计成绩（每册仅计首次完成）</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="stat-card" @click="goReview">
          <div class="stat-value clickable">{{ quizStore.progress?.review_count ?? 0 }}</div>
          <div class="stat-label">错题复习（答对 2 次退出）</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value">{{ doneCount }} / {{ sets.length }}</div>
          <div class="stat-label">已完成题集</div>
        </el-card>
      </el-col>
    </el-row>

    <h2>知识点题集</h2>
    <el-row :gutter="16" v-loading="loading">
      <el-col v-for="set in sets" :key="set.id" :xs="24" :sm="12" :md="8">
        <el-card shadow="hover" class="set-card">
          <div class="set-head">
            <span class="set-name">{{ set.name }}</span>
            <el-tag size="small" type="info">v{{ progressMap[set.id]?.version || set.latest_version }}</el-tag>
          </div>
          <p class="set-desc">{{ set.description }}</p>
          <div class="set-meta">共 {{ set.question_count }} 题</div>

          <template #footer>
            <el-tag v-if="sp(set.id)?.status === 'open'" type="warning" size="small">进行中 v{{ sp(set.id)?.version }}</el-tag>
            <el-tag v-else-if="sp(set.id)?.status === 'done'" type="success" size="small">
              已完成 {{ sp(set.id)?.score }}/{{ sp(set.id)?.total_count }}（v{{ sp(set.id)?.version }}）
            </el-tag>
            <el-tag v-else type="info" size="small">未开始</el-tag>

            <div class="set-actions">
              <el-button
                v-if="sp(set.id)?.status === 'open'"
                type="primary"
                size="small"
                @click="resume(set)"
              >继续作答</el-button>
              <el-button
                v-if="sp(set.id)?.status === 'done'"
                size="small"
                @click="reviewResult(set)"
              >查看原卷</el-button>
              <el-button
                v-if="sp(set.id)?.status === 'done'"
                type="primary"
                plain
                size="small"
                @click="restart(set)"
              >重新作答</el-button>
              <el-button
                v-if="!sp(set.id) || sp(set.id)?.status === 'not_started'"
                type="primary"
                size="small"
                @click="start(set)"
              >开始作答</el-button>
            </div>
          </template>
        </el-card>
      </el-col>
    </el-row>

    <el-alert
      v-if="!loading && (quizStore.progress?.review_count ?? 0) > 0"
      class="review-entry"
      type="warning"
      show-icon
      :closable="false"
    >
      <template #title>
        你有 {{ quizStore.progress?.review_count }} 道错题待复习，
        <el-link type="primary" :underline="false" @click="goReview">前往复习 →</el-link>
      </template>
    </el-alert>

    <div v-if="auth.isAdmin" class="admin-entry">
      <el-button plain @click="openAdmin">⚙️ 题库管理（管理员：新增/修改题目会发布新版本）</el-button>
    </div>

    <!-- 管理员题库管理 -->
    <el-dialog v-model="adminVisible" title="题库管理" width="720px" top="6vh">
      <el-radio-group v-model="adminCategory" size="small" class="admin-cat" @change="loadAdminQuestions">
        <el-radio-button value="plant">品种</el-radio-button>
        <el-radio-button value="article">文章</el-radio-button>
        <el-radio-button value="pest">病虫害</el-radio-button>
      </el-radio-group>
      <el-button type="primary" size="small" class="admin-add" @click="editQuestion(null)">新增题目</el-button>

      <el-table :data="adminQuestions" size="small" max-height="360" v-loading="adminLoading">
        <el-table-column prop="id" label="ID" width="56" />
        <el-table-column prop="title" label="题面" show-overflow-tooltip />
        <el-table-column label="答案" width="70">
          <template #default="{ row }">{{ String.fromCharCode(65 + row.answer_index) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="editQuestion(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-dialog v-model="editorVisible" :title="form.id ? `编辑题目 #${form.id}` : '新增题目'" width="600px" append-to-body>
        <el-form label-width="72px">
          <el-form-item label="分类">
            <el-radio-group v-model="form.category" :disabled="!!form.id">
              <el-radio value="plant">品种</el-radio>
              <el-radio value="article">文章</el-radio>
              <el-radio value="pest">病虫害</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="题面">
            <el-input v-model="form.title" type="textarea" :rows="2" />
          </el-form-item>
          <el-form-item v-for="(_, i) in form.options" :key="i" :label="`选项 ${String.fromCharCode(65 + i)}`">
            <div class="opt-row">
              <el-input v-model="form.options[i]" class="opt-input" />
              <el-button
                link
                type="danger"
                :disabled="form.options.length <= 2"
                @click="removeOption(i)"
              >删除</el-button>
            </div>
          </el-form-item>
          <el-form-item>
            <el-button size="small" :disabled="form.options.length >= 8" @click="form.options.push('')">+ 添加选项</el-button>
          </el-form-item>
          <el-form-item label="正确答案">
            <el-radio-group v-model="form.answer_index">
              <el-radio v-for="(o, i) in form.options" :key="i" :value="i">{{ String.fromCharCode(65 + i) }}. {{ o }}</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="解析">
            <el-input v-model="form.explanation" type="textarea" :rows="2" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="editorVisible = false">取消</el-button>
          <el-button type="primary" :loading="adminSaving" @click="saveQuestion">保存（发布新版本）</el-button>
        </template>
      </el-dialog>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  adminCreateQuestion,
  adminListQuestions,
  adminUpdateQuestion,
  listQuizSets,
  startAttempt,
} from '@/api/quiz'
import { useAuthStore } from '@/stores/authStore'
import { useQuizStore } from '@/stores/quizStore'
import type { QuizCategory, QuizQuestionAdmin, QuizSet, SetProgress } from '@/types/quiz'

const router = useRouter()
const auth = useAuthStore()
const quizStore = useQuizStore()

const sets = ref<QuizSet[]>([])
const loading = ref(true)

const progressMap = computed<Record<number, SetProgress>>(() => {
  const map: Record<number, SetProgress> = {}
  for (const p of quizStore.progress?.sets || []) map[p.set_id] = p
  return map
})

const doneCount = computed(() => (quizStore.progress?.sets || []).filter((p) => p.status === 'done').length)

function sp(setId: number): SetProgress | undefined {
  return progressMap.value[setId]
}

// ---- 管理员题库管理 ----
const adminVisible = ref(false)
const adminLoading = ref(false)
const adminSaving = ref(false)
const adminCategory = ref<QuizCategory>('plant')
const adminQuestions = ref<QuizQuestionAdmin[]>([])
const editorVisible = ref(false)
const form = reactive<{ id?: number; category: QuizCategory; title: string; options: string[]; answer_index: number; explanation: string }>({
  category: 'plant',
  title: '',
  options: ['', '', '', ''],
  answer_index: 0,
  explanation: '',
})

async function openAdmin() {
  adminVisible.value = true
  await loadAdminQuestions()
}

async function loadAdminQuestions() {
  adminLoading.value = true
  try {
    adminQuestions.value = await adminListQuestions(adminCategory.value)
  } finally {
    adminLoading.value = false
  }
}

function editQuestion(row: QuizQuestionAdmin | null) {
  if (row) {
    form.id = row.id
    form.category = row.category
    form.title = row.title
    form.options = [...row.options]
    form.answer_index = row.answer_index
    form.explanation = row.explanation
  } else {
    form.id = undefined
    form.category = adminCategory.value
    form.title = ''
    form.options = ['', '', '', '']
    form.answer_index = 0
    form.explanation = ''
  }
  editorVisible.value = true
}

// 删除选项时若删掉的是当前正确答案之前的项，需要把答案下标前移。
function removeOption(index: number) {
  form.options.splice(index, 1)
  if (form.answer_index === index) {
    form.answer_index = 0
  } else if (form.answer_index > index) {
    form.answer_index -= 1
  }
}

async function saveQuestion() {
  if (!form.title.trim()) {
    ElMessage.warning('请填写题面')
    return
  }
  if (form.options.some((o) => !o.trim())) {
    ElMessage.warning('请填写全部选项')
    return
  }
  if (form.answer_index < 0 || form.answer_index >= form.options.length) {
    ElMessage.warning('请选择正确答案')
    return
  }
  adminSaving.value = true
  try {
    const payload = {
      category: form.category,
      title: form.title.trim(),
      options: form.options.map((o) => o.trim()),
      answer_index: form.answer_index,
      explanation: form.explanation,
    }
    if (form.id) {
      await adminUpdateQuestion(form.id, payload)
      ElMessage.success('题目已更新，并已发布该题集新版本（进行中/已完成作答仍按原版本）')
    } else {
      await adminCreateQuestion(payload)
      ElMessage.success('题目已新增，并已发布该题集新版本')
    }
    editorVisible.value = false
    await loadAdminQuestions()
    // 刷新题集版本号。
    sets.value = await listQuizSets()
  } finally {
    adminSaving.value = false
  }
}

onMounted(async () => {
  // 测验需要记录个人成绩与错题，未登录先引导登录。
  if (!auth.token) {
    ElMessage.info('请先登录后参加测验')
    router.push({ path: '/login', query: { redirect: '/quiz' } })
    return
  }
  try {
    const [list] = await Promise.all([listQuizSets(), quizStore.fetchProgress(true)])
    sets.value = list
  } finally {
    loading.value = false
  }
})

async function start(set: QuizSet) {
  const attemptNo = genNo()
  try {
    // 响应中的 attempt_no 才是权威值：已存在进行中尝试时后端会幂等返回那一次。
    const attempt = await startAttempt(set.id, attemptNo)
    router.push({ path: `/quiz/attempt/${attempt.attempt_no}` })
  } catch {
    // 其它异常：刷新进度，让用户从卡片上的“继续作答”进入。
    await quizStore.fetchProgress(true)
  }
}

function resume(set: QuizSet) {
  const p = sp(set.id)
  if (p?.attempt_no) router.push(`/quiz/attempt/${p.attempt_no}`)
}

function reviewResult(set: QuizSet) {
  const p = sp(set.id)
  if (p?.attempt_no) router.push(`/quiz/attempt/${p.attempt_no}`)
}

// 重新作答：已完成后可以再开一次（新尝试号），但成绩仍只认每册首次完成结果。
function restart(set: QuizSet) {
  start(set)
}

function goReview() {
  router.push('/quiz/review')
}

// 本地尝试号生成，和 api 层一致（避免循环引用直接内联）。
function genNo(): string {
  const c = globalThis.crypto as Crypto | undefined
  if (c?.randomUUID) return c.randomUUID()
  return `attempt-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}
</script>

<style scoped>
.page { max-width: 1000px; margin: 0 auto; }
.summary { margin: 16px 0; }
.stat-card { text-align: center; margin-bottom: 12px; cursor: default; }
.stat-value { font-size: 28px; font-weight: 700; color: #2c6e49; }
.stat-value.clickable { color: #e6a23c; cursor: pointer; }
.stat-label { color: #888; font-size: 13px; margin-top: 4px; }
.set-card { margin-bottom: 16px; }
.set-head { display: flex; justify-content: space-between; align-items: center; }
.set-name { font-weight: 700; font-size: 16px; }
.set-desc { color: #666; min-height: 42px; margin: 8px 0; }
.set-meta { color: #999; font-size: 13px; }
.set-actions { margin-top: 10px; text-align: right; }
.review-entry { margin-top: 8px; }
.admin-entry { margin-top: 12px; text-align: right; }
.admin-cat { margin-bottom: 12px; }
.admin-add { margin: 0 0 12px 12px; }
.opt-row { display: flex; align-items: center; gap: 8px; width: 100%; }
.opt-input { flex: 1; }
</style>
