// セルは期間の日数ぶん描かれるので、Intl.DateTimeFormat を言語ごとに 1 つだけ作って使い回す (fix-mealplan-dialog-polish design.md D1)。
const formatters = new Map<string, Intl.DateTimeFormat>();

// S01 の格子とダイアログで使う「月/日と曜日」の表記。形そのものは Intl に任せ、自前で組み立てない。
export function formatMealDate(isoDate: string, language: string): string {
  let fmt = formatters.get(language);
  if (!fmt) {
    fmt = new Intl.DateTimeFormat(language, { month: "numeric", day: "numeric", weekday: "short" });
    formatters.set(language, fmt);
  }
  const [y, m, d] = isoDate.split("-").map(Number);
  return fmt.format(new Date(y, m - 1, d));
}
