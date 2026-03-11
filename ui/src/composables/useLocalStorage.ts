import { ref, watch, type Ref } from 'vue';

export function useLocalStorage(key: string, defaultValue: string = ''): Ref<string> {
  const stored = localStorage.getItem(key);
  const data = ref(stored ?? defaultValue);

  watch(data, (newVal) => {
    if (newVal === null || newVal === undefined) {
      localStorage.removeItem(key);
      return;
    }
    localStorage.setItem(key, newVal);
  });

  return data;
}
