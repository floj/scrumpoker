<template>
  <div class="dropdown">
    <button
      class="btn btn-outline-secondary dropdown-toggle"
      type="button"
      data-bs-toggle="dropdown"
      aria-expanded="false"
    >
      <i
        class="bi"
        :class="{
          'bi-sun-fill': theme === 'light',
          'bi-moon-stars-fill': theme === 'dark',
          'bi-circle-half': theme === 'auto',
        }"
        aria-label="Toggle theme"
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

<script setup lang="ts">
import { watch, onMounted, onBeforeUnmount, type Ref } from 'vue';
import { useLocalStorage } from '@/composables/useLocalStorage';

const theme = useLocalStorage('theme', 'auto');

const darkMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

function setTheme(resolved: 'light' | 'dark') {
  document.documentElement.setAttribute('data-bs-theme', resolved);
}

function getPreferredTheme(): 'light' | 'dark' {
  if (theme.value === 'light') {
    return 'light';
  }
  if (theme.value === 'dark') {
    return 'dark';
  }
  // else auto, so we check the system preference
  return darkMediaQuery.matches ? 'dark' : 'light';
}

function onSystemThemeChange() {
  if (theme.value === 'auto') {
    setTheme(getPreferredTheme());
  }
}

watch(theme, () => {
  setTheme(getPreferredTheme());
});

onMounted(() => {
  setTheme(getPreferredTheme());
  darkMediaQuery.addEventListener('change', onSystemThemeChange);
});

onBeforeUnmount(() => {
  darkMediaQuery.removeEventListener('change', onSystemThemeChange);
});
</script>
