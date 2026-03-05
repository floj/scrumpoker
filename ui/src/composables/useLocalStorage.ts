import { ref, watch, type Ref } from 'vue';

export type Serializer<T> = {
  read: (raw: string) => T;
  write: (value: T) => string;
};

const identity: Serializer<string> = {
  read: (raw) => raw,
  write: (value) => value,
};

function jsonSerializer<T>(): Serializer<T> {
  return {
    read: (raw) => JSON.parse(raw) as T,
    write: (value) => JSON.stringify(value),
  };
}

export function useLocalStorage(key: string): Ref<string>;
export function useLocalStorage<T>(key: string, defaultValue: T, serializer?: Serializer<T>): Ref<T>;
export function useLocalStorage<T>(key: string, defaultValue: T = '' as T, serializer?: Serializer<T>): Ref<T> {
  const s =
    serializer ?? (typeof defaultValue === 'string' ? (identity as unknown as Serializer<T>) : jsonSerializer<T>());

  const stored = localStorage.getItem(key);
  const data = ref(stored != null ? s.read(stored) : defaultValue) as Ref<T>;

  watch(
    data,
    (newVal) => {
      if (newVal != null && newVal !== '') {
        localStorage.setItem(key, s.write(newVal));
      } else {
        localStorage.removeItem(key);
      }
    },
    { deep: true },
  );

  return data;
}
