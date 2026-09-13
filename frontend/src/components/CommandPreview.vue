<script setup>
defineProps({
  command: { type: String, default: '' },
})
</script>

<template>
  <pre class="command-preview">{{ command || '（尚未生成命令）' }}</pre>
</template>
