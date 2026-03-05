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
      <tr v-for="[id, player] in playersOrdered" :key="id">
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

function revealedComparator([, a]: [string, Player], [, b]: [string, Player]) {
  const valueA = parseInt(a.card, 10);
  const valueB = parseInt(b.card, 10);
  if (a.card === b.card) {
    return a.name.localeCompare(b.name);
  }
  if (isNaN(valueA) && isNaN(valueB)) {
    return a.name.localeCompare(b.name);
  }
  if (isNaN(valueA)) {
    return 1;
  }
  if (isNaN(valueB)) {
    return -1;
  }
  return valueA - valueB;
}

function votedComparator([, a]: [string, Player], [, b]: [string, Player]) {
  if (a.voted === b.voted) {
    return a.name.localeCompare(b.name);
  }
  if (a.voted) {
    return -1;
  }
  return 1;
}

const playersOrdered = computed(() => {
  const entries = Object.entries(props.players);
  if (props.revealed) {
    entries.sort(revealedComparator);
  } else {
    entries.sort(votedComparator);
  }
  return entries;
});
</script>
