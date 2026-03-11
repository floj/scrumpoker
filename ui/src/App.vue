<script setup lang="ts">
import Settings from './components/Settings.vue';
</script>

<template>
  <div class="app-container mt-3">
    <div class="container-sm" mb-3>
      <Settings />
      <h1 class="text-center">no-fuzz estimates</h1>
    </div>
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
