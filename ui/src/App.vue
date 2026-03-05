<script setup lang="ts">
import { RouterView } from 'vue-router';
import ThemeToggle from './components/ThemeToggle.vue';
</script>

<script lang="ts">
import { apiBaseURL } from './utils/baseurl';

export default {
  name: 'App',
  components: {
    ThemeToggle,
  },
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
  <div class="container">
    <div class="theme-settings">
      <ThemeToggle></ThemeToggle>
    </div>
    <div>
      <div
        v-for="(toast, index) in toasts"
        :key="index"
        class="`alert d-flex align-items-center`"
        :class="`alert-${toast.type}`"
        role="alert"
      >
        <div>{{ toast.message }}</div>
      </div>
    </div>
    <RouterView @toast="showToast" />
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
