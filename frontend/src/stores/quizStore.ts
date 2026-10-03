import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getQuizProgress, listReview } from '@/api/quiz'
import type { QuizProgress, ReviewItem } from '@/types/quiz'

// 养护测验的共享状态：首页进度卡片与测验/复习页读取同一份结果，
// 避免两个页面各自请求导致口径不一致。
export const useQuizStore = defineStore('quiz', () => {
  const progress = ref<QuizProgress | null>(null)
  const reviewItems = ref<ReviewItem[]>([])
  const loaded = ref(false)
  const loading = ref(false)

  async function fetchProgress(force = false) {
    if (loading.value) return progress.value
    if (loaded.value && !force) return progress.value
    loading.value = true
    try {
      progress.value = await getQuizProgress()
      loaded.value = true
      return progress.value
    } finally {
      loading.value = false
    }
  }

  // 复习清单与进度一起刷新：review_count 与清单内容来自同一轮加载。
  async function fetchReview() {
    const [p, items] = await Promise.all([getQuizProgress(), listReview()])
    progress.value = p
    reviewItems.value = items
    loaded.value = true
    return items
  }

  function reset() {
    progress.value = null
    reviewItems.value = []
    loaded.value = false
  }

  return { progress, reviewItems, loaded, loading, fetchProgress, fetchReview, reset }
})
