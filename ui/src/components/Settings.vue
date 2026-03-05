<script lang="ts" setup>
import ThemeToggle from '@/components/ThemeToggle.vue';

import { useRoute } from 'vue-router';
import CopyLink from './CopyLink.vue';
import { computed } from 'vue';

const route = useRoute();

const isRoomView = computed(() => route.name === 'rooms');
const link = window.location.href;
</script>

<template>
  <div class="settings">
    <CopyLink v-if="isRoomView" :link="link"></CopyLink>
    <ThemeToggle></ThemeToggle>
  </div>
</template>

<style scoped>
.settings {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  justify-content: flex-end;
  align-items: center;
  align-content: stretch;
  gap: 8px;
}
</style>
