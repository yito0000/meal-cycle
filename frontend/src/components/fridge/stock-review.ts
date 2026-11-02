// 表示中の並び順で、currentId の次にある見直し済みでない行を返す。後ろに無ければ先頭から探す (add-fridge-stock-review design.md D4)。
// currentId が一覧に無い (「ない」で消した直後) ときは、元の位置 fromIndex にいま並んでいる行から探す。
export function nextUnreviewedId(
  rowIds: readonly string[],
  reviewed: ReadonlySet<string>,
  currentId: string,
  fromIndex?: number,
): string | null {
  const at = rowIds.indexOf(currentId);
  const start = at >= 0 ? at + 1 : Math.max(0, fromIndex ?? 0);
  const n = rowIds.length;
  for (let k = 0; k < n; k++) {
    const id = rowIds[(start + k) % n];
    if (id !== currentId && !reviewed.has(id)) return id;
  }
  return null;
}
