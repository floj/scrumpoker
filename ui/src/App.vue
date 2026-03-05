<script setup lang="ts">
import Settings from './components/Settings.vue';
</script>

<template>
  <div class="app-container mt-3">
    <Settings />
    <RouterView />
  </div>
</template>

<style scoped>
.app-container {
  max-width: 720px;
  margin-left: auto;
  margin-right: auto;
}
</style>
