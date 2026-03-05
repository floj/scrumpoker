<script lang="ts">
import { RoomService } from '@/services/roomService';

export default {
  name: 'HomeView',
  data() {
    return {
      creatingRoom: false,
    };
  },
  props: {
    roomService: {
      type: RoomService,
      required: true,
    },
  },
  inject: ['roomService'],
  methods: {
    async createNewRoom() {
      this.creatingRoom = true;
      try {
        const room = await this.roomService.createNewRoom();
        this.$router.push(`/rooms/${room.name}`);
      } finally {
        this.creatingRoom = false;
      }
    },
  },
};
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
