<template>
  <table class="table table-striped fs-3">
    <thead>
      <tr>
        <th scope="col" class="text-start">Player</th>
        <th v-if="revealed" scope="col" class="text-end">Card</th>
        <th v-if="!revealed" scope="col" class="text-end">Voted</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="(player, id) in playersOrdered" :key="id">
        <td class="text-start">{{ player.name }}</td>
        <td class="text-end">
          <span v-if="revealed">{{ player.card }}</span>
          <i v-if="!revealed && player.voted" class="bi bi-check"></i>
        </td>
      </tr>
    </tbody>
  </table>
</template>

<script lang="ts">
import type { Player } from '@/types';

export default {
  name: 'PlayerList',
  props: {
    players: {
      type: Object as () => Record<string, Player>,
      required: true,
    },
    revealed: {
      type: Boolean,
      required: true,
    },
  },
  computed: {
    playersOrdered() {
      const players = [] as Array<{ name: string; card: string; voted: boolean }>;
      for (const player of Object.values(this.players)) {
        players.push(player);
      }
      if (this.revealed) {
        players.sort((a, b) => {
          if (a.card === b.card) {
            return 0;
          }
          if (a.card === '?' || a.card === ':coffee:') {
            return 1;
          }
          if (b.card === '?' || b.card === ':coffee:') {
            return -1;
          }
          return parseInt(a.card) - parseInt(b.card);
        });
      }
      return players;
    },
  },
};
</script>

<style scoped></style>
