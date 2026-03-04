<script setup lang="ts">
import { RouterView } from 'vue-router';
import type { st } from 'vue-router/dist/index-DFCq6eJK.js';
</script>

<script lang="ts">
export default {
  name: 'App',
  data() {
    return {
      toasts: [] as Array<{ message: string; type: string }>,
    };
  },
  methods: {
    showToast(message: string, type: string) {
      console.log(`Toast message: ${message} (type: ${type})`);
    },
  },
};
</script>

<template>
  <div>
    <div v-for="(toast, index) in toasts" :key="index" class="`alert d-flex align-items-center`" :class="`alert-${toast.type}`" role="alert">
      <div>{{ toast.message }}</div>
    </div>
  </div>
  <RouterView />
</template>

<style scoped></style>
