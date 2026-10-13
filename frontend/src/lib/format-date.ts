// 表示用の日付表記。内部値 (search param / サービス引数) の YYYY-MM-DD は変えず、画面に出す直前だけ
// 環境の言語設定に左右されない yyyy/mm/dd にする (add-shopping-list design.md D15)。
export function formatDisplayDate(isoDate: string): string {
  return isoDate.replaceAll("-", "/");
}

// RFC3339 (UTC) のタイムスタンプを、PC の時刻で見た年月日 yyyy/mm/dd にする (fix-recipe-form-and-list design.md D4)。
export function formatTimestampDate(rfc3339: string): string {
  const d = new Date(rfc3339);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())}`;
}
