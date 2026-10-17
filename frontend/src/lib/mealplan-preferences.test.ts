import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  mealPlanKey,
  readPreference,
  useMealPlanPreference,
  writePreference,
} from "./mealplan-preferences";

const KEY = "mealcycle.mealplan.test";

function validateWeeks(raw: unknown): number | null {
  return raw === 1 || raw === 2 || raw === 4 ? raw : null;
}

beforeEach(() => {
  localStorage.clear();
});

// globals: false では Testing Library の自動 cleanup が登録されない (design.md D5)。
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("mealPlanKey", () => {
  it("should prefix key with mealcycle.mealplan. when called", () => {
    expect(mealPlanKey("weeks")).toBe("mealcycle.mealplan.weeks");
  });
});

describe("readPreference", () => {
  it("should return default value when key does not exist", () => {
    expect(readPreference(KEY, 1, validateWeeks)).toBe(1);
  });

  it("should return default value when stored value is not valid JSON", () => {
    localStorage.setItem(KEY, "{not json");
    expect(readPreference(KEY, 1, validateWeeks)).toBe(1);
  });

  it("should return default value when validate rejects the stored value", () => {
    localStorage.setItem(KEY, "3");
    expect(readPreference(KEY, 1, validateWeeks)).toBe(1);
  });

  it("should return stored value when validate accepts it", () => {
    localStorage.setItem(KEY, "4");
    expect(readPreference(KEY, 1, validateWeeks)).toBe(4);
  });
});

describe("writePreference", () => {
  it("should store value as JSON when writePreference is called", () => {
    writePreference(KEY, { weeks: 2 });
    expect(localStorage.getItem(KEY)).toBe('{"weeks":2}');
  });

  it("should not throw when localStorage.setItem throws", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("QuotaExceededError");
    });
    expect(() => writePreference(KEY, 2)).not.toThrow();
  });
});

describe("useMealPlanPreference", () => {
  it("should initialize hook state from stored value when useMealPlanPreference mounts", () => {
    localStorage.setItem(KEY, "2");
    const { result } = renderHook(() => useMealPlanPreference(KEY, 1, validateWeeks));
    expect(result.current[0]).toBe(2);
  });

  it("should persist value to localStorage when hook updater is called", () => {
    const { result } = renderHook(() => useMealPlanPreference(KEY, 1, validateWeeks));
    act(() => {
      result.current[1](4);
    });
    expect(result.current[0]).toBe(4);
    expect(localStorage.getItem(KEY)).toBe("4");
  });
});
