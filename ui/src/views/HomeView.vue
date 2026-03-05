<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { roomService } from '@/services/roomService';
import ThemeToggle from '@/components/ThemeToggle.vue';

const router = useRouter();
const creatingRoom = ref(false);

async function createNewRoom() {
  creatingRoom.value = true;
  try {
    const room = await roomService.createNewRoom();
    router.push(`/rooms/${room.name}`);
  } finally {
    creatingRoom.value = false;
  }
}
</script>

<template>
  <main class="container">
    <div class="text-center">
      <h1>Scrum Poker</h1>
      <button type="button" class="btn btn-primary btn-lg" @click="createNewRoom" :disabled="creatingRoom">
        Create new room
      </button>
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
