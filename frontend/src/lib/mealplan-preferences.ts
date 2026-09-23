import { useCallback, useState } from "react";

// S09 で ConfigService に移すときは、この hook の中身だけ差し替える (design.md D1)。
// key prefix は mealcycle.mealplan. に統一し、他 Wails アプリと origin を共有した場合の
// 衝突を防ぐ。呼び出し側で prefix をハードコードしないよう mealPlanKey で組み立てる。
const KEY_PREFIX = "mealcycle.mealplan.";

export function mealPlanKey(name: string): string {
  return `${KEY_PREFIX}${name}`;
}

export function readPreference<T>(
  key: string,
  defaultValue: T,
  validate: (raw: unknown) => T | null,
): T {
  try {
    const raw = window.localStorage.getItem(key);
    if (raw === null) {
      return defaultValue;
    }
    const parsed = JSON.parse(raw) as unknown;
    const valid = validate(parsed);
    return valid === null ? defaultValue : valid;
  } catch {
    return defaultValue;
  }
}

export function writePreference(key: string, value: unknown): void {
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // private mode / QuotaExceeded / SecurityError は握り潰す (design.md D4)。
  }
}

export function useMealPlanPreference<T>(
  key: string,
  defaultValue: T,
  validate: (raw: unknown) => T | null,
): [T, (next: T) => void] {
  const [value, setValue] = useState<T>(() => readPreference(key, defaultValue, validate));

  const update = useCallback(
    (next: T) => {
      setValue(next);
      writePreference(key, next);
    },
    [key],
  );

  return [value, update];
}
