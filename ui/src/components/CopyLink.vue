<template>
  <button class="btn btn-outline-secondary" @click="copyToClipboard">
    <i class="bi bi-clipboard"></i> Copy room link
  </button>
</template>

<script lang="ts" setup>
import { showToast } from '@/utils/toasts';

const props = defineProps<{ link: string }>();

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(props.link);
    showToast('Link copied', 'success', 1000);
  } catch {
    showToast('Failed to copy link');
  }
}
</script>
