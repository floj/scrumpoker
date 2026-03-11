import { ref, watch, computed, isRef, type Ref } from 'vue';

export function useLocalStorage(
  key: string | Ref<string> | (() => string),
  defaultValue: string = '',
): Ref<string> {
  const resolvedKey = computed(() => {
    if (typeof key === 'function') return key();
    if (isRef(key)) return key.value;
    return key;
  });

  const stored = localStorage.getItem(resolvedKey.value);
  const data = ref(stored ?? defaultValue);

  watch(resolvedKey, (newKey) => {
    const stored = localStorage.getItem(newKey);
    data.value = stored ?? defaultValue;
  });

  watch(data, (newVal) => {
    if (newVal === null || newVal === undefined) {
      localStorage.removeItem(resolvedKey.value);
      return;
    }
    localStorage.setItem(resolvedKey.value, newVal);
  });

  return data;
}
