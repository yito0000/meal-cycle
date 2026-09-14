// 献立カレンダー用の pure な日付ヘルパー。canonical な WeekStartOf は Go 側
// (internal/domain/mealplan) にあるが、mount ごとに Wails を往復させないため
// フロントに mirror を置き、7 日分の列ヘッダも生成する。

export function formatDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

export function weekStartOf(d: Date): Date {
  // getDay は Sun=0, Mon=1, .., Sat=6。月曜までの距離は (day + 6) % 7。
  const day = d.getDay();
  const offset = (day + 6) % 7;
  const monday = new Date(d.getFullYear(), d.getMonth(), d.getDate() - offset);
  return monday;
}

export function weekDates(weekStart: Date): Date[] {
  return periodDates(weekStart, 7);
}

// periodDates は periodStart から days 日ぶんの Date 列を返す。
// 1 週 = 7、2 週 = 14、4 週 = 28 で呼び出す。DST を跨いでも「同じ時刻の別日」を返せるよう、
// getFullYear / getMonth / getDate + オフセットで組み立てる。
export function periodDates(periodStart: Date, days: number): Date[] {
  return Array.from({ length: days }, (_, i) => {
    return new Date(periodStart.getFullYear(), periodStart.getMonth(), periodStart.getDate() + i);
  });
}

export function isSameLocalDate(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}
