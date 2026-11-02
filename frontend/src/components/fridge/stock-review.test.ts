import { describe, expect, it } from "vitest";
import { nextUnreviewedId } from "./stock-review";

describe("nextUnreviewedId", () => {
  it.each<{
    name: string;
    rowIds: string[];
    reviewed: string[];
    currentId: string;
    fromIndex?: number;
    want: string | null;
  }>([
    {
      name: "現在の行の次に未見直しの行があれば、その ID を返す",
      rowIds: ["a", "b", "c", "d"],
      reviewed: ["a", "c"],
      currentId: "b",
      want: "d",
    },
    {
      name: "現在の行より後がすべて見直し済みなら、先頭から探した未見直しの行の ID を返す",
      rowIds: ["a", "b", "c", "d"],
      reviewed: ["c", "d"],
      currentId: "b",
      want: "a",
    },
    {
      name: "すべて見直し済みなら null を返す",
      rowIds: ["a", "b", "c"],
      reviewed: ["a", "b", "c"],
      currentId: "b",
      want: null,
    },
    {
      name: "現在の行のほかがすべて見直し済みなら null を返す",
      rowIds: ["a", "b"],
      reviewed: ["a"],
      currentId: "b",
      want: null,
    },
    {
      name: "現在の行が一覧に無い (削除済み) とき、元の位置の次の未見直しの行を返す",
      rowIds: ["a", "c", "d"],
      reviewed: ["c"],
      currentId: "b",
      fromIndex: 1,
      want: "d",
    },
  ])("$name", ({ rowIds, reviewed, currentId, fromIndex, want }) => {
    expect(nextUnreviewedId(rowIds, new Set(reviewed), currentId, fromIndex)).toBe(want);
  });
});
