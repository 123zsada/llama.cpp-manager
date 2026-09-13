<script setup>
defineProps({
  explanations: { type: Array, default: () => [] },
})
</script>

<template>
  <div class="explainer">
    <div v-if="explanations.length === 0" class="hint">（暂无参数解释）</div>
    <div v-for="(e, idx) in explanations" :key="idx" class="explain-item">
      <code>{{ e.flag }} <span v-if="e.value">{{ e.value }}</span></code>
      <span class="desc">{{ e.desc }}</span>
      <span v-if="e.note" class="note">⚠ {{ e.note }}</span>
    </div>
  </div>
</template>
