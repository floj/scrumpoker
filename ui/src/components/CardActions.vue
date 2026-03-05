<template>
  <div class="text-center mb-3">
    <button v-if="!revealed" type="button" class="btn btn-lg btn-success w-50" @click="revealCards">Reveal</button>
    <button v-if="revealed" type="button" class="btn btn-lg btn-danger w-50" @click="resetCards">Reset</button>
  </div>
</template>

<script lang="ts">
import { roomService } from '@/services/roomService';

export default {
  name: 'CardActions',
  props: {
    revealed: {
      type: Boolean,
      required: true,
    },
    roomName: {
      type: String,
      required: true,
    },
  },
  methods: {
    async revealCards() {
      await roomService.revealCards(this.roomName);
    },
    async resetCards() {
      await roomService.resetCards(this.roomName);
    },
  },
};
</script>

<style lang="css" scoped></style>
