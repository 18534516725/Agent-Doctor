import '@testing-library/jest-dom/vitest';

const values = new Map<string, string>();
const memoryStorage = {
  getItem: (key: string) => values.get(key) ?? null,
  setItem: (key: string, value: string) => values.set(key, String(value)),
  removeItem: (key: string) => values.delete(key),
  clear: () => values.clear(),
  key: (index: number) => [...values.keys()][index] ?? null,
  get length() { return values.size; },
};
Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: memoryStorage });
Object.defineProperty(window, 'localStorage', { configurable: true, value: memoryStorage });
