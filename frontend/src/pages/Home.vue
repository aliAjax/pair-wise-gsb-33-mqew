<template>
  <div class="home">
    <section class="hero">
      <h1>植物养护知识百科平台</h1>
      <p class="sub">品种库 · 养护文章 · 病虫害防治 · 季节日历 · 问答社区</p>
      <el-input v-model="keyword" size="large" placeholder="搜索植物品种 / 养护文章 / 病虫害" class="search" @keyup.enter="doSearch">
        <template #append><el-button @click="doSearch">搜索</el-button></template>
      </el-input>
    </section>

    <section class="season-banner">
      <el-alert :title="`当季养护重点：${seasonTask}`" type="success" :closable="false" show-icon />
    </section>

    <section v-if="auth.isLoggedIn" class="quiz-banner">
      <el-card shadow="hover" :body-style="{ padding: '16px 20px' }">
        <div class="quiz-banner-inner">
          <div class="quiz-info">
            <div class="quiz-title">📝 养护知识小测验</div>
            <div class="quiz-meta" v-if="quizStore.progress">
              累计成绩 {{ quizStore.progress.total_score }} / {{ quizStore.progress.total_questions }}
              · 已完成 {{ doneSetCount }}/{{ quizStore.progress.sets.length }} 题集
              · <span :class="{ 'todo': quizStore.progress.review_count > 0 }">错题复习 {{ quizStore.progress.review_count }} 题</span>
            </div>
            <div class="quiz-meta" v-else>品种、文章、病虫害三大知识点题集，开始你的第一次作答。</div>
          </div>
          <div class="quiz-actions">
            <el-button v-if="(quizStore.progress?.review_count ?? 0) > 0" type="warning" @click="router.push('/quiz/review')">
              复习错题（{{ quizStore.progress?.review_count }}）
            </el-button>
            <el-button type="primary" @click="router.push('/quiz')">进入测验</el-button>
          </div>
        </div>
      </el-card>
    </section>

    <section>
      <h2>🔥 热门品种</h2>
      <el-row :gutter="16">
        <el-col v-for="p in hotPlants" :key="p.id" :xs="12" :sm="8" :md="6">
          <PlantCard :plant="p" />
        </el-col>
      </el-row>
    </section>

    <section>
      <h2>📖 最新养护文章</h2>
      <el-row :gutter="16">
        <el-col v-for="a in latestArticles" :key="a.id" :xs="12" :sm="8" :md="6">
          <CareArticleCard :article="a" />
        </el-col>
      </el-row>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import axios from 'axios'
import { useRouter } from 'vue-router'
import PlantCard from '@/components/common/PlantCard.vue'
import CareArticleCard from '@/components/common/CareArticleCard.vue'
import { currentSeasonTask } from '@/utils/season'
import { useAuthStore } from '@/stores/authStore'
import { useQuizStore } from '@/stores/quizStore'
import type { PlantSpecies } from '@/constants/plant'
import type { CareArticle } from '@/constants/article'

const router = useRouter()
const auth = useAuthStore()
const quizStore = useQuizStore()
const keyword = ref('')
const seasonTask = ref(currentSeasonTask())
const hotPlants = ref<PlantSpecies[]>([])
const latestArticles = ref<CareArticle[]>([])

const doneSetCount = computed(() => (quizStore.progress?.sets || []).filter((s) => s.status === 'done').length)

onMounted(async () => {
  const res = await axios.get('/api/v1/home/overview')
  hotPlants.value = res.data.data.hot_plants || []
  latestArticles.value = res.data.data.latest_articles || []
  seasonTask.value = res.data.data.season_task || seasonTask.value
  // 首页测验进度与复习页读取同一结果（共享 store）。
  if (auth.isLoggedIn) {
    quizStore.fetchProgress(true).catch(() => undefined)
  }
})

function doSearch() {
  if (keyword.value) {
    router.push({ path: '/plants', query: { keyword: keyword.value } })
  }
}
</script>

<style scoped>
.home { max-width: 1200px; margin: 0 auto; }
.hero { text-align: center; padding: 40px 0 20px; }
.hero h1 { color: #2c6e49; font-size: 32px; }
.sub { color: #888; }
.search { max-width: 560px; margin: 16px auto; }
.season-banner { margin: 8px 0 24px; }
.quiz-banner { margin: 8px 0 24px; }
.quiz-banner-inner { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.quiz-title { font-weight: 700; color: #2c6e49; font-size: 16px; }
.quiz-meta { color: #666; font-size: 13px; margin-top: 4px; }
.quiz-meta .todo { color: #e6a23c; font-weight: 600; }
.quiz-actions { display: flex; gap: 8px; flex-shrink: 0; }
h2 { color: #333; margin: 24px 0 16px; }
</style>
