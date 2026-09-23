// 表示用の日付表記。内部値 (search param / サービス引数) の YYYY-MM-DD は変えず、画面に出す直前だけ
// 環境の言語設定に左右されない yyyy/mm/dd にする (add-shopping-list design.md D15)。
export function formatDisplayDate(isoDate: string): string {
  return isoDate.replaceAll("-", "/");
}
