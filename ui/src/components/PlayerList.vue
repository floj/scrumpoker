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

<script setup lang="ts">
import { computed } from 'vue';
import type { Player } from '@/types';

const props = defineProps<{
  players: Record<string, Player>;
  revealed: boolean;
}>();

const playersOrdered = computed(() => {
  const list = Object.values(props.players).slice();
  if (props.revealed) {
    list.sort((a, b) => {
      if (a.card === b.card) return 0;
      const valueA = parseInt(a.card, 10);
      const valueB = parseInt(b.card, 10);
      if (isNaN(valueA) && isNaN(valueB)) return 0;
      if (isNaN(valueA)) return 1;
      if (isNaN(valueB)) return -1;
      return valueA - valueB;
    });
  }
  return list;
});
</script>
