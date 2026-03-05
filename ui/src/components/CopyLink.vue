<template>
  <div class="input-group fit-content">
    <input
      ref="input-link"
      type="text"
      name="roomlink"
      class="form-control"
      aria-label="Room link"
      readonly
      :value="link"
      @click="selectInput"
    />
    <button class="btn btn-outline-secondary" @click="copyToClipboard"><i class="bi bi-clipboard"></i></button>
  </div>
</template>

<script lang="ts" setup>
import { showToast } from '@/utils/toasts';
import { useTemplateRef } from 'vue';
const props = defineProps<{ link: string }>();

const inputLink = useTemplateRef('input-link');

function selectInput() {
  inputLink.value?.select();
}

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(props.link);
    showToast('Link copied', 'success', 1000);
  } catch (err) {
    showToast('Failed to copy link');
  }
}
</script>

<style scoped>
.fit-content {
  width: fit-content;
}
</style>
