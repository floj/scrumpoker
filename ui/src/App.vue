<script lang="ts">
import ThemeToggle from './components/ThemeToggle.vue';

export default {
  name: 'App',
  components: {
    ThemeToggle,
  },
  data() {
    return {
      toastIds: 0,
      toasts: [] as Array<{ id: number; message: string; type: string }>,
    };
  },
  methods: {
    showToast(message: string, type: string) {
      const id = this.toastIds++;
      this.toasts.push({ id, message, type });
      setTimeout(() => {
        this.toasts = this.toasts.filter((toast) => toast.id !== id);
      }, 5000);
    },
  },
};
</script>

<template>
  <div class="container">
    <div class="theme-settings">
      <ThemeToggle></ThemeToggle>
    </div>
    <div>
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="alert d-flex align-items-center"
        :class="`alert-${toast.type}`"
        role="alert"
      >
        <div>{{ toast.message }}</div>
      </div>
    </div>
    <RouterView />
  </div>
</template>

<style scoped>
.theme-settings {
  position: fixed;
  top: 1rem;
  right: 1rem;
  z-index: 1050;
}
</style>
