<template>
  <div class="dropdown">
    <button class="btn btn-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false">
      <i
        class="bi"
        :class="{
          'bi-sun-fill': theme === 'light',
          'bi-moon-stars-fill': theme === 'dark',
          'bi-circle-half': theme === 'auto',
        }"
      ></i>
    </button>
    <ul class="dropdown-menu">
      <li>
        <a class="dropdown-item" :class="{ active: theme === 'light' }" href="#" @click.prevent="theme = 'light'">
          <i class="bi bi-sun-fill"></i> Light
        </a>
      </li>
      <li>
        <a class="dropdown-item" :class="{ active: theme === 'dark' }" href="#" @click.prevent="theme = 'dark'">
          <i class="bi bi-moon-stars-fill"></i> Dark
        </a>
      </li>
      <li>
        <a class="dropdown-item" :class="{ active: theme === 'auto' }" href="#" @click.prevent="theme = 'auto'">
          <i class="bi bi-circle-half"></i> Auto
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts">
export default {
  name: 'ThemeToggle',
  data() {
    return {
      theme: 'auto' as 'light' | 'dark' | 'auto',
    };
  },
  watch: {
    theme(newVal) {
      localStorage.setItem('theme', newVal);
      this.setTheme(this.getPreferredTheme());
    },
  },
  created() {
    this.setTheme(this.getPreferredTheme());
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', this.onSystemThemeChange);
  },
  beforeUnmount() {
    window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', this.onSystemThemeChange);
  },
  methods: {
    setTheme(theme: 'light' | 'dark') {
      document.documentElement.setAttribute('data-bs-theme', theme);
    },
    getPreferredTheme(): 'light' | 'dark' {
      switch (this.theme) {
        case 'light':
          return 'light';
        case 'dark':
          return 'dark';
        case 'auto':
        default:
          return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
      }
    },
    onSystemThemeChange(e: MediaQueryListEvent) {
      const newTheme = this.getPreferredTheme();
      if (newTheme != this.theme) {
        this.setTheme(newTheme);
      }
    },
  },
};
</script>

<style scoped></style>
