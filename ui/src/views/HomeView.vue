<script lang="ts">
import { apiBaseURL } from '@/utils/baseurl';

import { roomService } from '@/services/roomService';

export default {
  name: 'HomeView',
  data() {
    return {};
  },
  methods: {
    async createNewRoom() {
      const room = await roomService.createNewRoom();
      this.$router.push(`/rooms/${room.name}`);
    },
  },
};
</script>

<template>
  <main class="container">
    <div class="text-center">
      <h1>Scrum Poker</h1>
      <button type="button" class="btn btn-primary btn-lg" @click="createNewRoom">Create new room</button>
    </div>
  </main>
</template>

<style scoped>
.container {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  justify-content: center;
  align-items: center;
  align-content: stretch;
  gap: 8px;
  height: 50vh;
}
</style>
