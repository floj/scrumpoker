<script setup lang="ts">
import { ref } from 'vue';
import ThemeToggle from './components/ThemeToggle.vue';

let toastIds = 0;
const toasts = ref<Array<{ id: number; message: string; type: string }>>([]);

function showToast(message: string, type: string) {
  const id = toastIds++;
  toasts.value.push({ id, message, type });
  setTimeout(() => {
    toasts.value = toasts.value.filter((toast) => toast.id !== id);
  }, 5000);
}
</script>

<template>
  <div class="container">
    <div class="theme-settings">
      <ThemeToggle></ThemeToggle>
    </div>
    <div>
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="alert d-flex align-items-center"
        :class="`alert-${toast.type}`"
        role="alert"
      >
        <div>{{ toast.message }}</div>
      </div>
    </div>
    <RouterView />
  </div>
</template>

<style scoped>
.theme-settings {
  position: fixed;
  top: 1rem;
  right: 1rem;
  z-index: 1050;
}
</style>
