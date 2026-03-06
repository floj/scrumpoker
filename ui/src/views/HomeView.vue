<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { roomService } from '@/services/roomService';
import { showToast } from '@/utils/toasts';

const router = useRouter();
const creatingRoom = ref(false);

async function createNewRoom() {
  creatingRoom.value = true;
  try {
    const room = await roomService.createNewRoom();
    router.push(`/rooms/${room.name}`);
  } catch {
    showToast('Failed to create a new room, please try again.');
  } finally {
    creatingRoom.value = false;
  }
}
</script>

<template>
  <div class="text-center">
    <img src="/favicon.ico" alt="Scrum Poker" class="mb-4 d-block mx-auto" />
    <button type="button" class="btn btn-primary btn-lg" @click="createNewRoom" :disabled="creatingRoom">
      Create new room
    </button>
  </div>
</template>
