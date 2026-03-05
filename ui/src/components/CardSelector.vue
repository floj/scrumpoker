<template>
  <div class="vcard-container">
    <div
      v-for="card in cards"
      :key="card"
      class="vcard"
      :class="{ 'vcard-selected': selectedCard === card }"
      @click="$emit('vote', card)"
    >
      <div>{{ card }}</div>
    </div>
  </div>
</template>

<script lang="ts">
export default {
  name: 'CardSelector',
  props: {
    cards: {
      type: Array as () => Array<string>,
      required: true,
    },
    selectedCard: {
      type: String,
      required: false,
      default: '',
    },
  },
  emits: {
    vote: (card: string) => true,
  },
};
</script>

<style scoped>
.vcard {
  width: 5rem;
  height: 7rem;
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 2rem;
  border: 2px solid var(--bs-primary);
  border-radius: var(--bs-border-radius-lg);
  padding: 1rem;
  cursor: pointer;
  user-select: none;
  transition: transform 0.1s ease-in-out;
}

.vcard:hover {
  transform: translateY(-0.5rem);
}
.vcard-selected {
  background-color: var(--bs-primary);
  color: white;
  border-color: var(--bs-primary);
  transform: translateY(-0.5rem);
}
.vcard-container {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  justify-content: center;
  align-content: flex-start;
  gap: 1rem;
}
</style>
