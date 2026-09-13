<script setup>
const props = defineProps({
  def: { type: Object, required: true },
  modelValue: { type: [String, Boolean, null], default: null },
})

const emit = defineEmits(['update:modelValue'])

function onText(e) {
  const v = e.target.value
  emit('update:modelValue', v === '' ? null : v)
}

function onCheck(e) {
  emit('update:modelValue', e.target.checked ? '' : null)
}
</script>

<template>
  <div class="param-item">
    <div class="flag">{{ def.flag }}<span v-if="def.alias">, {{ def.alias }}</span></div>
    <div class="desc">{{ def.desc || '（无说明）' }}</div>
    <div v-if="def.hasValue">
      <input
        type="text"
        :placeholder="'输入 ' + def.flag + ' 的值'"
        :value="modelValue === null ? '' : modelValue"
        @input="onText"
      />
    </div>
    <div v-else class="checkbox-row">
      <input
        type="checkbox"
        :checked="modelValue !== null && modelValue !== undefined"
        @change="onCheck"
      />
      <span class="hint">启用</span>
    </div>
  </div>
</template>
