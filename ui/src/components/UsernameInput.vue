<template>
  <div class="input-group mb-3 w-50 mx-auto">
    <span class="input-group-text" id="basic-addon1">Username</span>
    <input
      v-model="newUsername"
      type="text"
      class="form-control"
      placeholder="Username"
      aria-label="Username"
      @keydown.enter="changeUsername"
    />
    <button class="btn btn-primary" type="button" @click="changeUsername" :disabled="username === newUsername">
      Change
    </button>
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
  if (newUsername.value === props.username) return;
  emit('updateUsername', newUsername.value);
}
</script>
