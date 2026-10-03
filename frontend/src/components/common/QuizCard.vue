<template>
  <el-card class="quiz-card">
    <div class="q-title">{{ index + 1 }}. {{ question.title }}</div>
    <el-radio-group :model-value="modelValue" :disabled="readonly" @update:model-value="onSelect">
      <el-radio v-for="(opt, i) in question.options" :key="i" :value="i" class="option">
        {{ opt }}
        <el-tag v-if="showResult && i === question.answer_index" type="success" size="small">正确答案</el-tag>
        <el-tag v-else-if="showResult && isWrongChoice(i)" type="danger" size="small">你的选择</el-tag>
      </el-radio>
    </el-radio-group>
    <el-alert
      v-if="showResult && question.selected === undefined"
      title="本题未作答，按答错处理"
      type="warning"
      :closable="false"
      class="unanswered"
    />
    <div v-if="showResult && question.explanation" class="explanation">解析：{{ question.explanation }}</div>
  </el-card>
</template>

<script setup lang="ts">
import type { AttemptQuestion } from '@/types/quiz'

const props = defineProps<{
  question: AttemptQuestion
  index: number
  modelValue?: number
  readonly: boolean
  showResult: boolean
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: number): void }>()

function onSelect(v: number) {
  if (!props.readonly) emit('update:modelValue', v)
}

// 交卷后：用户选择与正确答案不同时展示错误标记（未作答的提示由 el-alert 展示）。
const isWrongChoice = (i: number) =>
  props.question.selected !== undefined && props.question.selected === i && i !== props.question.answer_index
</script>

<style scoped>
.quiz-card { margin-bottom: 12px; }
.q-title { font-weight: 700; margin-bottom: 8px; }
.option { display: block; margin: 6px 0; }
.explanation { margin-top: 8px; color: #3c8d5c; }
.unanswered { margin-top: 8px; }
</style>
