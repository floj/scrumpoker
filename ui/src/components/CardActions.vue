<template>
  <div class="text-center mb-3">
    <button v-if="!revealed" type="button" class="btn btn-lg btn-success w-50" @click="$emit('reveal')">Reveal</button>
    <button v-if="revealed" type="button" class="btn btn-lg btn-danger w-50" @click="$emit('reset')">Reset</button>
  </div>
</template>

<script lang="ts">
export default {
  name: 'CardActions',
  props: {
    revealed: {
      type: Boolean,
      required: true,
    },
  },
  emits: {
    reveal: () => true,
    reset: () => true,
  },
};
</script>

<style lang="css" scoped></style>
