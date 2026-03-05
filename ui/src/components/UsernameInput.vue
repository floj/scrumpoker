<template>
  <div class="input-group mb-3 mx-auto w-100 w-md-50">
    <label class="input-group-text" for="input-username">Username </label>
    <input
      id="input-username"
      v-model="newUsername"
      type="text"
      class="form-control"
      placeholder="Username"
      aria-label="Username"
      @keydown.enter="changeUsername"
    />
    <button class="btn btn-primary" type="button" @click="changeUsername">Change</button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

const props = defineProps<{ username: string }>();

const emit = defineEmits<{ updateUsername: [newUsername: string] }>();

const newUsername = ref(props.username);

watch(
  () => props.username,
  (val) => (newUsername.value = val),
);

function changeUsername() {
  if (newUsername.value === props.username) {
    return;
  }
  emit('updateUsername', newUsername.value);
}
</script>
