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

    <section v-if="auth.token && reviewProgress" class="review-banner" @click="$router.push('/quiz/review')">
      <el-card shadow="hover" class="review-card">
        <div class="review-info">
          <span class="review-title">📖 错题复习</span>
          <span>待复习 <b>{{ reviewProgress.pending }}</b> 题 · 已掌握 <b>{{ reviewProgress.resolved }}</b> 题</span>
        </div>
        <el-button type="success" size="small" plain>去复习</el-button>
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
import { onMounted, ref } from 'vue'
import axios from 'axios'
import { useRouter } from 'vue-router'
import PlantCard from '@/components/common/PlantCard.vue'
import CareArticleCard from '@/components/common/CareArticleCard.vue'
import { currentSeasonTask } from '@/utils/season'
import { useAuthStore } from '@/stores/authStore'
import { getQuizReviewProgress } from '@/api/quiz'
import type { PlantSpecies } from '@/constants/plant'
import type { CareArticle } from '@/constants/article'
import type { QuizReviewProgress } from '@/types/api'

const router = useRouter()
const auth = useAuthStore()
const keyword = ref('')
const seasonTask = ref(currentSeasonTask())
const hotPlants = ref<PlantSpecies[]>([])
const latestArticles = ref<CareArticle[]>([])
const reviewProgress = ref<QuizReviewProgress | null>(null)

onMounted(async () => {
  const res = await axios.get('/api/v1/home/overview')
  hotPlants.value = res.data.data.hot_plants || []
  latestArticles.value = res.data.data.latest_articles || []
  seasonTask.value = res.data.data.season_task || seasonTask.value
  // 首页与复习页读取同一进度结果
  if (auth.token) {
    try {
      reviewProgress.value = await getQuizReviewProgress()
    } catch {
      reviewProgress.value = null
    }
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
.review-banner { margin: 0 0 24px; cursor: pointer; }
.review-card :deep(.el-card__body) { display: flex; align-items: center; justify-content: space-between; }
.review-info { display: flex; gap: 16px; align-items: center; color: #555; }
.review-info b { color: #3c8d5c; }
.review-title { font-weight: 700; color: #2c6e49; }
h2 { color: #333; margin: 24px 0 16px; }
</style>
