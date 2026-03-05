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

<script lang="ts">
export default {
  name: 'UsernameInput',
  data() {
    return {
      newUsername: this.username,
    };
  },
  props: {
    username: {
      type: String,
      required: true,
    },
  },
  emits: {
    updateUsername: (newUsername: string) => true,
  },
  watch: {
    username(newUsername) {
      this.newUsername = newUsername;
    },
  },
  methods: {
    changeUsername() {
      if (this.newUsername === this.username) {
        return;
      }
      this.$emit('updateUsername', this.newUsername);
    },
  },
};
</script>
