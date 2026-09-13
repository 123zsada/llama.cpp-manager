<script setup>
defineProps({
  name: { type: String, required: true },
  count: { type: Number, default: 0 },
  active: { type: Boolean, default: false },
})
defineEmits(['select'])
</script>

<template>
  <div class="group-title" :class="{ active }" @click="$emit('select', name)">
    {{ name }} <span class="hint">({{ count }})</span>
  </div>
</template>
