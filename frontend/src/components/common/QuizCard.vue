<template>
  <el-card class="quiz-card" :class="{ wrong: submitted && !question.correct }">
    <div class="q-title">
      {{ index + 1 }}. {{ question.question }}
      <el-tag v-if="submitted" :type="question.correct ? 'success' : 'danger'" size="small">
        {{ question.correct ? '答对' : '答错' }}
      </el-tag>
    </div>
    <el-radio-group v-model="selected" :disabled="submitted">
      <el-radio v-for="(opt, i) in question.options" :key="i" :value="i" class="option">
        {{ opt }}
        <el-tag v-if="submitted && i === question.answer" type="success" size="small">正确答案</el-tag>
        <el-tag v-else-if="submitted && selected === i" type="danger" size="small">你的选择</el-tag>
      </el-radio>
    </el-radio-group>
    <div v-if="submitted && question.explanation" class="explanation">解析：{{ question.explanation }}</div>
  </el-card>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { QuizQuestionResult } from '@/types/api'

const props = defineProps<{ question: QuizQuestionResult; index: number; submitted: boolean; modelValue: number | undefined }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()
const selected = ref<number | undefined>(props.modelValue)
watch(() => props.modelValue, (v) => { selected.value = v })
watch(selected, (v) => { if (v !== undefined) emit('update:modelValue', v) })
</script>

<style scoped>
.quiz-card { margin-bottom: 12px; }
.quiz-card.wrong { border-color: #f56c6c; }
.q-title { font-weight: 700; margin-bottom: 8px; }
.option { display: block; margin: 6px 0; }
.explanation { margin-top: 8px; color: #3c8d5c; }
</style>
