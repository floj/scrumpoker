<script lang="ts">
import { apiBaseURL } from '@/utils/baseurl';

export default {
  name: 'HomeView',
  data() {
    return {};
  },
  methods: {
    async createNewRoom() {
      try {
        const response = await fetch(`${apiBaseURL()}/api/v1/rooms/`, {
          method: 'POST',
        });
        const data = await response.json();
        if (!response.ok) {
          this.$emit('show-toast', `Failed to create a new room: ${data.error}`, 'danger');
          return;
        }
        this.$router.push(`/rooms/${data.name}`);
      } catch (error) {
        this.$emit('show-toast', `Failed to create a new room`, 'danger');
        return;
      }
    },
  },
};
</script>

<template>
  <main>
    <h1>Scrum Poker</h1>
    <button type="button" class="btn btn-primary btn-lg" @click="createNewRoom">Create new room</button>
  </main>
</template>

<style scoped>
@media (min-width: 1024px) {
  main {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }
}
</style>
